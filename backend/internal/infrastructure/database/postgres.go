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
	return db.AutoMigrate(&UserModel{}, &AccountModel{}, &MembershipModel{}, &MembershipClaimModel{}, &PlanModel{}, &SubscriptionModel{}, &TransactionModel{}, &ReplySetModel{}, &ReplyItemModel{}, &AutomationRuleModel{}, &ProductModel{}, &ChatMessageModel{}, &MetaConnectionModel{}, &MetaPageTokenModel{}, &MetaWhatsAppConnectionModel{})
}

// MigrateMetaCredentials keeps the credential tables compatible with the
// running API even when full AUTO_MIGRATE is disabled in production. These
// migrations only add/adjust columns required to load encrypted Meta tokens;
// without them an older database makes every connected account look missing.
func MigrateMetaCredentials(db *gorm.DB) error {
	return db.AutoMigrate(&MetaConnectionModel{}, &MetaPageTokenModel{}, &MetaWhatsAppConnectionModel{})
}
