package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/usecase"
)

func (h *Handler) chatStream(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	if accountID == "" {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Set("X-Accel-Buffering", "no")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	ch := h.Stream.Subscribe(accountID)
	r, w := io.Pipe()

	go func() {
		defer h.Stream.Unsubscribe(accountID, ch)
		defer w.Close()

		fmt.Fprintf(w, "retry: 3000\nevent: connected\ndata: {\"status\":\"ok\"}\n\n")
		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}
				data, _ := json.Marshal(event)
				if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", data); err != nil {
					return
				}
			case <-heartbeat.C:
				if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
					return
				}
			}
		}
	}()

	return c.SendStream(r)
}

func (h *Handler) resolveReplySet(ctx context.Context, accountID, replySetID, message string) (*domain.ReplySet, *domain.Product, error) {
	// 1. Explicit replySetID
	if replySetID != "" && h.Reply != nil {
		set, err := h.Reply.GetSet(ctx, replySetID, accountID)
		if err == nil && set != nil {
			var prod *domain.Product
			if h.Automation != nil && h.Product != nil {
				flows, _ := h.Automation.GetFlows(ctx, accountID)
				for _, f := range flows {
					if f.ReplySetID == set.ID && f.ProductID != "" {
						p, _ := h.Product.GetProduct(ctx, "", accountID, f.ProductID)
						if p != nil {
							prod = p
							break
						}
					}
				}
			}
			return set, prod, nil
		}
	}

	raw := strings.TrimSpace(message)
	if raw == "" {
		return nil, nil, nil
	}

	clean := strings.TrimSpace(strings.TrimLeft(raw, "/#:!# "))
	triggers := []string{raw}
	if clean != "" && clean != raw {
		triggers = append(triggers, clean)
	}

	// 2. Direct match by Reply Set name or code
	if h.Reply != nil {
		for _, trig := range triggers {
			set, err := h.Reply.FindSetByTrigger(ctx, accountID, trig)
			if err == nil && set != nil {
				var prod *domain.Product
				if h.Automation != nil && h.Product != nil {
					flows, _ := h.Automation.GetFlows(ctx, accountID)
					for _, f := range flows {
						if f.ReplySetID == set.ID && f.ProductID != "" {
							p, _ := h.Product.GetProduct(ctx, "", accountID, f.ProductID)
							if p != nil {
								prod = p
								break
							}
						}
					}
				}
				return set, prod, nil
			}
		}
	}

	// 3. Match by Product code / name / ID
	if h.Product != nil {
		products, prodErr := h.Product.ListProducts(ctx, "", accountID)
		if prodErr == nil {
			var flows []domain.AutomationFlow
			if h.Automation != nil {
				flows, _ = h.Automation.GetFlows(ctx, accountID)
			}
			for _, trig := range triggers {
				trigLower := strings.ToLower(trig)
				cleanTrig := strings.TrimSpace(strings.TrimLeft(trigLower, "/#:!# "))
				for _, prod := range products {
					cleanProdCode := strings.TrimSpace(strings.TrimLeft(strings.ToLower(prod.Code), "/#:!# "))
					codeMatch := (prod.Code != "" && strings.EqualFold(prod.Code, trigLower)) || (cleanTrig != "" && cleanProdCode != "" && cleanTrig == cleanProdCode)
					nameMatch := strings.EqualFold(prod.Name, trigLower) || strings.EqualFold(prod.Name, cleanTrig)
					idMatch := strings.EqualFold(prod.ID, trigLower) || strings.EqualFold(prod.ID, cleanTrig)
					if codeMatch || nameMatch || idMatch {
						// 3a. If product is linked to an active automation flow
						for _, f := range flows {
							if f.IsActive && f.ProductID == prod.ID && f.ReplySetID != "" {
								if h.Reply != nil {
									set, err := h.Reply.GetSet(ctx, f.ReplySetID, accountID)
									if err == nil && set != nil {
										pCopy := prod
										return set, &pCopy, nil
									}
								}
							}
						}

						// 3b. Product matched directly without linked flow: synthesize a Reply Set on the fly
						pCopy := prod
						synthSet := &domain.ReplySet{
							ID:        "synth-product:" + prod.ID,
							AccountID: accountID,
							Name:      prod.Name,
							Code:      prod.Code,
						}
						return synthSet, &pCopy, nil
					}
				}
			}
		}
	}

	return nil, nil, nil
}

func (h *Handler) resolvePageID(ctx context.Context, accountID, recipientID string) string {
	if recipientID != "" && h.Leads != nil {
		if lead, err := h.Leads.GetLeadBySender(ctx, accountID, recipientID); err == nil && lead != nil && lead.PageID != "" {
			return lead.PageID
		}
	}
	if h.meta != nil {
		if pages, err := h.meta.ListPages(ctx, "", accountID); err == nil {
			for _, p := range pages {
				if p.Connected && p.ID != "" {
					return p.ID
				}
			}
			if len(pages) > 0 {
				return pages[0].ID
			}
		}
	}
	return ""
}

func (h *Handler) sendChatMessage(c fiber.Ctx) error {
	var body struct {
		PageID      string `json:"page_id"`
		RecipientID string `json:"recipient_id"`
		Message     string `json:"message"`
		ReplySetID string `json:"reply_set_id"`
		MediaURL    string `json:"media_url"`
		MediaType   string `json:"media_type"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "invalid_request", err.Error())
	}
	accountID := c.Locals("accountID").(string)

	if strings.TrimSpace(body.PageID) == "" {
		body.PageID = h.resolvePageID(c.Context(), accountID, body.RecipientID)
	}

	// Check if admin entered a code matching a Reply Set or requested a specific set
	matchedSet, matchedProd, _ := h.resolveReplySet(c.Context(), accountID, body.ReplySetID, body.Message)
	if matchedSet != nil {
		var activeItems []domain.ReplyItem
		if strings.HasPrefix(matchedSet.ID, "synth-product:") && matchedProd != nil {
			// Synthesize product sequence: Image first (if available) + Product details and price
			if matchedProd.ImageUrl != "" {
				activeItems = append(activeItems, domain.ReplyItem{
					ID:         "synth-img-" + matchedProd.ID,
					ReplySetID: matchedSet.ID,
					Type:       "image",
					Content:    matchedProd.ImageUrl,
					OrderIndex: 1,
					IsEnabled:  true,
				})
			}
			productText := fmt.Sprintf("%s\n\n%s\n\nລາຄາ: %s", matchedProd.Name, matchedProd.Description, matchedProd.Price)
			activeItems = append(activeItems, domain.ReplyItem{
				ID:         "synth-txt-" + matchedProd.ID,
				ReplySetID: matchedSet.ID,
				Type:       "text",
				Content:    productText,
				OrderIndex: 2,
				IsEnabled:  true,
			})
		} else if h.Reply != nil {
			items, err := h.Reply.GetReplySetItems(c.Context(), matchedSet.ID, accountID)
			if err == nil && len(items) > 0 {
				for _, item := range items {
					if item.IsEnabled && strings.TrimSpace(item.Content) != "" {
						activeItems = append(activeItems, item)
					}
				}
				sort.Slice(activeItems, func(i, j int) bool {
					return activeItems[i].OrderIndex < activeItems[j].OrderIndex
				})

				if matchedProd != nil {
					for i := range activeItems {
						if activeItems[i].Type == "text" {
							activeItems[i].Content = strings.ReplaceAll(activeItems[i].Content, "{{product.name}}", matchedProd.Name)
							activeItems[i].Content = strings.ReplaceAll(activeItems[i].Content, "{{product.price}}", matchedProd.Price)
							activeItems[i].Content = strings.ReplaceAll(activeItems[i].Content, "{{product.description}}", matchedProd.Description)
						}
					}
				}
			}
		}

		if len(activeItems) > 0 {
			// Deliver the sequence asynchronously with realistic delays and typing indicators
			go func(accID, pageID, recipientID string, replyItems []domain.ReplyItem, setName string) {
				sendCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
				defer cancel()

				_ = h.meta.SendSenderAction(sendCtx, pageID, recipientID, "mark_seen")
				time.Sleep(500 * time.Millisecond)

				for index, step := range replyItems {
					if step.Type == "image" {
						time.Sleep(1000 * time.Millisecond)
					} else if step.Type == "text" {
						if index > 0 {
							time.Sleep(800 * time.Millisecond)
						}
						_ = h.meta.SendSenderAction(sendCtx, pageID, recipientID, "typing_on")
						runeCount := len([]rune(step.Content))
						delayMs := 1200 + (runeCount * 15)
						if delayMs < 1500 {
							delayMs = 1500
						} else if delayMs > 2800 {
							delayMs = 2800
						}
						time.Sleep(time.Duration(delayMs) * time.Millisecond)
					} else {
						time.Sleep(1200 * time.Millisecond)
					}

					var sendErr error
					if step.Type == "text" {
						sendErr = h.meta.SendMessage(sendCtx, accID, pageID, recipientID, step.Content)
					} else {
						sendErr = h.meta.SendMedia(sendCtx, accID, pageID, recipientID, step.Type, step.Content)
					}

					if sendErr != nil {
						slog.Error("failed to send reply set step from admin", "page_id", pageID, "recipient_id", recipientID, "step", index+1, "type", step.Type, "error", sendErr)
						continue
					}

					if h.Stream != nil {
						h.Stream.Broadcast(accID, usecase.ChatEvent{
							AccountID: accID,
							PageID:    pageID,
							SenderID:  recipientID,
							Message:   step.Content,
							Type:      step.Type,
							Timestamp: fmt.Sprintf("%d", time.Now().UnixMilli()),
							Platform:  "system",
						})
					}
				}
			}(accountID, body.PageID, body.RecipientID, activeItems, matchedSet.Name)

			return c.JSON(fiber.Map{
				"status":         "success",
				"reply_set_sent": true,
				"reply_set_name": matchedSet.Name,
				"items_count":    len(activeItems),
			})
		}
	}

	if strings.TrimSpace(body.MediaURL) != "" {
		mType := strings.ToLower(strings.TrimSpace(body.MediaType))
		if mType == "voice" || mType == "void" {
			mType = "audio"
		} else if mType == "vedio" {
			mType = "video"
		} else if mType == "" {
			mType = "image"
		}
		if err := h.meta.SendMedia(c.Context(), accountID, body.PageID, body.RecipientID, mType, body.MediaURL); err != nil {
			return fail(c, fiber.StatusInternalServerError, "send_failed", err.Error())
		}
		if h.Stream != nil {
			h.Stream.Broadcast(accountID, usecase.ChatEvent{
				AccountID: accountID,
				PageID:    body.PageID,
				SenderID:  body.RecipientID,
				Message:   body.MediaURL,
				Type:      mType,
				Timestamp: fmt.Sprintf("%d", time.Now().UnixMilli()),
				Platform:  "system",
			})
		}
		return c.JSON(fiber.Map{"status": "success"})
	}

	if err := h.meta.SendMessage(c.Context(), accountID, body.PageID, body.RecipientID, body.Message); err != nil {
		return fail(c, fiber.StatusInternalServerError, "send_failed", err.Error())
	}

	// Also broadcast the message back so it shows up in the UI immediately
	if h.Stream != nil {
		h.Stream.Broadcast(accountID, usecase.ChatEvent{
			AccountID: accountID,
			PageID:    body.PageID,
			SenderID:  body.RecipientID,
			Message:   body.Message,
			Type:      "text",
			Timestamp: fmt.Sprintf("%d", time.Now().UnixMilli()),
			Platform:  "system",
		})
	}

	return c.JSON(fiber.Map{"status": "success"})
}
