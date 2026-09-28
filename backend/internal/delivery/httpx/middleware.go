package httpx

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/usecase"
)

const userIDKey = "authenticatedUserID"

func authenticate(tokens domain.TokenService) fiber.Handler {
	return func(c fiber.Ctx) error {
		parts := strings.Fields(c.Get(fiber.HeaderAuthorization))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return fail(c, 401, "unauthorized", "Bearer token is required.")
		}
		claims, err := tokens.Parse(parts[1])
		if err != nil {
			return fail(c, 401, "unauthorized", "Token is invalid or expired.")
		}
		c.Locals(userIDKey, claims.UserID)
		return c.Next()
	}
}

func userID(c fiber.Ctx) string { value, _ := c.Locals(userIDKey).(string); return value }

func requestContext(c fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Context(), 10*time.Second)
}

func requireAccount(auth *usecase.Auth, claim domain.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		accountID := strings.TrimSpace(c.Get("X-Account-ID"))
		if accountID == "" {
			return fail(c, 400, "account_required", "X-Account-ID is required.")
		}
		ctx, cancel := requestContext(c)
		defer cancel()
		membership, err := auth.Membership(ctx, userID(c), accountID)
		if err != nil {
			return fail(c, 403, "account_forbidden", "Account access was denied.")
		}
		if !membership.Can(claim) {
			return fail(c, 403, "permission_denied", "Required permission: "+string(claim))
		}
		c.Locals("accountID", accountID)
		return c.Next()
	}
}
