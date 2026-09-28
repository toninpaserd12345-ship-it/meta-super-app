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
	if cfg.StorageDriver == "postgres" {
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
		if err = database.SeedAdmin(db, passwords, cfg.SeedAdminEmail, cfg.SeedAdminPassword, cfg.SeedAccountName); err != nil {
			slog.Error("seed failed", "error", err)
			os.Exit(1)
		}
		if err = database.SeedPlans(db); err != nil {
			slog.Error("seed plans failed", "error", err)
			os.Exit(1)
		}
		repo := repository.NewGorm(db)
		users, accounts, billingRepo = repo, repo, repo
	} else {
		mockPassword := cfg.SeedAdminPassword
		if mockPassword == "" {
			mockPassword = "password123"
		}
		hash, hashErr := passwords.Hash(mockPassword)
		if hashErr != nil {
			slog.Error("mock password failed", "error", hashErr)
			os.Exit(1)
		}
		repo := repository.NewMemory(hash)
		users, accounts, billingRepo = repo, repo, repo
	}
	tokens := security.NewJWT(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL)
	auth := usecase.NewAuth(users, accounts, passwords, tokens)
	metaMode := "mock"
	var metaConnector domain.MetaConnector = metainfra.NewMockConnector()
	if cfg.MetaAppID != "" && cfg.MetaAppSecret != "" {
		metaMode = "live"
		metaConnector = metainfra.NewGraphConnector(metainfra.GraphConfig{AppID: cfg.MetaAppID, AppSecret: cfg.MetaAppSecret, RedirectURI: cfg.MetaRedirectURI, Version: cfg.MetaGraphVersion, WebhookFields: cfg.MetaWebhookFields, WebhookVerifyToken: cfg.MetaWebhookVerifyToken, StateFile: cfg.MetaStateFile})
	}
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

	app := httpx.NewHandler(auth, meta, team, billing, storageUseCase, tokens, metaMode, cfg.MetaFrontendRedirect, "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002").App()
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
