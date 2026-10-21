package httpx

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/usecase"
)

func (h *Handler) notificationPreferences(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.Notifications.List(ctx, c.Locals("accountID").(string), userID(c))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) saveNotificationPreferences(c fiber.Ctx) error {
	var body struct {
		Items []usecase.NotificationPreference `json:"items"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "invalid_request", "Notification preferences are invalid.")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.Notifications.Save(ctx, c.Locals("accountID").(string), userID(c), body.Items)
	if errors.Is(err, usecase.ErrInvalidNotificationPreference) {
		return fail(c, fiber.StatusBadRequest, "invalid_request", "Notification type or frequency is invalid.")
	}
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}
