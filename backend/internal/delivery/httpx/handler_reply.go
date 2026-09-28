package httpx

import (
	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/domain"
)

// Request bodies
type createReplySetRequest struct {
	Name string `json:"name"`
}

type updateReplyItemsRequest struct {
	Items []domain.ReplyItem `json:"items"`
}

func (h *Handler) GetReplySets(c fiber.Ctx) error {
	accountID := c.Locals("account_id").(string)

	sets, err := h.Reply.GetSets(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(sets)
}

func (h *Handler) GetReplySet(c fiber.Ctx) error {
	accountID := c.Locals("account_id").(string)
	setID := c.Params("id")

	set, err := h.Reply.GetSet(c.Context(), setID, accountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(set)
}

func (h *Handler) CreateReplySet(c fiber.Ctx) error {
	accountID := c.Locals("account_id").(string)

	var req createReplySetRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	set, err := h.Reply.CreateSet(c.Context(), accountID, req.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(set)
}

func (h *Handler) UpdateReplySet(c fiber.Ctx) error {
	accountID := c.Locals("account_id").(string)
	setID := c.Params("id")

	var req createReplySetRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	err := h.Reply.UpdateSet(c.Context(), setID, accountID, req.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) DeleteReplySet(c fiber.Ctx) error {
	accountID := c.Locals("account_id").(string)
	setID := c.Params("id")

	err := h.Reply.DeleteSet(c.Context(), setID, accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) UpdateReplyItems(c fiber.Ctx) error {
	accountID := c.Locals("account_id").(string)
	setID := c.Params("id")

	var req updateReplyItemsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	err := h.Reply.UpdateItems(c.Context(), setID, accountID, req.Items)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusOK)
}
