package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/meta-super-app/backend/internal/domain"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type cachedMembership struct {
	membership *domain.Membership
	expiresAt  int64
}

type Auth struct {
	users     domain.UserRepository
	accounts  domain.AccountRepository
	passwords domain.PasswordHasher
	tokens    domain.TokenService

	// In-memory cache to prevent DB overload from middleware
	memCache sync.Map
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
	if err != nil {
		return nil, err
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

func (u *Auth) Me(ctx context.Context, userID string) (*domain.User, error) {
	user, err := u.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.Accounts, err = u.accounts.ListForUser(ctx, user.ID)
	return user, err
}

func (u *Auth) Membership(ctx context.Context, userID, accountID string) (*domain.Membership, error) {
	cacheKey := userID + ":" + accountID
	now := time.Now().Unix()

	// 1. Try to read from fast in-memory cache
	if val, ok := u.memCache.Load(cacheKey); ok {
		cached := val.(cachedMembership)
		if now < cached.expiresAt {
			if cached.membership == nil {
				return nil, errors.New("membership not found (cached)")
			}
			return cached.membership, nil
		}
	}

	// 2. Cache miss or expired, fetch from database
	membership, err := u.accounts.FindMembership(ctx, userID, accountID)

	// 3. Save to cache with 10 seconds TTL
	// This reduces DB load by ~99% if a user makes 100 requests in 10s,
	// while still keeping permission revocation "near-instant" (max 10s delay).
	if err != nil {
		u.memCache.Store(cacheKey, cachedMembership{membership: nil, expiresAt: now + 10})
		return nil, err
	}

	u.memCache.Store(cacheKey, cachedMembership{membership: membership, expiresAt: now + 10})
	return membership, nil
}

func (u *Auth) LoginOrCreateByFacebook(ctx context.Context, fbid, name, email string) (*LoginOutput, error) {
	user, err := u.users.FindByFacebookID(ctx, fbid)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			user, err = u.users.RegisterFacebookUser(ctx, fbid, name, email)
			if err != nil {
				return nil, fmt.Errorf("failed to register fb user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to find by fb id: %w", err)
		}
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
