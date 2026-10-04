package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"

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
	return &domain.User{ID: m.ID, Name: m.Name, Email: m.Email, Password: m.PasswordHash, FacebookID: m.FacebookID}
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
	return fmt.Errorf("RegisterFacebookUser failed: %w", err)
}

func (r *GormRepository) FindByFacebookID(ctx context.Context, fbid string) (*domain.User, error) {
	var m database.UserModel
	if err := r.db.WithContext(ctx).Where("facebook_id = ?", fbid).First(&m).Error; err != nil {
		return nil, mapError(err)
	}
	return &domain.User{ID: m.ID, Name: m.Name, Email: m.Email, Password: m.PasswordHash, FacebookID: m.FacebookID}, nil
}

func (r *GormRepository) RegisterFacebookUser(ctx context.Context, fbid, name, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		u := database.UserModel{
			ID:           uuid.NewString(),
			Name:         name,
			Email:        email,
			FacebookID:   &fbid,
			PasswordHash: "facebook_oauth",
		}
		if err := tx.Create(&u).Error; err != nil {
			return fmt.Errorf("RegisterFacebookUser failed: %w", err)
		}
		a := database.AccountModel{
			ID:   uuid.NewString(),
			Name: name + "'s Store",
			Slug: uuid.NewString(),
		}
		if err := tx.Create(&a).Error; err != nil {
			return fmt.Errorf("RegisterFacebookUser failed: %w", err)
		}
		m := database.MembershipModel{
			ID:        uuid.NewString(),
			UserID:    u.ID,
			AccountID: a.ID,
			Role:      string(domain.RoleOwner),
		}
		if err := tx.Create(&m).Error; err != nil {
			return fmt.Errorf("RegisterFacebookUser failed: %w", err)
		}
		for _, claim := range domain.DefaultRoleClaims[domain.RoleOwner] {
			if err := tx.Create(&database.MembershipClaimModel{
				ID:           uuid.NewString(),
				MembershipID: m.ID,
				Claim:        string(claim),
			}).Error; err != nil {
				return fmt.Errorf("RegisterFacebookUser failed: %w", err)
			}
		}
		user = domain.User{ID: u.ID, Name: u.Name, Email: u.Email, FacebookID: u.FacebookID}
		return nil
	})
	return &user, err
}
