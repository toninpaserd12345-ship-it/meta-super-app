package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// World-Class Standard: Connection Pooling Optimization
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)           // Maximum idle connections in the pool
	sqlDB.SetMaxOpenConns(100)          // Maximum open connections to the database
	sqlDB.SetConnMaxLifetime(time.Hour) // Maximum amount of time a connection may be reused

	return db, nil
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&UserModel{}, &AccountModel{}, &MembershipModel{}, &MembershipClaimModel{}, &PlanModel{}, &SubscriptionModel{}, &TransactionModel{}, &ReplySetModel{}, &ReplyItemModel{}, &AutomationRuleModel{}, &ProductModel{}, &ChatMessageModel{}, &MetaConnectionModel{}, &MetaPageTokenModel{}, &MetaWhatsAppConnectionModel{}); err != nil {
		return err
	}
	return migrateReplySetCodes(db)
}

// MigrateMetaCredentials keeps the credential tables compatible with the
// running API even when full AUTO_MIGRATE is disabled in production. These
// migrations only add/adjust columns required to load encrypted Meta tokens;
// without them an older database makes every connected account look missing.
func MigrateMetaCredentials(db *gorm.DB) error {
	return db.AutoMigrate(&MetaConnectionModel{}, &MetaPageTokenModel{}, &MetaWhatsAppConnectionModel{})
}

// MigrateRuntimeSchema applies additive columns needed by code paths that are
// used on every deployment. Production may disable full seed migrations, but
// it must never start with a schema older than the running automation API.
func MigrateRuntimeSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(&AutomationRuleModel{}, &ChatMessageModel{}, &ProductModel{}, &ReplySetModel{}); err != nil {
		return err
	}
	if err := migrateReplySetCodes(db); err != nil {
		return err
	}
	// Older releases stored one row per Ad/Post without a flow identity. Group
	// compatible rows deterministically so the new dashboard shows one
	// automation with many targets and keeps all existing bindings intact.
	return db.Exec(`
		UPDATE automation_rule_models
		SET flow_id = md5(concat_ws('|', account_id::text, page_id, COALESCE(product_id::text, ''), reply_set_id::text, is_active::text)),
			flow_name = CASE WHEN COALESCE(flow_name, '') = '' THEN 'Imported automation' ELSE flow_name END,
			first_message_only = TRUE
		WHERE COALESCE(flow_id, '') = ''
	`).Error
}

// migrateReplySetCodes preserves old Reply Sets while giving every set a
// stable, human-readable code. The partial case-insensitive index prevents
// duplicate codes inside one workspace without blocking a rolling deploy
// while legacy rows are being backfilled.
func migrateReplySetCodes(db *gorm.DB) error {
	if err := db.Exec(`
		UPDATE reply_set_models
		SET code = 'RS-' || upper(substr(replace(id::text, '-', ''), 1, 16))
		WHERE code IS NULL OR trim(code) = ''
	`).Error; err != nil {
		return err
	}
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_reply_set_account_code
		ON reply_set_models (account_id, lower(code))
		WHERE code IS NOT NULL AND trim(code) <> ''
	`).Error
}
