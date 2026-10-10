package httpx

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/mail"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/responsetime"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/usecase"
)

type Handler struct {
	auth       *usecase.Auth
	meta       *usecase.Meta
	team       *usecase.Team
	billing    *usecase.Billing
	Storage    *usecase.StorageUseCase
	Reply      *usecase.Reply
	Automation *usecase.Automation
	Product    *usecase.Product
	Stream     *usecase.ChatStream
	Leads      *usecase.Leads

	tokens                         domain.TokenService
	metaMode, metaFrontendRedirect string
	// facebookLoginUserID, facebookLoginAccountID string (Removed)
	ticketMu sync.Mutex
	tickets  map[string]loginTicket
}

type loginTicket struct {
	Result    *usecase.LoginOutput
	AccountID string
	ExpiresAt time.Time
}

type facebookLoginExchange struct {
	*usecase.LoginOutput
	AccountID string `json:"accountId,omitempty"`
}

func NewHandler(auth *usecase.Auth, meta *usecase.Meta, team *usecase.Team, billing *usecase.Billing, storage *usecase.StorageUseCase, reply *usecase.Reply, automation *usecase.Automation, product *usecase.Product, stream *usecase.ChatStream, leads *usecase.Leads, tokens domain.TokenService, metaMode, metaFrontendRedirect string) *Handler {
	return &Handler{auth: auth, meta: meta, team: team, billing: billing, Storage: storage, Reply: reply, Automation: automation, Product: product, Stream: stream, Leads: leads, tokens: tokens, metaMode: metaMode, metaFrontendRedirect: metaFrontendRedirect, tickets: make(map[string]loginTicket)}
}

func (h *Handler) App() *fiber.App {
	app := fiber.New(fiber.Config{AppName: "Meta Super App API", ErrorHandler: errorHandler, BodyLimit: 1 << 20, ReadTimeout: 20_000_000_000, WriteTimeout: 20_000_000_000, IdleTimeout: 60_000_000_000})
	app.Use(logger.New(logger.Config{
		// Never log request/response bodies or authorization headers. Login bodies,
		// JWTs, Meta access tokens and webhook payloads may contain secrets.
		Format: "${time} ${ip} ${status} ${latency} ${method} ${path}\n",
	}), recover.New(), requestid.New(), responsetime.New(), helmet.New(), cors.New(cors.Config{AllowOrigins: []string{"*"}, AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Account-ID"}}))
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
	auth.Get("/api/v1/chat/history", requireAccount(h.auth, domain.ClaimPagesRead), h.getChatHistory)
	auth.Post("/api/v1/chat/send", requireAccount(h.auth, domain.ClaimPagesConnect), h.sendChatMessage)
	auth.Get("/api/v1/chat/stream", requireAccount(h.auth, domain.ClaimPagesRead), h.chatStream)
	auth.Get("/api/v1/leads", requireAccount(h.auth, "customers:read"), h.listLeads)
	auth.Get("/api/v1/leads/settings", requireAccount(h.auth, "settings:read"), h.leadSettings)
	auth.Put("/api/v1/leads/settings", requireAccount(h.auth, "settings:update"), h.saveLeadSettings)
	auth.Patch("/api/v1/leads/:id/read", requireAccount(h.auth, "customers:update"), h.markLeadRead)

	// Storage
	auth.Post("/api/v1/storage/upload", requireAccount(h.auth, domain.ClaimPagesConnect), h.UploadFile)

	// Quick Replies
	auth.Get("/api/v1/replies", requireAccount(h.auth, domain.ClaimPagesRead), h.GetReplySets)
	auth.Get("/api/v1/replies/:id", requireAccount(h.auth, domain.ClaimPagesRead), h.GetReplySet)
	auth.Post("/api/v1/replies", requireAccount(h.auth, domain.ClaimPagesConnect), h.CreateReplySet)
	auth.Put("/api/v1/replies/:id", requireAccount(h.auth, domain.ClaimPagesConnect), h.UpdateReplySet)
	auth.Delete("/api/v1/replies/:id", requireAccount(h.auth, domain.ClaimPagesConnect), h.DeleteReplySet)
	auth.Put("/api/v1/replies/:id/items", requireAccount(h.auth, domain.ClaimPagesConnect), h.UpdateReplyItems)

	// Post/Ad Automation
	auth.Get("/api/v1/automation/rules", requireAccount(h.auth, domain.ClaimPagesRead), h.GetAutomationRules)
	auth.Post("/api/v1/automation/rules", requireAccount(h.auth, domain.ClaimPagesConnect), h.CreateAutomationRule)
	auth.Put("/api/v1/automation/rules/:id", requireAccount(h.auth, domain.ClaimPagesConnect), h.UpdateAutomationRule)
	auth.Delete("/api/v1/automation/rules/:id", requireAccount(h.auth, domain.ClaimPagesConnect), h.DeleteAutomationRule)
	auth.Get("/api/v1/automations", requireAccount(h.auth, domain.ClaimPagesRead), h.GetAutomationFlows)
	auth.Post("/api/v1/automations", requireAccount(h.auth, domain.ClaimPagesConnect), h.CreateAutomationFlow)
	auth.Put("/api/v1/automations/:id", requireAccount(h.auth, domain.ClaimPagesConnect), h.UpdateAutomationFlow)
	auth.Patch("/api/v1/automations/:id/status", requireAccount(h.auth, domain.ClaimPagesConnect), h.UpdateAutomationFlowStatus)
	auth.Delete("/api/v1/automations/:id", requireAccount(h.auth, domain.ClaimPagesConnect), h.DeleteAutomationFlow)

	auth.Get("/api/v1/meta/pages", requireAccount(h.auth, domain.ClaimPagesRead), h.metaPages)
	auth.Get("/api/v1/meta/whatsapp/signup/config", requireAccount(h.auth, domain.ClaimPagesRead), h.whatsAppSignupConfig)
	auth.Post("/api/v1/meta/whatsapp/signup/complete", requireAccount(h.auth, domain.ClaimPagesConnect), h.completeWhatsAppSignup)
	auth.Get("/api/v1/meta/pages/:pageID/picture", requireAccount(h.auth, domain.ClaimPagesRead), h.metaPagePicture)
	auth.Post("/api/v1/meta/oauth/start", requireAccount(h.auth, domain.ClaimPagesConnect), h.startMetaOAuth)
	auth.Post("/api/v1/meta/pages/connect", requireAccount(h.auth, domain.ClaimPagesConnect), h.connectMetaPage)
	auth.Post("/api/v1/meta/pages/disconnect", requireAccount(h.auth, domain.ClaimPagesConnect), h.disconnectMetaPage)
	auth.Get("/api/v1/meta/pages/:pageID/posts", requireAccount(h.auth, domain.ClaimPagesRead), h.metaPosts)
	auth.Get("/api/v1/meta/ad-accounts", requireAccount(h.auth, domain.ClaimPagesRead), h.metaAdAccounts)
	auth.Get("/api/v1/meta/ad-accounts/:adAccountID/campaigns", requireAccount(h.auth, domain.ClaimPagesRead), h.metaCampaigns)
	auth.Get("/api/v1/meta/campaigns/:campaignID/ads", requireAccount(h.auth, domain.ClaimPagesRead), h.metaAds)
	auth.Get("/api/v1/products", requireAccount(h.auth, domain.ClaimPagesRead), h.products)
	auth.Post("/api/v1/products", requireAccount(h.auth, domain.ClaimPagesConnect), h.saveProduct)
	auth.Delete("/api/v1/products/:id", requireAccount(h.auth, domain.ClaimPagesConnect), h.deleteProduct)
	// Team Management
	auth.Get("/api/v1/team", requireAccount(h.auth, "users:read"), h.teamList)
	auth.Post("/api/v1/team/invite", requireAccount(h.auth, "users:invite"), h.teamInvite)
	auth.Put("/api/v1/team/:userID", requireAccount(h.auth, "users:update"), h.teamUpdateRole)
	auth.Delete("/api/v1/team/:userID", requireAccount(h.auth, "users:remove"), h.teamRemove)

	// Billing
	auth.Get("/api/v1/billing/plans", requireAccount(h.auth, "dashboard:read"), h.billingPlans)
	auth.Get("/api/v1/billing/subscription", requireAccount(h.auth, "billing:read"), h.billingSubscription)
	auth.Post("/api/v1/billing/checkout", requireAccount(h.auth, "billing:read"), h.billingCheckout)

	return app
}

func (h *Handler) products(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.Product.ListProducts(ctx, userID(c), c.Locals("accountID").(string))
	if err != nil {
		return usecaseError(c, err)
	}
	for i := range items {
		if items[i].ImageUrl != "" {
			items[i].ImageUrl = h.Storage.GetPublicURL(items[i].ImageUrl)
		}
	}
	return c.JSON(fiber.Map{"items": items, "storage": "database"})
}

func (h *Handler) saveProduct(c fiber.Ctx) error {
	var body domain.Product
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, 400, "invalid_request", "Product is invalid.")
	}

	if body.ImageUrl != "" {
		body.ImageUrl = h.Storage.StripPublicURL(body.ImageUrl)
	}

	ctx, cancel := requestContext(c)
	defer cancel()
	item, err := h.Product.SaveProduct(ctx, userID(c), c.Locals("accountID").(string), body)
	if err != nil {
		return fail(c, 400, "product_failed", err.Error())
	}
	if item.ImageUrl != "" {
		item.ImageUrl = h.Storage.GetPublicURL(item.ImageUrl)
	}
	return c.JSON(fiber.Map{"item": item})
}

func (h *Handler) deleteProduct(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	accountID := c.Locals("accountID").(string)
	productID := c.Params("id")
	rules, err := h.Automation.GetRules(ctx, accountID)
	if err != nil {
		return usecaseError(c, err)
	}
	for _, rule := range rules {
		if rule.ProductID == productID {
			return fail(c, fiber.StatusConflict, "product_in_use", "This product is used by an Automation. Remove or change that Automation before deleting the product.")
		}
	}
	if err = h.Product.DeleteProduct(ctx, userID(c), accountID, productID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fail(c, fiber.StatusNotFound, "product_not_found", "The product was not found in this workspace.")
		}
		return usecaseError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
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
	authorizationURL, err := h.meta.AuthorizationURL(ctx, "", "", nil)
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
	return c.JSON(facebookLoginExchange{LoginOutput: item.Result, AccountID: item.AccountID})
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
func (h *Handler) metaPages(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	pages, err := h.meta.ListPages(ctx, userID(c), c.Locals("accountID").(string))
	if err != nil {
		return usecaseError(c, err)
	}
	diagnostics, diagnosticsErr := h.meta.WhatsAppDiagnostics(ctx, c.Locals("accountID").(string))
	if diagnosticsErr != nil {
		slog.Warn("unable to load WhatsApp diagnostics", "error", diagnosticsErr)
	}
	return c.JSON(fiber.Map{"items": pages, "mode": h.metaMode, "whatsappDiagnostics": diagnostics})
}
func (h *Handler) whatsAppSignupConfig(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	return c.JSON(h.meta.WhatsAppSignupConfig(ctx))
}
func (h *Handler) completeWhatsAppSignup(c fiber.Ctx) error {
	var body domain.MetaWhatsAppSignupInput
	if err := c.Bind().JSON(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "invalid_request", "WhatsApp signup result is invalid.")
	}
	ctx, cancel := requestContext(c)
	defer cancel()
	items, err := h.meta.CompleteWhatsAppSignup(ctx, c.Locals("accountID").(string), body)
	if err != nil {
		slog.Error("WhatsApp Embedded Signup failed", "error", err)
		return fail(c, fiber.StatusBadGateway, "whatsapp_signup_failed", err.Error())
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) metaPagePicture(c fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()
	picture, err := h.meta.PagePicture(ctx, c.Locals("accountID").(string), c.Params("pageID"))
	if err != nil {
		return usecaseError(c, err)
	}
	c.Set(fiber.HeaderContentType, picture.ContentType)
	c.Set(fiber.HeaderCacheControl, "no-store, no-cache, must-revalidate")
	c.Set(fiber.HeaderPragma, "no-cache")
	return c.Send(picture.Data)
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
		return c.Redirect().To(redirect + "?meta=error&reason=" + url.QueryEscape(metaOAuthFailureReason(err)))
	}
	var login *usecase.LoginOutput
	var errLogin error
	if result.UserID != "" {
		login, errLogin = h.auth.LoginByUserID(ctx, result.UserID)
	} else {
		login, errLogin = h.auth.LoginOrCreateByFacebook(ctx, result.FacebookID, result.Name, result.Email)
	}
	if errLogin != nil {
		slog.Error("facebook app login failed", "error", errLogin)
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
	h.tickets[ticket] = loginTicket{Result: login, AccountID: result.AccountID, ExpiresAt: time.Now().Add(2 * time.Minute)}
	h.ticketMu.Unlock()
	return c.Redirect().To(redirect + "?ticket=" + url.QueryEscape(ticket))
}

// metaOAuthFailureReason deliberately exposes only the failed OAuth stage.
// The detailed Graph error remains in server logs and credentials are never
// included in the browser redirect.
func metaOAuthFailureReason(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "oauth state"):
		return "The Facebook connection session expired. Start Reconnect Facebook again."
	case strings.Contains(message, "exchange authorization code"):
		return "Meta could not exchange the Facebook authorization. Check the OAuth redirect URI and try again."
	case strings.Contains(message, "exchange long-lived token"):
		return "Meta could not renew the Facebook access token. Reconnect Facebook and approve the requested access."
	case strings.Contains(message, "debug access token"), strings.Contains(message, "access token is invalid"):
		return "Meta returned an invalid Facebook access token. Reconnect Facebook and approve the requested access."
	case strings.Contains(message, "load Facebook Pages"):
		return "Meta authorized Facebook, but Page access could not be loaded. Confirm that Pages are selected in Edit settings."
	case strings.Contains(message, "encrypt Meta access token"), strings.Contains(message, "save encrypted Meta authorization"):
		return "Facebook authorized successfully, but the server could not save the connection."
	default:
		return "Facebook authorization could not be completed. Please try Reconnect Facebook again."
	}
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
