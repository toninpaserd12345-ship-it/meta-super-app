package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/gofiber/fiber/v3"
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

func (h *Handler) sendChatMessage(c fiber.Ctx) error {
	var body struct {
		PageID      string `json:"page_id"`
		RecipientID string `json:"recipient_id"`
		Message     string `json:"message"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "invalid_request", err.Error())
	}
	accountID := c.Locals("accountID").(string)

	if err := h.meta.SendMessage(c.Context(), accountID, body.PageID, body.RecipientID, body.Message); err != nil {
		return fail(c, fiber.StatusInternalServerError, "send_failed", err.Error())
	}

	// Also broadcast the message back so it shows up in the UI immediately
	if h.Stream != nil {
		h.Stream.Broadcast(accountID, usecase.ChatEvent{
			AccountID: accountID,
			PageID:    body.PageID,
			SenderID:  body.RecipientID, // In the UI, SenderID is used to group the chat. Even though WE are the sender, group it by recipient.
			Message:   body.Message,
			Type:      "text",
			Timestamp: "now",
			Platform:  "system", // indicating we sent it
		})
	}

	return c.JSON(fiber.Map{"status": "success"})
}
