package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
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
	}
	metaConnector.SetAutomationProvider(appProvider)

	// Initialize Chat Stream
	chatRepo := repository.NewGormChat(db)
	chatStream := usecase.NewChatStream(chatRepo)
	metaConnector.SetChatStream(chatStream)
	metaConnector.SetProductRepo(productRepo)

	app := httpx.NewHandler(auth, meta, team, billing, storageUseCase, replyUseCase, automationUseCase, productUseCase, chatStream, tokens, metaMode, cfg.MetaFrontendRedirect).App()
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
}

func (p *appAutomationProvider) FindActiveRuleForTrigger(ctx context.Context, pageID, triggerType, triggerValue string) (*domain.AutomationRule, error) {
	return p.automation.FindActiveRuleForTrigger(ctx, pageID, triggerType, triggerValue)
}

func (p *appAutomationProvider) GetKeywordRules(ctx context.Context, pageID string) ([]domain.AutomationRule, error) {
	return p.automation.GetKeywordRules(ctx, pageID)
}

func (p *appAutomationProvider) GetReplySetItems(ctx context.Context, replySetID string, accountID string) ([]domain.ReplyItem, error) {
	return p.reply.GetReplySetItems(ctx, replySetID, accountID)
}
