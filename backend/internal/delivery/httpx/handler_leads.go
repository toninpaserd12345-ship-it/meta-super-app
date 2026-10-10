package httpx

import (
	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/usecase"
)

func (h *Handler) listLeads(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.Leads.List(ctx, c.Locals("accountID").(string))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) leadSettings(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	settings, steps, err := h.Leads.Settings(ctx, c.Locals("accountID").(string))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"settings": settings, "steps": steps, "capiConfigured": settings.CAPIToken != ""})
}
func (h *Handler) saveLeadSettings(c fiber.Ctx) error {
	var body struct {
		Settings usecase.LeadSettingsInput   `json:"settings"`
		Steps    []usecase.FollowUpStepInput `json:"steps"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, 400, "invalid_request", "Lead settings are invalid.")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	if err := h.Leads.SaveSettings(ctx, c.Locals("accountID").(string), body.Settings, body.Steps); err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"saved": true})
}
func (h *Handler) markLeadRead(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	if err := h.Leads.MarkRead(ctx, c.Locals("accountID").(string), c.Params("id")); err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"updated": true})
}

func (h *Handler) updateLeadStatus(c fiber.Ctx) error {
	var body struct {
		Status string `json:"status"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "invalid_request", err.Error())
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	// id is actually sender_id
	if err := h.Leads.UpdateStatus(ctx, c.Locals("accountID").(string), c.Params("id"), body.Status); err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"status": "success"})
}
