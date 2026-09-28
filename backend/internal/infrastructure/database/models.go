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
