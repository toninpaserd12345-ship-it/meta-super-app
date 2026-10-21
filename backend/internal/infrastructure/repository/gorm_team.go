package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
)

func (r *GormRepository) CreateUser(ctx context.Context, user *domain.User) error {
	model := database.UserModel{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.Password,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *GormRepository) ListMembers(ctx context.Context, accountID string) ([]domain.MembershipUser, error) {
	var models []database.MembershipModel
	if err := r.db.WithContext(ctx).Preload("Account").Preload("Claims").Where("account_id = ?", accountID).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	// We need to fetch the users for these memberships
	var userIDs []string
	for _, m := range models {
		userIDs = append(userIDs, m.UserID)
	}

	var userModels []database.UserModel
	if len(userIDs) > 0 {
		if err := r.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&userModels).Error; err != nil {
			return nil, fmt.Errorf("list users for members: %w", err)
		}
	}

	userMap := make(map[string]*domain.User)
	for _, u := range userModels {
		userMap[u.ID] = mapUser(u)
	}

	result := make([]domain.MembershipUser, 0, len(models))
	for _, model := range models {
		mu := domain.MembershipUser{
			Membership: mapMembership(model),
			User:       userMap[model.UserID],
		}
		result = append(result, mu)
	}
	return result, nil
}

func (r *GormRepository) AddMember(ctx context.Context, accountID, userID string, role domain.Role, claims []domain.Permission) error {
	membership := database.MembershipModel{
		ID:        uuid.NewString(),
		UserID:    userID,
		AccountID: accountID,
		Role:      string(role),
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&membership).Error; err != nil {
			return err
		}
		for _, claim := range claims {
			c := database.MembershipClaimModel{
				ID:           uuid.NewString(),
				MembershipID: membership.ID,
				Claim:        string(claim),
			}
			if err := tx.Create(&c).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GormRepository) UpdateMemberRole(ctx context.Context, accountID, userID string, role domain.Role, claims []domain.Permission) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var membership database.MembershipModel
		if err := tx.Where("account_id = ? AND user_id = ?", accountID, userID).First(&membership).Error; err != nil {
			return err
		}

		membership.Role = string(role)
		if err := tx.Save(&membership).Error; err != nil {
			return err
		}

		// Delete old claims
		if err := tx.Where("membership_id = ?", membership.ID).Delete(&database.MembershipClaimModel{}).Error; err != nil {
			return err
		}

		// Insert new claims
		for _, claim := range claims {
			c := database.MembershipClaimModel{
				ID:           uuid.NewString(),
				MembershipID: membership.ID,
				Claim:        string(claim),
			}
			if err := tx.Create(&c).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GormRepository) RemoveMember(ctx context.Context, accountID, userID string) error {
	return r.db.WithContext(ctx).Where("account_id = ? AND user_id = ?", accountID, userID).Delete(&database.MembershipModel{}).Error
}

func (r *GormRepository) EnsureDefaultAccount(ctx context.Context, userID, name string) error {
	trimmedName := strings.TrimSpace(name)
	accountName := "My Store"
	if trimmedName != "" {
		accountName = trimmedName + "'s Store"
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&database.MembershipModel{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
			return fmt.Errorf("check memberships: %w", err)
		}
		if count > 0 {
			return nil
		}

		account := database.AccountModel{
			ID:   uuid.NewString(),
			Name: accountName,
			Slug: uuid.NewString(),
		}
		if err := tx.Create(&account).Error; err != nil {
			return fmt.Errorf("create default account: %w", err)
		}

		membership := database.MembershipModel{
			ID:        uuid.NewString(),
			UserID:    userID,
			AccountID: account.ID,
			Role:      string(domain.RoleOwner),
		}
		if err := tx.Create(&membership).Error; err != nil {
			return fmt.Errorf("create default membership: %w", err)
		}

		for _, claim := range domain.DefaultRoleClaims[domain.RoleOwner] {
			if err := tx.Create(&database.MembershipClaimModel{
				ID:           uuid.NewString(),
				MembershipID: membership.ID,
				Claim:        string(claim),
			}).Error; err != nil {
				return fmt.Errorf("create default claims: %w", err)
			}
		}
		return nil
	})
}

