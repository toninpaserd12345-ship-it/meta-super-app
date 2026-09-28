package httpx

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/mail"
	"net/url"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/responsetime"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/usecase"
)

type Handler struct {
	auth                                        *usecase.Auth
	meta                                        *usecase.Meta
	tokens                                      domain.TokenService
	metaMode, metaFrontendRedirect              string
	facebookLoginUserID, facebookLoginAccountID string
	ticketMu                                    sync.Mutex
	tickets                                     map[string]loginTicket
}

type loginTicket struct {
	Result    *usecase.LoginOutput
	ExpiresAt time.Time
}

func NewHandler(auth *usecase.Auth, meta *usecase.Meta, tokens domain.TokenService, metaMode, metaFrontendRedirect, facebookLoginUserID, facebookLoginAccountID string) *Handler {
	return &Handler{auth: auth, meta: meta, tokens: tokens, metaMode: metaMode, metaFrontendRedirect: metaFrontendRedirect, facebookLoginUserID: facebookLoginUserID, facebookLoginAccountID: facebookLoginAccountID, tickets: make(map[string]loginTicket)}
}

func (h *Handler) App() *fiber.App {
	app := fiber.New(fiber.Config{AppName: "Meta Super App API", ErrorHandler: errorHandler, BodyLimit: 1 << 20, ReadTimeout: 20_000_000_000, WriteTimeout: 20_000_000_000, IdleTimeout: 60_000_000_000})
	app.Use(recover.New(), requestid.New(), responsetime.New(), helmet.New())
	app.Use(func(c fiber.Ctx) error { c.Set(fiber.HeaderCacheControl, "no-store"); return c.Next() })
	app.Get("/health", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Post("/auth/login", h.login)
	app.Post("/auth/facebook/start", h.startFacebookLogin)
	app.Post("/auth/facebook/exchange", h.exchangeFacebookLogin)
	app.Get("/api/v1/meta/oauth/callback", h.completeMetaOAuth)
	app.Get("/api/v1/meta/webhook", h.verifyMetaWebhook)
	app.Post("/api/v1/meta/webhook", h.receiveMetaWebhook)
	auth := app.Group("", authenticate(h.tokens))
	auth.Get("/auth/me", h.me)
	auth.Get("/api/v1/accounts", h.accounts)
	auth.Get("/api/v1/dashboard", requireAccount(h.auth, "dashboard:read"), h.dashboard)
	auth.Get("/api/v1/meta/pages", requireAccount(h.auth, domain.ClaimPagesRead), h.metaPages)
	auth.Post("/api/v1/meta/oauth/start", requireAccount(h.auth, domain.ClaimPagesConnect), h.startMetaOAuth)
	auth.Post("/api/v1/meta/pages/connect", requireAccount(h.auth, domain.ClaimPagesConnect), h.connectMetaPage)
	auth.Post("/api/v1/meta/pages/disconnect", requireAccount(h.auth, domain.ClaimPagesConnect), h.disconnectMetaPage)
	auth.Get("/api/v1/meta/pages/:pageID/posts", requireAccount(h.auth, domain.ClaimPagesRead), h.metaPosts)
	auth.Get("/api/v1/meta/ad-accounts", requireAccount(h.auth, domain.ClaimPagesRead), h.metaAdAccounts)
	auth.Get("/api/v1/meta/ad-accounts/:adAccountID/campaigns", requireAccount(h.auth, domain.ClaimPagesRead), h.metaCampaigns)
	auth.Get("/api/v1/meta/campaigns/:campaignID/ads", requireAccount(h.auth, domain.ClaimPagesRead), h.metaAds)
	auth.Get("/api/v1/meta/product-bindings", requireAccount(h.auth, domain.ClaimPagesRead), h.productBindings)
	auth.Post("/api/v1/meta/product-bindings", requireAccount(h.auth, domain.ClaimPagesConnect), h.saveProductBinding)
	auth.Post("/api/v1/meta/product-bindings/batch", requireAccount(h.auth, domain.ClaimPagesConnect), h.saveProductBindings)
	auth.Get("/api/v1/products", requireAccount(h.auth, domain.ClaimPagesRead), h.products)
	auth.Post("/api/v1/products", requireAccount(h.auth, domain.ClaimPagesConnect), h.saveProduct)
	auth.Get("/api/v1/reply-flows", requireAccount(h.auth, domain.ClaimPagesRead), h.replyFlows)
	auth.Post("/api/v1/reply-flows", requireAccount(h.auth, domain.ClaimPagesConnect), h.saveReplyFlow)
	return app
}

func (h *Handler) products(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.meta.ListProducts(ctx, userID(c), c.Locals("accountID").(string))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": items, "storage": "memory"})
}

func (h *Handler) saveProduct(c fiber.Ctx) error {
	var body domain.Product
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, 400, "invalid_request", "Product is invalid.")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	item, err := h.meta.SaveProduct(ctx, userID(c), c.Locals("accountID").(string), body)
	if err != nil {
		return fail(c, 400, "product_failed", err.Error())
	}
	return c.JSON(fiber.Map{"item": item})
}

func (h *Handler) replyFlows(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.meta.ListReplyFlows(ctx, userID(c), c.Locals("accountID").(string))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": items, "storage": "memory"})
}

func (h *Handler) saveReplyFlow(c fiber.Ctx) error {
	var body domain.ReplyFlow
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, 400, "invalid_request", "Reply flow is invalid.")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	item, err := h.meta.SaveReplyFlow(ctx, userID(c), c.Locals("accountID").(string), body)
	if err != nil {
		return fail(c, 400, "reply_flow_failed", err.Error())
	}
	return c.JSON(fiber.Map{"item": item})
}

func (h *Handler) login(c fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, 400, "invalid_request", "Request body is invalid.")
	}
	if _, err := mail.ParseAddress(body.Email); err != nil || len(body.Password) < 8 || len(body.Password) > 128 {
		return fail(c, 400, "validation_error", "A valid email and password are required.")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	result, err := h.auth.Login(ctx, usecase.LoginInput{Email: body.Email, Password: body.Password})
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(result)
}
func (h *Handler) startFacebookLogin(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	authorizationURL, err := h.meta.AuthorizationURL(ctx, h.facebookLoginUserID, h.facebookLoginAccountID, nil)
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"authorizationUrl": authorizationURL})
}
func (h *Handler) exchangeFacebookLogin(c fiber.Ctx) error {
	var body struct {
		Ticket string `json:"ticket"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Ticket == "" {
		return fail(c, 400, "invalid_ticket", "Login ticket is required.")
	}
	h.ticketMu.Lock()
	item, ok := h.tickets[body.Ticket]
	delete(h.tickets, body.Ticket)
	h.ticketMu.Unlock()
	if !ok || time.Now().After(item.ExpiresAt) {
		return fail(c, 401, "invalid_ticket", "Login ticket is invalid or expired.")
	}
	return c.JSON(item.Result)
}
func (h *Handler) me(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	user, err := h.auth.Me(ctx, userID(c))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(user)
}
func (h *Handler) accounts(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	user, err := h.auth.Me(ctx, userID(c))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": user.Accounts})
}
func (h *Handler) dashboard(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"accountId": c.Locals("accountID"), "summary": fiber.Map{"customers": 0, "orders": 0}})
}
func (h *Handler) metaPages(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	pages, err := h.meta.ListPages(ctx, userID(c), c.Locals("accountID").(string))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": pages, "mode": h.metaMode})
}
func (h *Handler) startMetaOAuth(c fiber.Ctx) error {
	var body struct {
		Permissions []string `json:"permissions"`
	}
	if len(c.Body()) > 0 {
		if err := c.Bind().JSON(&body); err != nil {
			return fail(c, 400, "invalid_request", "Permission selection is invalid.")
		}
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	authorizationURL, err := h.meta.AuthorizationURL(ctx, userID(c), c.Locals("accountID").(string), body.Permissions)
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"authorizationUrl": authorizationURL})
}
func (h *Handler) completeMetaOAuth(c fiber.Ctx) error {
	redirect := h.metaFrontendRedirect
	if message := c.Query("error_description"); message != "" {
		return c.Redirect().To(redirect + "?meta=error&reason=" + url.QueryEscape(message))
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	result, err := h.meta.CompleteAuthorization(ctx, c.Query("state"), c.Query("code"))
	if err != nil {
		slog.Error("meta oauth callback failed", "error", err)
		return c.Redirect().To(redirect + "?meta=error&reason=" + url.QueryEscape("Facebook authorization failed"))
	}
	login, err := h.auth.LoginByUserID(ctx, result.UserID)
	if err != nil {
		slog.Error("facebook app login failed", "error", err)
		return c.Redirect().To(redirect + "?meta=error")
	}
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return fail(c, 500, "ticket_failed", "Login could not be completed.")
	}
	ticket := base64.RawURLEncoding.EncodeToString(bytes)
	h.ticketMu.Lock()
	for key, item := range h.tickets {
		if time.Now().After(item.ExpiresAt) {
			delete(h.tickets, key)
		}
	}
	h.tickets[ticket] = loginTicket{Result: login, ExpiresAt: time.Now().Add(2 * time.Minute)}
	h.ticketMu.Unlock()
	return c.Redirect().To(redirect + "?ticket=" + url.QueryEscape(ticket))
}
func (h *Handler) verifyMetaWebhook(c fiber.Ctx) error {
	if c.Query("hub.mode") != "subscribe" || !h.meta.VerifyWebhook(c.Query("hub.verify_token")) {
		return fail(c, fiber.StatusForbidden, "webhook_verification_failed", "Webhook verification failed.")
	}
	challenge := c.Query("hub.challenge")
	if challenge == "" {
		return fail(c, fiber.StatusBadRequest, "challenge_required", "Webhook challenge is required.")
	}
	return c.SendString(challenge)
}
func (h *Handler) receiveMetaWebhook(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	body := append([]byte(nil), c.Body()...)
	if err := h.meta.ReceiveWebhook(ctx, body, c.Get("X-Hub-Signature-256")); err != nil {
		slog.Warn("meta webhook rejected", "error", err)
		return fail(c, fiber.StatusUnauthorized, "invalid_webhook", "Webhook signature or payload is invalid.")
	}
	return c.SendStatus(fiber.StatusOK)
}
func (h *Handler) connectMetaPage(c fiber.Ctx) error {
	var body struct {
		PageID string `json:"pageId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, 400, "invalid_request", "Page selection is invalid.")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	page, err := h.meta.ConnectPage(ctx, userID(c), c.Locals("accountID").(string), body.PageID)
	if err != nil {
		slog.Error("meta page subscription failed", "page_id", body.PageID, "error", err)
		return fail(c, fiber.StatusBadGateway, "meta_subscription_failed", "Meta rejected the Page Webhook subscription. Verify Page access and pages_manage_metadata permission.")
	}
	return c.JSON(fiber.Map{"page": page})
}
func (h *Handler) disconnectMetaPage(c fiber.Ctx) error {
	var body struct {
		PageID string `json:"pageId"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, 400, "invalid_request", "Page selection is invalid.")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	page, err := h.meta.DisconnectPage(ctx, userID(c), c.Locals("accountID").(string), body.PageID)
	if err != nil {
		slog.Error("meta page unsubscription failed", "page_id", body.PageID, "error", err)
		return fail(c, fiber.StatusBadGateway, "meta_unsubscription_failed", "Meta rejected the Page Webhook unsubscription.")
	}
	return c.JSON(fiber.Map{"page": page})
}

func (h *Handler) metaPosts(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	posts, err := h.meta.ListPosts(ctx, userID(c), c.Locals("accountID").(string), c.Params("pageID"))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": posts})
}
func (h *Handler) metaAdAccounts(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.meta.ListAdAccounts(ctx, userID(c), c.Locals("accountID").(string))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) metaCampaigns(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.meta.ListCampaigns(ctx, userID(c), c.Locals("accountID").(string), c.Params("adAccountID"))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) metaAds(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.meta.ListAds(ctx, userID(c), c.Locals("accountID").(string), c.Params("campaignID"))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) productBindings(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.meta.ListProductBindings(ctx, userID(c), c.Locals("accountID").(string))
	if err != nil {
		return usecaseError(c, err)
	}
	return c.JSON(fiber.Map{"items": items, "storage": "memory", "replyPolicy": "first-message-per-user"})
}

func (h *Handler) saveProductBinding(c fiber.Ctx) error {
	var body domain.ProductBinding
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, 400, "invalid_request", "Product binding is invalid.")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	item, err := h.meta.SaveProductBinding(ctx, userID(c), c.Locals("accountID").(string), body)
	if err != nil {
		return fail(c, 400, "binding_failed", err.Error())
	}
	return c.JSON(fiber.Map{"item": item})
}
func (h *Handler) saveProductBindings(c fiber.Ctx) error {
	var body struct {
		Items []domain.ProductBinding `json:"items"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, 400, "invalid_request", "Automation batch is invalid.")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.meta.SaveProductBindings(ctx, userID(c), c.Locals("accountID").(string), body.Items)
	if err != nil {
		return fail(c, 400, "batch_binding_failed", err.Error())
	}
	return c.JSON(fiber.Map{"items": items, "count": len(items)})
}
