package httpx

import (
	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/usecase"
)

func (h *Handler) listWarehouses(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	list, err := h.Inventory.ListWarehouses(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"items": list})
}

func (h *Handler) createWarehouse(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	var in usecase.CreateWarehouseInput
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	wh, err := h.Inventory.CreateWarehouse(c.Context(), accountID, in)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(wh)
}

func (h *Handler) updateWarehouse(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	id := c.Params("id")
	var in usecase.UpdateWarehouseInput
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	wh, err := h.Inventory.UpdateWarehouse(c.Context(), accountID, id, in)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(wh)
}

func (h *Handler) deleteWarehouse(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	id := c.Params("id")
	if err := h.Inventory.DeleteWarehouse(c.Context(), accountID, id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "deleted"})
}

func (h *Handler) getProductStocks(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	productID := c.Params("id")
	stocks, err := h.Inventory.GetProductStocks(c.Context(), accountID, productID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"items": stocks})
}

func (h *Handler) setProductStock(c fiber.Ctx) error {
	accountID := c.Locals("accountID").(string)
	productID := c.Params("id")
	var in usecase.SetStockInput
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if in.WarehouseID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "warehouseId is required"})
	}
	err := h.Inventory.SetProductStock(c.Context(), accountID, productID, in.WarehouseID, in.Quantity, in.AlertThreshold)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "updated"})
}
