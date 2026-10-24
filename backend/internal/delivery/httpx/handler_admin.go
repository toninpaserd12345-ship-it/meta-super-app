package httpx

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) adminLogin(c fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" || body.Password == "" {
		return fail(c, fiber.StatusBadRequest, "invalid_request", "Email and password are required.")
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	out, err := h.admin.Login(ctx, strings.TrimSpace(body.Email), body.Password)
	if err != nil {
		return fail(c, fiber.StatusUnauthorized, "invalid_credentials", "Invalid admin email or password.")
	}

	return c.JSON(out)
}

func (h *Handler) adminOverview(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()

	overview, err := h.admin.Overview(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(overview)
}

func (h *Handler) adminListTenants(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()

	search := c.Query("search")
	tenants, err := h.admin.ListTenants(ctx, search)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"items": tenants})
}

func (h *Handler) adminListPages(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()

	pages, err := h.admin.ListAllPages(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"items": pages})
}

func (h *Handler) adminListUsers(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()

	users, err := h.admin.ListAllUsers(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"items": users})
}

func (h *Handler) adminSetTenantStatus(c fiber.Ctx) error {
	accountID := c.Params("id")
	var body struct {
		Status string `json:"status"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Status == "" {
		return fail(c, fiber.StatusBadRequest, "invalid_request", "Status is required (active or suspended).")
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	if err := h.admin.SetTenantStatus(ctx, accountID, body.Status); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok", "accountId": accountID, "newStatus": body.Status})
}

func (h *Handler) adminDeleteTenant(c fiber.Ctx) error {
	accountID := c.Params("id")
	if accountID == "" {
		return fail(c, fiber.StatusBadRequest, "invalid_request", "Account ID is required.")
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	if err := h.admin.DeleteTenant(ctx, accountID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok", "deletedAccountId": accountID})
}

func (h *Handler) adminListTransactions(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()

	status := c.Query("status")
	txns, err := h.admin.ListTransactions(ctx, status)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"items": txns})
}

func (h *Handler) adminApproveTransaction(c fiber.Ctx) error {
	txnID := c.Params("id")
	if txnID == "" {
		return fail(c, fiber.StatusBadRequest, "invalid_request", "Transaction ID is required.")
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	if err := h.admin.ApproveTransaction(ctx, txnID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok", "transactionId": txnID, "approved": true})
}

func (h *Handler) adminRejectTransaction(c fiber.Ctx) error {
	txnID := c.Params("id")
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind().JSON(&body)

	ctx, cancel := requestContext(c)
	defer cancel()

	if err := h.admin.RejectTransaction(ctx, txnID, body.Reason); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok", "transactionId": txnID, "rejected": true})
}
