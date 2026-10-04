package httpx

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) chatStream(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	if accountID == "" {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	ch := h.Stream.Subscribe(accountID)
	r, w := io.Pipe()

	go func() {
		defer h.Stream.Unsubscribe(accountID, ch)
		defer w.Close()

		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\"}\n\n")

		for {
			event, ok := <-ch
			if !ok {
				break
			}

			data, _ := json.Marshal(event)
			_, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
			if err != nil {
				break
			}
		}
	}()

	return c.SendStream(r)
}
