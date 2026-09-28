package httpx

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/usecase"
)

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func fail(c fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

func usecaseError(c fiber.Ctx, err error) error {
	if errors.Is(err, usecase.ErrInvalidCredentials) {
		return fail(c, fiber.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect.")
	}
	slog.Error("request failed", "error", err, "path", c.Path())
	return fail(c, fiber.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
}

func errorHandler(c fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	var fiberError *fiber.Error
	if errors.As(err, &fiberError) {
		status = fiberError.Code
	}
	if status >= 500 {
		slog.Error("fiber request failed", "error", err, "path", c.Path())
	}
	return fail(c, status, "request_failed", fiber.ErrInternalServerError.Message)
}
