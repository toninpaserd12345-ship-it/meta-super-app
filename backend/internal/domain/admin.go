package domain

import (
	"context"
	"time"
)

type PlatformOverview struct {
	TotalTenants     int64 `json:"totalTenants"`
	ActiveTenants    int64 `json:"activeTenants"`
	SuspendedTenants int64 `json:"suspendedTenants"`
	ProSubscribers   int64 `json:"proSubscribers"`
	TrialSubscribers int64 `json:"trialSubscribers"`
	FreeSubscribers  int64 `json:"freeSubscribers"`
	PendingApprovals int64 `json:"pendingApprovals"`
	TotalRevenue     int64 `json:"totalRevenue"`
}

type TenantDetail struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Slug                string     `json:"slug"`
	Status              string     `json:"status"` // "active", "suspended"
	OwnerName           string     `json:"ownerName"`
	OwnerEmail          string     `json:"ownerEmail"`
	UserCount           int64      `json:"userCount"`
	ConnectedPagesCount int64      `json:"connectedPagesCount"`
	PlanID              string     `json:"planId"`
	PlanName            string     `json:"planName"`
	PlanStatus          string     `json:"planStatus"`
	CurrentPeriodEnd    *time.Time `json:"currentPeriodEnd,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
}

type AdminRepository interface {
	GetPlatformOverview(ctx context.Context) (*PlatformOverview, error)
	ListTenants(ctx context.Context, search string) ([]TenantDetail, error)
	SetTenantStatus(ctx context.Context, accountID, status string) error
	DeleteTenant(ctx context.Context, accountID string) error
	ListAdminTransactions(ctx context.Context, status string) ([]Transaction, error)
	GetTransaction(ctx context.Context, txnID string) (*Transaction, error)
	ApproveTransaction(ctx context.Context, txnID string) error
	RejectTransaction(ctx context.Context, txnID, reason string) error
	IsUserSuperAdmin(ctx context.Context, userID string) (bool, error)
}
