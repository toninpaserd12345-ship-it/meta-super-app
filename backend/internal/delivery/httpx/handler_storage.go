package httpx

import (
	"github.com/gofiber/fiber/v3"
)

func (h *Handler) UploadFile(c fiber.Ctx) error {
	// Parse the multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}

	// Optional folder parameter
	folder := c.FormValue("folder", "general")

	// Call UseCase
	url, err := h.Storage.UploadFile(c.Context(), file, folder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"url": url,
	})
}
