package httpx

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/usecase"
)

// Request bodies
type createReplySetRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type updateReplyItemRequest struct {
	Type       string `json:"type"`
	Content    string `json:"content"`
	OrderIndex int    `json:"orderIndex"`
	IsEnabled  *bool  `json:"isEnabled"`
}

type updateReplyItemsRequest struct {
	Items []updateReplyItemRequest `json:"items"`
}

func (h *Handler) GetReplySets(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)

	sets, err := h.Reply.GetSets(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(sets)
}

func (h *Handler) GetReplySet(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	setID := c.Params("id")

	set, err := h.Reply.GetSet(c.Context(), setID, accountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(set)
}

func (h *Handler) CreateReplySet(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)

	var req createReplySetRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	set, err := h.Reply.CreateSet(c.Context(), accountID, req.Code, req.Name)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidReplySetCode) || errors.Is(err, usecase.ErrReplySetCodeExists) || errors.Is(err, usecase.ErrReplySetNameExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(set)
}

func (h *Handler) UpdateReplySet(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	setID := c.Params("id")

	var req createReplySetRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	err := h.Reply.UpdateSet(c.Context(), setID, accountID, req.Code, req.Name)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidReplySetCode) || errors.Is(err, usecase.ErrReplySetCodeExists) || errors.Is(err, usecase.ErrReplySetNameExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) DeleteReplySet(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	setID := c.Params("id")
	rules, err := h.Automation.GetRules(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	for _, rule := range rules {
		if rule.ReplySetID == setID {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "This Reply Set is used by an Auto Reply. Remove that automation before deleting the set.",
			})
		}
	}

	err = h.Reply.DeleteSet(c.Context(), setID, accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) UpdateReplyItems(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	setID := c.Params("id")

	var req updateReplyItemsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	items := make([]domain.ReplyItem, len(req.Items))
	for index, item := range req.Items {
		enabled := true
		if item.IsEnabled != nil {
			enabled = *item.IsEnabled
		}
		items[index] = domain.ReplyItem{
			Type:       item.Type,
			Content:    item.Content,
			OrderIndex: item.OrderIndex,
			IsEnabled:  enabled,
		}
	}
	hasEnabledItem := false
	for _, item := range items {
		if item.IsEnabled {
			hasEnabledItem = true
			break
		}
	}
	if !hasEnabledItem {
		rules, err := h.Automation.GetRules(c.Context(), accountID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		for _, rule := range rules {
			if rule.ReplySetID == setID && rule.IsActive {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "Pause or remove active Auto Replies before disabling every message in this set.",
				})
			}
		}
	}

	err := h.Reply.UpdateItems(c.Context(), setID, accountID, items)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusOK)
}
