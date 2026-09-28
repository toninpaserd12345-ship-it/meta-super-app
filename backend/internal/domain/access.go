package domain

import "time"

type Role string
type Permission string

const (
	RoleOwner   Role = "owner"
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleMember  Role = "member"
	RoleViewer  Role = "viewer"

	ClaimAll          Permission = "*"
	ClaimPagesRead    Permission = "pages:read"
	ClaimPagesConnect Permission = "pages:connect"
)

type Account struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	LogoURL   string    `json:"logoUrl,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type Membership struct {
	Account Account      `json:"account"`
	Role    Role         `json:"role"`
	Claims  []Permission `json:"claims"`
}

type User struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Email    string       `json:"email"`
	Password string       `json:"-"`
	Accounts []Membership `json:"accounts"`
}

func (m Membership) Can(required Permission) bool {
	for _, claim := range m.Claims {
		if claim == ClaimAll || claim == required {
			return true
		}
	}
	return false
}

var DefaultRoleClaims = map[Role][]Permission{
	RoleOwner:   {ClaimAll},
	RoleAdmin:   {"dashboard:read", ClaimPagesRead, ClaimPagesConnect, "analytics:read", "customers:read", "customers:create", "customers:update", "customers:delete", "orders:read", "orders:create", "orders:update", "orders:delete", "users:read", "users:invite", "users:update", "users:remove", "settings:read", "settings:update", "billing:read"},
	RoleManager: {"dashboard:read", ClaimPagesRead, ClaimPagesConnect, "analytics:read", "customers:read", "customers:create", "customers:update", "orders:read", "orders:create", "orders:update", "users:read"},
	RoleMember:  {"dashboard:read", "customers:read", "orders:read", "orders:create", "orders:update"},
	RoleViewer:  {"dashboard:read", "analytics:read", "customers:read", "orders:read"},
}
