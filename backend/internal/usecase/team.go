package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
)

var ErrUnauthorized = errors.New("unauthorized")

type Team struct {
	accounts  domain.AccountRepository
	users     domain.UserRepository
	passwords domain.PasswordHasher
}

func NewTeam(accounts domain.AccountRepository, users domain.UserRepository, passwords domain.PasswordHasher) *Team {
	return &Team{accounts: accounts, users: users, passwords: passwords}
}

type InviteInput struct {
	Email    string      `json:"email"`
	Name     string      `json:"name"`
	Role     domain.Role `json:"role"`
}

type InviteOutput struct {
	User     *domain.User `json:"user"`
	Password string       `json:"password,omitempty"` // Generated password for new users
}

func (u *Team) ListMembers(ctx context.Context, accountID string) ([]domain.MembershipUser, error) {
	return u.accounts.ListMembers(ctx, accountID)
}

func (u *Team) InviteMember(ctx context.Context, accountID string, in InviteInput) (*InviteOutput, error) {
	user, err := u.users.FindByEmail(ctx, in.Email)
	var generatedPassword string
	
	if err != nil { // User does not exist, create a new one
		generatedPassword = uuid.NewString()[:8]
		hash, _ := u.passwords.Hash(generatedPassword)
		user = &domain.User{
			ID:       uuid.NewString(),
			Name:     in.Name,
			Email:    in.Email,
			Password: hash,
		}
		if err := u.users.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	}
	
	claims := domain.DefaultRoleClaims[in.Role]
	if err := u.accounts.AddMember(ctx, accountID, user.ID, in.Role, claims); err != nil {
		return nil, err
	}
	
	return &InviteOutput{
		User:     user,
		Password: generatedPassword,
	}, nil
}

func (u *Team) UpdateRole(ctx context.Context, accountID, userID string, role domain.Role) error {
	claims := domain.DefaultRoleClaims[role]
	return u.accounts.UpdateMemberRole(ctx, accountID, userID, role, claims)
}

func (u *Team) RemoveMember(ctx context.Context, accountID, userID string) error {
	return u.accounts.RemoveMember(ctx, accountID, userID)
}
