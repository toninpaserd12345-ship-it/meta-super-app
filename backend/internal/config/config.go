package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment, HTTPAddr, StorageDriver, DatabaseURL, JWTSecret, JWTIssuer  string
	MetaAppID, MetaAppSecret, MetaRedirectURI, MetaFrontendRedirect          string
	MetaGraphVersion, MetaWebhookFields, MetaWebhookVerifyToken              string
	MetaStateFile, MetaWhatsAppConfigID                                      string
	AccessTokenTTL                                                           time.Duration
	AutoMigrate                                                              bool
	SeedAdminEmail, SeedAdminPassword, SeedAccountName                       string
	R2AccountID, R2AccessKeyID, R2SecretAccessKey, R2BucketName, R2PublicURL string
}

func Load() (Config, error) {
	_ = godotenv.Load()
	ttl, err := time.ParseDuration(value("ACCESS_TOKEN_TTL", "8h"))
	if err != nil {
		return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL: %w", err)
	}
	auto, err := strconv.ParseBool(value("AUTO_MIGRATE", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("AUTO_MIGRATE: %w", err)
	}
	cfg := Config{
		Environment:            value("APP_ENV", "development"),
		HTTPAddr:               value("HTTP_ADDR", ":8080"),
		StorageDriver:          "postgres",
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		JWTIssuer:              value("JWT_ISSUER", "meta-super-app"),
		AccessTokenTTL:         ttl,
		AutoMigrate:            true,
		SeedAdminEmail:         os.Getenv("SEED_ADMIN_EMAIL"),
		SeedAdminPassword:      os.Getenv("SEED_ADMIN_PASSWORD"),
		SeedAccountName:        value("SEED_ACCOUNT_NAME", "Demo Company"),
		MetaAppID:              os.Getenv("META_APP_ID"),
		MetaAppSecret:          os.Getenv("META_APP_SECRET"),
		MetaRedirectURI:        value("META_REDIRECT_URI", "http://localhost:8080/api/v1/meta/oauth/callback"),
		MetaFrontendRedirect:   value("META_FRONTEND_REDIRECT", "http://127.0.0.1:3001/api/auth/facebook/callback"),
		MetaGraphVersion:       value("META_GRAPH_VERSION", "v23.0"),
		MetaWebhookFields:      value("META_WEBHOOK_FIELDS", "messages,messaging_postbacks,messaging_optins,message_deliveries,message_reads,message_echoes,feed"),
		MetaWebhookVerifyToken: os.Getenv("META_WEBHOOK_VERIFY_TOKEN"),
		MetaStateFile:          value("META_STATE_FILE", ".data/meta-state.json"),
		MetaWhatsAppConfigID:   value("META_WHATSAPP_CONFIG_ID", "991544757321935"),
		R2AccountID:            os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKeyID:          os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey:      os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2BucketName:           os.Getenv("R2_BUCKET_NAME"),
		R2PublicURL:            os.Getenv("R2_PUBLIC_URL"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required in .env")
	}
	if cfg.MetaAppID == "" {
		return Config{}, fmt.Errorf("META_APP_ID is required in .env")
	}
	if cfg.MetaAppSecret == "" {
		return Config{}, fmt.Errorf("META_APP_SECRET is required in .env")
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
