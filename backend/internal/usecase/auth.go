package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/meta-super-app/backend/internal/domain"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Auth struct {
	users     domain.UserRepository
	accounts  domain.AccountRepository
	passwords domain.PasswordHasher
	tokens    domain.TokenService
}

func NewAuth(users domain.UserRepository, accounts domain.AccountRepository, passwords domain.PasswordHasher, tokens domain.TokenService) *Auth {
	return &Auth{users: users, accounts: accounts, passwords: passwords, tokens: tokens}
}

type LoginInput struct{ Email, Password string }
type LoginOutput struct {
	AccessToken string       `json:"access_token"`
	User        *domain.User `json:"user"`
}

func (u *Auth) Login(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	user, err := u.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil || u.passwords.Compare(user.Password, in.Password) != nil {
		return nil, ErrInvalidCredentials
	}
	user.Accounts, err = u.accounts.ListForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	token, err := u.tokens.Issue(user.ID)
	if err != nil {
		return nil, err
	}
	return &LoginOutput{AccessToken: token, User: user}, nil
}

func (u *Auth) LoginByUserID(ctx context.Context, userID string) (*LoginOutput, error) {
	user, err := u.Me(ctx, userID)
	if err != nil { return nil, err }
	token, err := u.tokens.Issue(user.ID)
	if err != nil { return nil, err }
	return &LoginOutput{AccessToken: token, User: user}, nil
}

func (u *Auth) Me(ctx context.Context, userID string) (*domain.User, error) {
	user, err := u.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.Accounts, err = u.accounts.ListForUser(ctx, user.ID)
	return user, err
}

func (u *Auth) Membership(ctx context.Context, userID, accountID string) (*domain.Membership, error) {
	return u.accounts.FindMembership(ctx, userID, accountID)
}
