package httpx

import (
	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/usecase"
)

func (h *Handler) teamList(c fiber.Ctx) error {
	members, err := h.team.ListMembers(c.Context(), c.Locals("accountID").(string))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(members)
}

func (h *Handler) teamInvite(c fiber.Ctx) error {
	var body usecase.InviteInput
	if err := c.Bind().JSON(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}

	if body.Email == "" || body.Role == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and role are required")
	}

	out, err := h.team.InviteMember(c.Context(), c.Locals("accountID").(string), body)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(out)
}

func (h *Handler) teamUpdateRole(c fiber.Ctx) error {
	userID := c.Params("userID")
	var body struct {
		Role domain.Role `json:"role"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Role == "" {
		return fiber.NewError(fiber.StatusBadRequest, "invalid role")
	}

	if err := h.team.UpdateRole(c.Context(), c.Locals("accountID").(string), userID, body.Role); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) teamRemove(c fiber.Ctx) error {
	userID := c.Params("userID")
	if err := h.team.RemoveMember(c.Context(), c.Locals("accountID").(string), userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) billingPlans(c fiber.Ctx) error {
	plans, err := h.billing.ListPlans(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(plans)
}

func (h *Handler) billingSubscription(c fiber.Ctx) error {
	sub, err := h.billing.GetSubscription(c.Context(), c.Locals("accountID").(string))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(sub)
}

func (h *Handler) billingCheckout(c fiber.Ctx) error {
	var body usecase.CheckoutInput
	if err := c.Bind().JSON(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}
	if body.PlanID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "planId is required")
	}

	if err := h.billing.Checkout(c.Context(), c.Locals("accountID").(string), body); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
