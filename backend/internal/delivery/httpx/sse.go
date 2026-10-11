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

	// 3. Match by Product code / name / ID linked to an active flow
	if h.Product != nil && h.Automation != nil && h.Reply != nil {
		products, prodErr := h.Product.ListProducts(ctx, "", accountID)
		if prodErr == nil {
			flows, _ := h.Automation.GetFlows(ctx, accountID)
			for _, trig := range triggers {
				trigLower := strings.ToLower(trig)
				for _, prod := range products {
					codeMatch := prod.Code != "" && strings.EqualFold(prod.Code, trigLower)
					nameMatch := strings.EqualFold(prod.Name, trigLower)
					idMatch := strings.EqualFold(prod.ID, trigLower)
					if codeMatch || nameMatch || idMatch {
						for _, f := range flows {
							if f.IsActive && f.ProductID == prod.ID && f.ReplySetID != "" {
								set, err := h.Reply.GetSet(ctx, f.ReplySetID, accountID)
								if err == nil && set != nil {
									pCopy := prod
									return set, &pCopy, nil
								}
							}
						}
					}
				}
			}
		}
	}

	return nil, nil, nil
}

func (h *Handler) sendChatMessage(c fiber.Ctx) error {
	var body struct {
		PageID      string `json:"page_id"`
		RecipientID string `json:"recipient_id"`
		Message     string `json:"message"`
		ReplySetID string `json:"reply_set_id"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "invalid_request", err.Error())
	}
	accountID := c.Locals("accountID").(string)

	// Check if admin entered a code matching a Reply Set or requested a specific set
	matchedSet, matchedProd, _ := h.resolveReplySet(c.Context(), accountID, body.ReplySetID, body.Message)
	if matchedSet != nil && h.Reply != nil {
		items, err := h.Reply.GetReplySetItems(c.Context(), matchedSet.ID, accountID)
		if err == nil && len(items) > 0 {
			var activeItems []domain.ReplyItem
			for _, item := range items {
				if item.IsEnabled && strings.TrimSpace(item.Content) != "" {
					activeItems = append(activeItems, item)
				}
			}
			sort.Slice(activeItems, func(i, j int) bool {
				return activeItems[i].OrderIndex < activeItems[j].OrderIndex
			})

			if len(activeItems) > 0 {
				if matchedProd != nil {
					for i := range activeItems {
						if activeItems[i].Type == "text" {
							activeItems[i].Content = strings.ReplaceAll(activeItems[i].Content, "{{product.name}}", matchedProd.Name)
							activeItems[i].Content = strings.ReplaceAll(activeItems[i].Content, "{{product.price}}", matchedProd.Price)
							activeItems[i].Content = strings.ReplaceAll(activeItems[i].Content, "{{product.description}}", matchedProd.Description)
						}
					}
				}

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
