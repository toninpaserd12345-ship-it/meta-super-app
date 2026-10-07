package httpx

import (
	"github.com/gofiber/fiber/v3"
)

func (h *Handler) getChatHistory(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)

	if h.Stream == nil {
		return c.JSON([]interface{}{})
	}

	history, err := h.Stream.GetHistory(c.Context(), accountID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "db_error", err.Error())
	}

	return c.JSON(history)
}
