package database

import (
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
	"gorm.io/gorm"
)

type Hasher interface{ Hash(string) (string, error) }

func SeedAdmin(db *gorm.DB, hasher Hasher, email, password, accountName string) error {
	if email == "" || password == "" {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var user UserModel
		if err := tx.Where("email = ?", strings.ToLower(email)).First(&user).Error; err == nil {
			if !user.IsSuperAdmin {
				_ = tx.Model(&user).Update("is_super_admin", true).Error
			}
			var mem MembershipModel
			if tx.Where("user_id = ?", user.ID).First(&mem).Error == nil {
				var sub SubscriptionModel
				if tx.Where("account_id = ?", mem.AccountID).First(&sub).Error != nil {
					_ = tx.Create(&SubscriptionModel{
						ID:               uuid.NewString(),
						AccountID:        mem.AccountID,
						PlanID:           "pro",
						Status:           "active",
						LockedPrice:      500000,
						CurrentPeriodEnd: time.Now().AddDate(100, 0, 0),
					}).Error
				}
			}
			return nil
		}
		hash, err := hasher.Hash(password)
		if err != nil {
			return err
		}
		user = UserModel{ID: "00000000-0000-4000-8000-000000000001", Name: "System Admin", Email: strings.ToLower(email), PasswordHash: hash, IsSuperAdmin: true}
		account := AccountModel{ID: "00000000-0000-4000-8000-000000000002", Name: accountName, Slug: slugify(accountName), Status: "active"}
		membership := MembershipModel{ID: uuid.NewString(), UserID: user.ID, AccountID: account.ID, Role: string(domain.RoleOwner)}
		if err = tx.Create(&user).Error; err != nil {
			return err
		}
		if err = tx.Create(&account).Error; err != nil {
			return err
		}
		if err = tx.Create(&membership).Error; err != nil {
			return err
		}
		for _, claim := range domain.DefaultRoleClaims[domain.RoleOwner] {
			if err = tx.Create(&MembershipClaimModel{ID: uuid.NewString(), MembershipID: membership.ID, Claim: string(claim)}).Error; err != nil {
				return err
			}
		}
		_ = tx.Create(&SubscriptionModel{
			ID:               uuid.NewString(),
			AccountID:        account.ID,
			PlanID:           "pro",
			Status:           "active",
			LockedPrice:      500000,
			CurrentPeriodEnd: time.Now().AddDate(100, 0, 0),
		}).Error
		return nil
	})
}

func SeedPlans(db *gorm.DB) error {
	plans := []PlanModel{
		{ID: "free", Name: "Free Plan", Price: 0, MaxUsers: 3, MaxPages: 1, IsActive: true},
		{ID: "pro", Name: "Pro Plan", Price: 500000, MaxUsers: 10, MaxPages: 10, IsActive: true},
	}
	for _, p := range plans {
		if err := db.Where("id = ?", p.ID).FirstOrCreate(&p).Error; err != nil {
			return err
		}
	}
	return nil
}

func slugify(value string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
