package httpx

import (
	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/usecase"
)

func (h *Handler) listOrders(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	status := c.Query("status", "")
	orders, err := h.Orders.ListOrders(c.Context(), accountID, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"items": orders})
}

func (h *Handler) getOrder(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	id := c.Params("id")
	order, err := h.Orders.GetOrder(c.Context(), accountID, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Order not found"})
	}
	return c.JSON(order)
}

func (h *Handler) createOrder(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	var in usecase.CreateOrderInput
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	order, err := h.Orders.CreateOrder(c.Context(), accountID, in)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(order)
}

func (h *Handler) updateOrderStatus(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	id := c.Params("id")
	var in usecase.UpdateOrderStatusInput
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	order, err := h.Orders.UpdateOrderStatus(c.Context(), accountID, id, in)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(order)
}

func (h *Handler) deleteOrder(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	id := c.Params("id")
	if err := h.Orders.DeleteOrder(c.Context(), accountID, id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "deleted"})
}
