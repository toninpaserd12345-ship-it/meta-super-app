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

type TenantUser struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type TenantPage struct {
	PageID      string    `json:"pageId"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	PictureURL  string    `json:"pictureUrl"`
	IsConnected bool      `json:"isConnected"`
	CreatedAt   time.Time `json:"createdAt"`
}

type GlobalPageItem struct {
	PageID      string    `json:"pageId"`
	PageName    string    `json:"pageName"`
	Category    string    `json:"category"`
	PictureURL  string    `json:"pictureUrl"`
	IsConnected bool      `json:"isConnected"`
	AccountID   string    `json:"accountId"`
	StoreName   string    `json:"storeName"`
	CreatedAt   time.Time `json:"createdAt"`
}

type GlobalUserItem struct {
	UserID       string    `json:"userId"`
	UserName     string    `json:"userName"`
	Email        string    `json:"email"`
	IsSuperAdmin bool      `json:"isSuperAdmin"`
	AccountID    string    `json:"accountId"`
	StoreName    string    `json:"storeName"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
}

type TenantDetail struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	Slug                string       `json:"slug"`
	Status              string       `json:"status"` // "active", "suspended"
	OwnerName           string       `json:"ownerName"`
	OwnerEmail          string       `json:"ownerEmail"`
	UserCount           int64        `json:"userCount"`
	ConnectedPagesCount int64        `json:"connectedPagesCount"`
	PlanID              string       `json:"planId"`
	PlanName            string       `json:"planName"`
	PlanStatus          string       `json:"planStatus"`
	CurrentPeriodEnd    *time.Time   `json:"currentPeriodEnd,omitempty"`
	Users               []TenantUser `json:"users"`
	Pages               []TenantPage `json:"pages"`
	CreatedAt           time.Time    `json:"createdAt"`
}

type AdminPlanItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Price           int    `json:"price"`
	MaxUsers        int    `json:"maxUsers"`
	MaxPages        int    `json:"maxPages"`
	IsActive        bool   `json:"isActive"`
	SubscriberCount int64  `json:"subscriberCount"`
}

type AdminRepository interface {
	GetPlatformOverview(ctx context.Context) (*PlatformOverview, error)
	ListTenants(ctx context.Context, search string) ([]TenantDetail, error)
	ListAllPages(ctx context.Context) ([]GlobalPageItem, error)
	ListAllUsers(ctx context.Context) ([]GlobalUserItem, error)
	SetTenantStatus(ctx context.Context, accountID, status string) error
	DeleteTenant(ctx context.Context, accountID string) error
	ListAdminTransactions(ctx context.Context, status string) ([]Transaction, error)
	GetTransaction(ctx context.Context, txnID string) (*Transaction, error)
	ApproveTransaction(ctx context.Context, txnID string) error
	RejectTransaction(ctx context.Context, txnID, reason string) error
	IsUserSuperAdmin(ctx context.Context, userID string) (bool, error)
	ListAdminPlans(ctx context.Context) ([]AdminPlanItem, error)
	SavePlan(ctx context.Context, plan Plan) error
	DeletePlan(ctx context.Context, planID string) error
	SetTenantPlan(ctx context.Context, accountID, planID string, days int, status string) error
}
