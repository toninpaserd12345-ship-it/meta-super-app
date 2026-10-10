package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/usecase"
)

// SendWhatsAppMessage sends a text message via WhatsApp Business API
func (g *GraphConnector) sendWhatsAppMessage(ctx context.Context, phoneNumberID, accessToken, recipientPhone, text string) error {
	payload, err := json.Marshal(map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                recipientPhone,
		"type":              "text",
		"text": map[string]string{
			"body": text,
		},
	})
	if err != nil {
		return err
	}
	return g.executeWhatsAppRequest(ctx, phoneNumberID, accessToken, payload)
}

// SendWhatsAppMedia sends a media message (image/video/audio) via WhatsApp Business API
func (g *GraphConnector) sendWhatsAppMedia(ctx context.Context, phoneNumberID, accessToken, recipientPhone, mediaType, mediaURL string) error {
	// WhatsApp supports: image, video, audio, document.
	waType := mediaType
	if waType == "audio" {
		waType = "audio" // same
	} else if waType == "video" {
		waType = "video"
	} else {
		waType = "image"
	}

	payload, err := json.Marshal(map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                recipientPhone,
		"type":              waType,
		waType: map[string]string{
			"link": mediaURL,
		},
	})
	if err != nil {
		return err
	}
	return g.executeWhatsAppRequest(ctx, phoneNumberID, accessToken, payload)
}

func (g *GraphConnector) executeWhatsAppRequest(ctx context.Context, phoneNumberID, accessToken string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.facebook.com/"+g.cfg.Version+"/"+phoneNumberID+"/messages", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	var envelope struct {
		Error *graphError `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	if resp.StatusCode >= 400 || envelope.Error != nil {
		return graphResponseError(resp.StatusCode, envelope.Error)
	}
	return nil
}

// WhatsApp Webhook Payload Structures
type WhatsAppWebhookEvent struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaID string `json:"wa_id"`
				} `json:"contacts"`
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text"`
					Type string `json:"type"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// processWhatsAppWebhook is called from ReceiveWebhook when object == "whatsapp_business_account"
func (g *GraphConnector) processWhatsAppWebhook(ctx context.Context, body []byte) error {
	var event WhatsAppWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("invalid whatsapp webhook payload: %w", err)
	}

	for _, entry := range event.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" || change.Value.MessagingProduct != "whatsapp" {
				continue
			}

			phoneNumberID := change.Value.Metadata.PhoneNumberID
			accountID, page, found := g.findPage(phoneNumberID)
			if !found || !strings.EqualFold(page.Category, "WhatsApp") || !page.Connected {
				slog.Warn("WhatsApp webhook ignored for an unknown or disabled phone", "phone_number_id", phoneNumberID)
				continue
			}
			for _, message := range change.Value.Messages {
				senderPhone := message.From
				textBody := strings.TrimSpace(message.Text.Body)

				if senderPhone == "" || textBody == "" {
					continue
				}
				var directSet *domain.ReplySet
				if g.automation != nil {
					directSet, _ = g.automation.FindReplySetByTrigger(ctx, accountID, textBody)
				}
				seenKey := "whatsapp:" + phoneNumberID + ":" + senderPhone
				if directSet != nil {
					seenKey += ":reply-set-name:" + directSet.ID
				}
				g.mu.Lock()
				lastReply, alreadyReplied := g.firstReplies[seenKey]
				busy := g.inFlight[seenKey]
				cooldown := 24 * time.Hour
				if directSet != nil {
					cooldown = 2 * time.Second
				}
				if !alreadyReplied || time.Since(lastReply) >= cooldown {
					g.inFlight[seenKey] = true
				} else {
					busy = true
				}
				g.mu.Unlock()

				slog.Info("WhatsApp message received", "phone_number_id", phoneNumberID, "sender", senderPhone, "text", textBody)

				if g.chatStream != nil {
					g.chatStream.Broadcast(accountID, usecase.ChatEvent{
						AccountID: accountID,
						PageID:    phoneNumberID,
						SenderID:  senderPhone,
						Message:   textBody,
						Type:      "text",
						Timestamp: message.Timestamp,
						Platform:  "whatsapp",
					})
				}

				if g.automation != nil && !busy && directSet != nil {
					sent, sendErr := g.sendWhatsAppReplySet(ctx, accountID, phoneNumberID, page.AccessToken, senderPhone, directSet.ID, nil)
					if sendErr != nil {
						slog.Error("send WhatsApp Reply Set by name", "reply_set_id", directSet.ID, "error", sendErr)
					}
					if sent {
						g.mu.Lock()
						g.firstReplies[seenKey] = time.Now()
						g.mu.Unlock()
					}
				} else if g.automation != nil && !busy {
					keywordRules, err := g.automation.GetKeywordRules(ctx, phoneNumberID)
					if err == nil {
						sent := false
						for _, kr := range keywordRules {
							if matchesKeywords(textBody, []string{kr.TriggerValue}) {
								slog.Info("WhatsApp automation triggered!", "rule_id", kr.ID)
								sent, err = g.sendWhatsAppReplySet(ctx, accountID, phoneNumberID, page.AccessToken, senderPhone, kr.ReplySetID, &kr)
								if err != nil {
									slog.Error("send WhatsApp automatic reply", "rule_id", kr.ID, "error", err)
								}
								if sent {
									g.mu.Lock()
									g.firstReplies[seenKey] = time.Now()
									g.mu.Unlock()
								}
								break
							}
						}
					}
				}
				g.mu.Lock()
				delete(g.inFlight, seenKey)
				g.mu.Unlock()
			}
		}
	}
	return nil
}

func (g *GraphConnector) sendWhatsAppReplySet(ctx context.Context, accountID, phoneNumberID, accessToken, recipientID, replySetID string, rule *domain.AutomationRule) (bool, error) {
	items, err := g.automation.GetReplySetItems(ctx, replySetID, accountID)
	if err != nil {
		return false, err
	}
	sent := false
	for _, item := range g.resolveReplyItems(rule, items) {
		if !item.IsEnabled {
			continue
		}
		if sent {
			time.Sleep(1500 * time.Millisecond)
		}
		if item.Type == "text" {
			err = g.sendWhatsAppMessage(ctx, phoneNumberID, accessToken, recipientID, item.Content)
		} else {
			err = g.sendWhatsAppMedia(ctx, phoneNumberID, accessToken, recipientID, item.Type, item.Content)
		}
		if err != nil {
			return sent, err
		}
		sent = true
	}
	return sent, nil
}
