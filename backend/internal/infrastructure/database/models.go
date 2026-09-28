package database

import "time"

type UserModel struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	Name         string `gorm:"size:120;not null"`
	Email        string `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AccountModel struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	Name      string `gorm:"size:160;not null"`
	Slug      string `gorm:"size:100;uniqueIndex;not null"`
	LogoURL   string `gorm:"size:500"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MembershipModel struct {
	ID        string                 `gorm:"type:uuid;primaryKey"`
	UserID    string                 `gorm:"type:uuid;not null;uniqueIndex:idx_user_account"`
	AccountID string                 `gorm:"type:uuid;not null;uniqueIndex:idx_user_account"`
	Role      string                 `gorm:"size:30;not null"`
	Account   AccountModel           `gorm:"foreignKey:AccountID"`
	Claims    []MembershipClaimModel `gorm:"foreignKey:MembershipID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MembershipClaimModel struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	MembershipID string `gorm:"type:uuid;not null;uniqueIndex:idx_membership_claim"`
	Claim        string `gorm:"size:100;not null;uniqueIndex:idx_membership_claim"`
}

type PlanModel struct {
	ID         string `gorm:"size:50;primaryKey"` // e.g., "free", "pro"
	Name       string `gorm:"size:100;not null"`
	Price      int    // Price in local currency (e.g., LAK) or USD cents
	MaxUsers   int
	MaxPages   int
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ReplySetModel struct {
	ID        string           `gorm:"type:uuid;primaryKey"`
	AccountID string           `gorm:"type:uuid;not null;index"`
	Name      string           `gorm:"size:255;not null"`
	Items     []ReplyItemModel `gorm:"foreignKey:ReplySetID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReplyItemModel struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	ReplySetID string `gorm:"type:uuid;not null;index"`
	Type       string `gorm:"size:20;not null"`   // "text", "image", "video", "audio"
	Content    string `gorm:"type:text;not null"` // Text message OR URL of the media
	OrderIndex int    `gorm:"not null"`           // Used for sorting front/back
}

type SubscriptionModel struct {
	ID               string       `gorm:"type:uuid;primaryKey"`
	AccountID        string       `gorm:"type:uuid;uniqueIndex;not null"`
	PlanID           string       `gorm:"size:50;not null"`
	Status           string       `gorm:"size:50;not null"` // e.g., "active", "expired", "canceled"
	CurrentPeriodEnd time.Time    `gorm:"not null"`
	Plan             PlanModel    `gorm:"foreignKey:PlanID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TransactionModel struct {
	ID             string            `gorm:"type:uuid;primaryKey"`
	AccountID      string            `gorm:"type:uuid;not null"`
	Amount         int               `gorm:"not null"`
	Currency       string            `gorm:"size:10;not null"`
	Status         string            `gorm:"size:50;not null"` // e.g., "pending", "completed", "failed"
	PaymentMethod  string            `gorm:"size:50"`
	Reference      string            `gorm:"size:255"`         // Transfer ref or Stripe Session ID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
