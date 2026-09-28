package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment, HTTPAddr, StorageDriver, DatabaseURL, JWTSecret, JWTIssuer string
	MetaAppID, MetaAppSecret, MetaRedirectURI, MetaFrontendRedirect         string
	MetaGraphVersion, MetaWebhookFields, MetaWebhookVerifyToken             string
	MetaStateFile                                                           string
	AccessTokenTTL                                                          time.Duration
	AutoMigrate                                                             bool
	SeedAdminEmail, SeedAdminPassword, SeedAccountName                      string
}

func Load() (Config, error) {
	_ = godotenv.Load()
	ttl, err := time.ParseDuration(value("ACCESS_TOKEN_TTL", "8h"))
	if err != nil {
		return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL: %w", err)
	}
	auto, err := strconv.ParseBool(value("AUTO_MIGRATE", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("AUTO_MIGRATE: %w", err)
	}
	cfg := Config{Environment: value("APP_ENV", "development"), HTTPAddr: value("HTTP_ADDR", ":8080"), StorageDriver: value("STORAGE_DRIVER", "memory"), DatabaseURL: os.Getenv("DATABASE_URL"), JWTSecret: os.Getenv("JWT_SECRET"), JWTIssuer: value("JWT_ISSUER", "meta-super-app"), AccessTokenTTL: ttl, AutoMigrate: auto, SeedAdminEmail: os.Getenv("SEED_ADMIN_EMAIL"), SeedAdminPassword: os.Getenv("SEED_ADMIN_PASSWORD"), SeedAccountName: value("SEED_ACCOUNT_NAME", "Demo Company"), MetaAppID: os.Getenv("META_APP_ID"), MetaAppSecret: os.Getenv("META_APP_SECRET"), MetaRedirectURI: value("META_REDIRECT_URI", "http://localhost:8080/api/v1/meta/oauth/callback"), MetaFrontendRedirect: value("META_FRONTEND_REDIRECT", "http://127.0.0.1:3001/api/auth/facebook/callback"), MetaGraphVersion: value("META_GRAPH_VERSION", "v23.0"), MetaWebhookFields: value("META_WEBHOOK_FIELDS", "feed"), MetaWebhookVerifyToken: os.Getenv("META_WEBHOOK_VERIFY_TOKEN"), MetaStateFile: value("META_STATE_FILE", ".data/meta-state.json")}
	if cfg.StorageDriver != "memory" && cfg.StorageDriver != "postgres" {
		return Config{}, fmt.Errorf("STORAGE_DRIVER must be memory or postgres")
	}
	if cfg.StorageDriver == "postgres" && cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required for postgres storage")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	return cfg, nil
}
func value(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
