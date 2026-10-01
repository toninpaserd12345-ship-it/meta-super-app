package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&UserModel{}, &AccountModel{}, &MembershipModel{}, &MembershipClaimModel{}, &PlanModel{}, &SubscriptionModel{}, &TransactionModel{}, &ReplySetModel{}, &ReplyItemModel{}, &AutomationRuleModel{}, &ProductModel{})
}
