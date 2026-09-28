package domain

import "context"

type UserRepository interface {
	FindByEmail(context.Context, string) (*User, error)
	FindByID(context.Context, string) (*User, error)
	CreateUser(context.Context, *User) error
}

type AccountRepository interface {
	ListForUser(context.Context, string) ([]Membership, error)
	FindMembership(context.Context, string, string) (*Membership, error)
	ListMembers(ctx context.Context, accountID string) ([]MembershipUser, error)
	AddMember(ctx context.Context, accountID, userID string, role Role, claims []Permission) error
	UpdateMemberRole(ctx context.Context, accountID, userID string, role Role, claims []Permission) error
	RemoveMember(ctx context.Context, accountID, userID string) error
}

// MembershipUser combines Membership and User data for display
type MembershipUser struct {
	Membership
	User *User `json:"user"`
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
