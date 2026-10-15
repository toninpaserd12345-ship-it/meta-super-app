package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/meta-super-app/backend/internal/config"
	"github.com/meta-super-app/backend/internal/delivery/httpx"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	metainfra "github.com/meta-super-app/backend/internal/infrastructure/meta"
	"github.com/meta-super-app/backend/internal/infrastructure/repository"
	"github.com/meta-super-app/backend/internal/infrastructure/security"
	"github.com/meta-super-app/backend/internal/infrastructure/storage"
	"github.com/meta-super-app/backend/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	passwords := security.NewBcrypt(12)
	var users domain.UserRepository
	var accounts domain.AccountRepository
	var billingRepo domain.BillingRepository
	var replyRepo domain.ReplyRepository
	var automationRepo domain.AutomationRepository
	var productRepo domain.ProductRepository

	db, openErr := database.Open(cfg.DatabaseURL)
	if openErr != nil {
		slog.Error("database failed", "error", openErr)
		os.Exit(1)
	}
	if cfg.AutoMigrate {
		if err = database.Migrate(db); err != nil {
			slog.Error("migration failed", "error", err)
			os.Exit(1)
		}
	}
	if err = database.MigrateMetaCredentials(db); err != nil {
		slog.Error("Meta credential migration failed", "error", err)
		os.Exit(1)
	}
	if err = database.MigrateRuntimeSchema(db); err != nil {
		slog.Error("runtime schema migration failed", "error", err)
		os.Exit(1)
	}
	if err = database.SeedAdmin(db, passwords, cfg.SeedAdminEmail, cfg.SeedAdminPassword, cfg.SeedAccountName); err != nil {
		slog.Error("seed failed", "error", err)

	}
	if err = database.SeedPlans(db); err != nil {
		slog.Error("seed plans failed", "error", err)

	}
	repo := repository.NewGorm(db)
	users, accounts, billingRepo, replyRepo, automationRepo, productRepo = repo, repo, repo, repo, repo, repo

	tokens := security.NewJWT(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL)
	auth := usecase.NewAuth(users, accounts, passwords, tokens)
	metaMode := "live"
	metaConnector := metainfra.NewGraphConnector(metainfra.GraphConfig{AppID: cfg.MetaAppID, AppSecret: cfg.MetaAppSecret, RedirectURI: cfg.MetaRedirectURI, Version: cfg.MetaGraphVersion, WebhookFields: cfg.MetaWebhookFields, WebhookVerifyToken: cfg.MetaWebhookVerifyToken, StateFile: cfg.MetaStateFile, EncryptionKey: cfg.JWTSecret, WhatsAppConfigID: cfg.MetaWhatsAppConfigID}, db)
	meta := usecase.NewMeta(metaConnector)
	team := usecase.NewTeam(accounts, users, passwords)
	billing := usecase.NewBilling(billingRepo)

	// Initialize Storage (R2)
	ctxR2 := context.Background()
	storageRepo, err := storage.NewR2StorageService(ctxR2, cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2BucketName, cfg.R2PublicURL)
	if err != nil {
		slog.Warn("Storage service not initialized (missing config or error)", "error", err)
	}
	storageUseCase := usecase.NewStorageUseCase(storageRepo)

	// Initialize Reply System
	replyUseCase := usecase.NewReply(replyRepo)

	// Initialize Automation System
	automationUseCase := usecase.NewAutomation(automationRepo)
	productUseCase := usecase.NewProduct(productRepo)

	appProvider := &appAutomationProvider{
		automation: automationUseCase,
		reply:      replyUseCase,
		product:    productUseCase,
	}
	metaConnector.SetAutomationProvider(appProvider)

	// Initialize Chat Database (Dual DB support: can be primary DB or dedicated chat DB)
	chatDB := db
	if cfg.ChatDatabaseURL != "" && cfg.ChatDatabaseURL != cfg.DatabaseURL {
		cdb, chatErr := database.Open(cfg.ChatDatabaseURL)
		if chatErr != nil {
			slog.Warn("dedicated chat database connection failed, falling back to primary db", "error", chatErr)
		} else {
			slog.Info("dedicated chat database connected successfully")
			chatDB = cdb
			if cfg.AutoMigrate {
				if err := chatDB.AutoMigrate(&database.ChatMessageModel{}); err != nil {
					slog.Warn("chat database automigrate failed", "error", err)
				}
			}
		}
	}

	// Initialize Chat Stream
	chatRepo := repository.NewGormChat(chatDB)
	chatStream := usecase.NewChatStream(chatRepo)
	metaConnector.SetChatStream(chatStream)
	metaConnector.SetProductRepo(productRepo)
	leads := usecase.NewLeads(db, metaConnector, chatStream, cfg.MetaGraphVersion)
	metaConnector.SetLeadSink(leads)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			leads.ProcessFollowUps(context.Background())
		}
	}()

	inventoryUseCase := usecase.NewInventory(db)
	ordersUseCase := usecase.NewOrders(db, leads, chatStream, inventoryUseCase)

	app := httpx.NewHandler(auth, meta, team, billing, storageUseCase, replyUseCase, automationUseCase, productUseCase, chatStream, leads, inventoryUseCase, ordersUseCase, tokens, metaMode, cfg.MetaFrontendRedirect).App()
	go func() {
		slog.Info("Fiber API listening", "address", cfg.HTTPAddr, "environment", cfg.Environment, "storage", cfg.StorageDriver, "meta", metaMode)
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			slog.Error("server failed", "error", err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- app.Shutdown() }()
	select {
	case err := <-done:
		if err != nil {
			slog.Error("shutdown failed", "error", err)
		}
	case <-ctx.Done():
		slog.Error("shutdown timed out")
	}
}

type appAutomationProvider struct {
	automation *usecase.Automation
	reply      *usecase.Reply
	product    *usecase.Product
}

func (p *appAutomationProvider) FindActiveRuleForTrigger(ctx context.Context, pageID, triggerType, triggerValue string) (*domain.AutomationRule, error) {
	return p.automation.FindActiveRuleForTrigger(ctx, pageID, triggerType, triggerValue)
}

func (p *appAutomationProvider) GetKeywordRules(ctx context.Context, pageID string) ([]domain.AutomationRule, error) {
	return p.automation.GetKeywordRules(ctx, pageID)
}

func (p *appAutomationProvider) FindReplySetByTrigger(ctx context.Context, accountID, trigger string) (*domain.ReplySet, error) {
	set, err := p.reply.FindSetByTrigger(ctx, accountID, trigger)
	if err == nil && set != nil {
		return set, nil
	}

	// Also check if trigger matches any Product Code or Product Name that has an Automation Flow
	if p.product != nil {
		products, prodErr := p.product.ListProducts(ctx, "", accountID)
		if prodErr == nil {
			triggerLower := strings.ToLower(strings.TrimSpace(trigger))
			strippedLower := strings.ToLower(strings.TrimSpace(strings.TrimLeft(trigger, "/#:!# ")))
			for _, prod := range products {
				prodCodeLower := strings.ToLower(strings.TrimSpace(prod.Code))
				prodNameLower := strings.ToLower(strings.TrimSpace(prod.Name))
				match := triggerLower == strings.ToLower(prod.ID) || triggerLower == prodNameLower || (prodCodeLower != "" && triggerLower == prodCodeLower)
				if !match && strippedLower != "" {
					match = strippedLower == strings.ToLower(prod.ID) || strippedLower == prodNameLower || (prodCodeLower != "" && strippedLower == prodCodeLower)
				}
				// Also match if message contains the product code (e.g. "AG29") or product name (e.g. "kito AG29")
				if !match {
					if prodCodeLower != "" && len(prodCodeLower) >= 2 && strings.Contains(triggerLower, prodCodeLower) {
						match = true
					} else if prodNameLower != "" && len([]rune(prodNameLower)) >= 3 && strings.Contains(triggerLower, prodNameLower) {
						match = true
					}
				}
				if match {
					flows, flowErr := p.automation.GetFlows(ctx, accountID)
					if flowErr == nil {
						for _, f := range flows {
							if f.IsActive && f.ProductID == prod.ID && f.ReplySetID != "" {
								return p.reply.GetSet(ctx, f.ReplySetID, accountID)
							}
						}
					}
					break
				}
			}
		}
	}

	return nil, nil
}

func (p *appAutomationProvider) GetReplySetItems(ctx context.Context, replySetID string, accountID string) ([]domain.ReplyItem, error) {
	return p.reply.GetReplySetItems(ctx, replySetID, accountID)
}

func (p *appAutomationProvider) GetRule(ctx context.Context, id, accountID string) (*domain.AutomationRule, error) {
	return p.automation.GetRule(ctx, id, accountID)
}

