package httpx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/usecase"
)

type createAutomationRequest struct {
	PageID       string `json:"pageId"`
	TriggerType  string `json:"triggerType"`
	TriggerValue string `json:"triggerValue"`
	ProductID    string `json:"productId"`
	ReplySetID   string `json:"replySetId"`
}

type updateAutomationRequest struct {
	TriggerType  string `json:"triggerType"`
	TriggerValue string `json:"triggerValue"`
	ProductID    string `json:"productId"`
	ReplySetID   string `json:"replySetId"`
	IsActive     bool   `json:"isActive"`
}

func (h *Handler) GetAutomationRules(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)

	rules, err := h.Automation.GetRules(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"items": rules})
}

func (h *Handler) CreateAutomationRule(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)

	var req createAutomationRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := h.validateAutomationReferences(c, accountID, req.ProductID, req.ReplySetID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	rule, err := h.Automation.CreateRule(c.Context(), accountID, req.PageID, req.TriggerType, req.TriggerValue, req.ProductID, req.ReplySetID)
	if err != nil {
		return automationError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(rule)
}

func (h *Handler) UpdateAutomationRule(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	id := c.Params("id")

	var req updateAutomationRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := h.validateAutomationReferences(c, accountID, req.ProductID, req.ReplySetID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	err := h.Automation.UpdateRule(c.Context(), id, accountID, req.TriggerType, req.TriggerValue, req.ProductID, req.ReplySetID, req.IsActive)
	if err != nil {
		return automationError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) DeleteAutomationRule(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	id := c.Params("id")

	err := h.Automation.DeleteRule(c.Context(), id, accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) validateAutomationReferences(c fiber.Ctx, accountID, productID, replySetID string) error {
	productID, replySetID = strings.TrimSpace(productID), strings.TrimSpace(replySetID)
	if productID == "" || replySetID == "" {
		return fmt.Errorf("productId and replySetId are required")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	replySet, err := h.Reply.GetSet(ctx, replySetID, accountID)
	if err != nil {
		return fmt.Errorf("the selected Reply Set does not belong to this account")
	}
	hasEnabledItem := false
	for _, item := range replySet.Items {
		if item.IsEnabled {
			hasEnabledItem = true
			break
		}
	}
	if !hasEnabledItem {
		return fmt.Errorf("the selected Reply Set needs at least one enabled message")
	}
	products, err := h.Product.ListProducts(ctx, userID(c), accountID)
	if err != nil {
		return fmt.Errorf("could not verify the selected Product: %w", err)
	}
	for _, product := range products {
		if product.ID == productID {
			return nil
		}
	}
	return fmt.Errorf("the selected Product does not belong to this account")
}

func automationError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, usecase.ErrInvalidAutomationRule):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, usecase.ErrAutomationRuleConflict):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
}
