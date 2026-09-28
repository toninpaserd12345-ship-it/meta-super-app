package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
)

type GormRepository struct{ db *gorm.DB }

func NewGorm(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (r *GormRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model database.UserModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		return nil, mapError(err)
	}
	return mapUser(model), nil
}

func (r *GormRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var model database.UserModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, mapError(err)
	}
	return mapUser(model), nil
}

func (r *GormRepository) ListForUser(ctx context.Context, userID string) ([]domain.Membership, error) {
	var models []database.MembershipModel
	if err := r.db.WithContext(ctx).Preload("Account").Preload("Claims").Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	result := make([]domain.Membership, 0, len(models))
	for _, model := range models {
		result = append(result, mapMembership(model))
	}
	return result, nil
}

func (r *GormRepository) FindMembership(ctx context.Context, userID, accountID string) (*domain.Membership, error) {
	var model database.MembershipModel
	err := r.db.WithContext(ctx).Preload("Account").Preload("Claims").Where("user_id = ? AND account_id = ?", userID, accountID).First(&model).Error
	if err != nil {
		return nil, mapError(err)
	}
	value := mapMembership(model)
	return &value, nil
}

func mapUser(m database.UserModel) *domain.User {
	return &domain.User{ID: m.ID, Name: m.Name, Email: m.Email, Password: m.PasswordHash}
}
func mapMembership(m database.MembershipModel) domain.Membership {
	claims := make([]domain.Permission, 0, len(m.Claims))
	for _, claim := range m.Claims {
		claims = append(claims, domain.Permission(claim.Claim))
	}
	return domain.Membership{Account: domain.Account{ID: m.Account.ID, Name: m.Account.Name, Slug: m.Account.Slug, LogoURL: m.Account.LogoURL, CreatedAt: m.Account.CreatedAt}, Role: domain.Role(m.Role), Claims: claims}
}
func mapError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("not found: %w", err)
	}
	return err
}
