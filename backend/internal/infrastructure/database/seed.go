package database

import (
	"strings"
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
		var count int64
		if err := tx.Model(&UserModel{}).Where("email = ?", strings.ToLower(email)).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		hash, err := hasher.Hash(password)
		if err != nil {
			return err
		}
		user := UserModel{ID: uuid.NewString(), Name: "System Admin", Email: strings.ToLower(email), PasswordHash: hash}
		account := AccountModel{ID: uuid.NewString(), Name: accountName, Slug: slugify(accountName)}
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
		return nil
	})
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
