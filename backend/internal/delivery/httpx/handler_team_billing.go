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
	targetUserID := c.Params("userID")
	callerID := userID(c)
	if targetUserID == callerID {
		return fail(c, fiber.StatusBadRequest, "cannot_remove_self", "You cannot remove yourself from the workspace.")
	}

	accountID := c.Locals("accountID").(string)
	mem, err := h.auth.Membership(c.Context(), targetUserID, accountID)
	if err == nil && mem != nil && mem.Role == domain.RoleOwner {
		return fail(c, fiber.StatusBadRequest, "cannot_remove_owner", "You cannot remove the owner of the workspace.")
	}

	if err := h.team.RemoveMember(c.Context(), accountID, targetUserID); err != nil {
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

func (h *Handler) billingStartTrial(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	ctx, cancel := requestContext(c)
	defer cancel()

	sub, err := h.billing.StartTrial(ctx, accountID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok", "subscription": sub})
}

func (h *Handler) billingSelectFree(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	ctx, cancel := requestContext(c)
	defer cancel()

	sub, err := h.billing.SelectFree(ctx, accountID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok", "subscription": sub})
}

func (h *Handler) onboardingStatus(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	ctx, cancel := requestContext(c)
	defer cancel()

	sub, err := h.billing.GetSubscription(ctx, accountID)
	hasPlan := (err == nil && sub != nil && (sub.Status == "active" || sub.Status == "trialing"))

	pages, _ := h.meta.ListPages(ctx, userID(c), accountID)
	connectedCount := 0
	var connectedPages []fiber.Map
	for _, p := range pages {
		if p.Connected {
			connectedCount++
			connectedPages = append(connectedPages, fiber.Map{
				"id":         p.ID,
				"name":       p.Name,
				"category":   p.Category,
				"pictureUrl": p.PictureURL,
			})
		}
	}

	step := "completed"
	if !hasPlan {
		step = "plan"
	} else if connectedCount == 0 {
		step = "page"
	}

	var planInfo fiber.Map
	if sub != nil && sub.Plan != nil {
		lockedPrice := sub.LockedPrice
		if lockedPrice == 0 {
			lockedPrice = sub.Plan.Price
		}
		planInfo = fiber.Map{
			"id":               sub.PlanID,
			"name":             sub.Plan.Name,
			"price":            sub.Plan.Price,
			"lockedPrice":      lockedPrice,
			"status":           sub.Status,
			"currentPeriodEnd": sub.CurrentPeriodEnd,
		}
	}

	return c.JSON(fiber.Map{
		"hasPlan":             hasPlan,
		"plan":                planInfo,
		"connectedPagesCount": connectedCount,
		"connectedPages":      connectedPages,
		"totalAvailablePages": len(pages),
		"step":                step, // "plan", "page", "completed"
		"isCompleted":         step == "completed",
	})
}
