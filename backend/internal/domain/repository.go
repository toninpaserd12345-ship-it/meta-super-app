package domain

import "context"

type UserRepository interface {
	FindByEmail(context.Context, string) (*User, error)
	FindByID(context.Context, string) (*User, error)
}

type AccountRepository interface {
	ListForUser(context.Context, string) ([]Membership, error)
	FindMembership(context.Context, string, string) (*Membership, error)
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Compare(hash, password string) error
}

type TokenClaims struct{ UserID string }
type TokenService interface {
	Issue(userID string) (string, error)
	Parse(token string) (*TokenClaims, error)
}
