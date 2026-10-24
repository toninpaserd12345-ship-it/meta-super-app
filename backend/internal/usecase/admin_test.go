package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meta-super-app/backend/internal/domain"
)

type mockAdminRepo struct {
	txns     map[string]*domain.Transaction
	tenants  map[string]string // accountID -> status
	isSuper  bool
	overview *domain.PlatformOverview
}

func (m *mockAdminRepo) GetPlatformOverview(ctx context.Context) (*domain.PlatformOverview, error) {
	if m.overview != nil {
		return m.overview, nil
	}
	return &domain.PlatformOverview{TotalTenants: 10, ActiveTenants: 8, SuspendedTenants: 2, ProSubscribers: 5, FreeSubscribers: 5}, nil
}

func (m *mockAdminRepo) ListTenants(ctx context.Context, search string) ([]domain.TenantDetail, error) {
	var list []domain.TenantDetail
	for id, status := range m.tenants {
		list = append(list, domain.TenantDetail{ID: id, Name: "Shop " + id, Status: status})
	}
	return list, nil
}

func (m *mockAdminRepo) SetTenantStatus(ctx context.Context, accountID, status string) error {
	m.tenants[accountID] = status
	return nil
}

func (m *mockAdminRepo) DeleteTenant(ctx context.Context, accountID string) error {
	delete(m.tenants, accountID)
	return nil
}

func (m *mockAdminRepo) ListAdminTransactions(ctx context.Context, status string) ([]domain.Transaction, error) {
	var list []domain.Transaction
	for _, txn := range m.txns {
		if status == "" || txn.Status == status {
			list = append(list, *txn)
		}
	}
	return list, nil
}

func (m *mockAdminRepo) GetTransaction(ctx context.Context, txnID string) (*domain.Transaction, error) {
	txn, ok := m.txns[txnID]
	if !ok {
		return nil, errors.New("not found")
	}
	return txn, nil
}

func (m *mockAdminRepo) ApproveTransaction(ctx context.Context, txnID string) error {
	txn, ok := m.txns[txnID]
	if !ok {
		return errors.New("not found")
	}
	txn.Status = "completed"
	return nil
}

func (m *mockAdminRepo) RejectTransaction(ctx context.Context, txnID, reason string) error {
	txn, ok := m.txns[txnID]
	if !ok {
		return errors.New("not found")
	}
	txn.Status = "failed"
	txn.RejectionReason = reason
	return nil
}

func (m *mockAdminRepo) ListAllPages(ctx context.Context) ([]domain.GlobalPageItem, error) {
	return []domain.GlobalPageItem{}, nil
}

func (m *mockAdminRepo) ListAllUsers(ctx context.Context) ([]domain.GlobalUserItem, error) {
	return []domain.GlobalUserItem{}, nil
}

func (m *mockAdminRepo) IsUserSuperAdmin(ctx context.Context, userID string) (bool, error) {
	return m.isSuper, nil
}

type mockBillingRepo struct {
	subs map[string]*domain.Subscription
}

func (b *mockBillingRepo) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	return []domain.Plan{
		{ID: "free", Name: "Free Plan", Price: 0, MaxUsers: 3, MaxPages: 1, IsActive: true},
		{ID: "pro", Name: "Pro Plan", Price: 500000, MaxUsers: 10, MaxPages: 10, IsActive: true},
	}, nil
}

func (b *mockBillingRepo) GetSubscription(ctx context.Context, accountID string) (*domain.Subscription, error) {
	sub, ok := b.subs[accountID]
	if !ok {
		return nil, errors.New("subscription not found")
	}
	return sub, nil
}

func (b *mockBillingRepo) UpsertSubscription(ctx context.Context, sub *domain.Subscription) error {
	b.subs[sub.AccountID] = sub
	return nil
}

func (b *mockBillingRepo) CreateTransaction(ctx context.Context, txn *domain.Transaction) error {
	return nil
}

func (b *mockBillingRepo) UpdateTransactionStatus(ctx context.Context, txnID string, status string) error {
	return nil
}

func (b *mockBillingRepo) ListTransactions(ctx context.Context, accountID string) ([]domain.Transaction, error) {
	return nil, nil
}

func TestAdminApproveTransaction(t *testing.T) {
	ctx := context.Background()
	adminRepo := &mockAdminRepo{
		txns: map[string]*domain.Transaction{
			"tx-1": {
				ID:        "tx-1",
				AccountID: "acc-1",
				PlanID:    "pro",
				Amount:    500000,
				Status:    "pending",
			},
		},
		tenants: map[string]string{"acc-1": "suspended"},
	}
	billingRepo := &mockBillingRepo{
		subs: map[string]*domain.Subscription{
			"acc-1": {
				ID:               "sub-1",
				AccountID:        "acc-1",
				PlanID:           "pro",
				Status:           "expired",
				CurrentPeriodEnd: time.Now().Add(-24 * time.Hour),
			},
		},
	}

	admin := NewAdmin(adminRepo, billingRepo, nil, nil, nil)

	err := admin.ApproveTransaction(ctx, "tx-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify transaction completed
	txn, _ := adminRepo.GetTransaction(ctx, "tx-1")
	if txn.Status != "completed" {
		t.Errorf("txn status = %s, want completed", txn.Status)
	}

	// Verify subscription extended and active
	sub, _ := billingRepo.GetSubscription(ctx, "acc-1")
	if sub.Status != "active" {
		t.Errorf("sub status = %s, want active", sub.Status)
	}
	if !sub.CurrentPeriodEnd.After(time.Now()) {
		t.Errorf("sub CurrentPeriodEnd = %v, want after now", sub.CurrentPeriodEnd)
	}

	// Verify tenant reactivated from suspended
	if adminRepo.tenants["acc-1"] != "active" {
		t.Errorf("tenant status = %s, want active", adminRepo.tenants["acc-1"])
	}
}

func TestAdminRejectTransaction(t *testing.T) {
	ctx := context.Background()
	adminRepo := &mockAdminRepo{
		txns: map[string]*domain.Transaction{
			"tx-2": {
				ID:        "tx-2",
				AccountID: "acc-2",
				PlanID:    "pro",
				Amount:    500000,
				Status:    "pending",
			},
		},
		tenants: map[string]string{"acc-2": "active"},
	}
	billingRepo := &mockBillingRepo{subs: map[string]*domain.Subscription{}}

	admin := NewAdmin(adminRepo, billingRepo, nil, nil, nil)

	err := admin.RejectTransaction(ctx, "tx-2", "Payment slip is blurry or invalid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	txn, _ := adminRepo.GetTransaction(ctx, "tx-2")
	if txn.Status != "failed" {
		t.Errorf("txn status = %s, want failed", txn.Status)
	}
	if txn.RejectionReason != "Payment slip is blurry or invalid" {
		t.Errorf("txn rejectionReason = %s", txn.RejectionReason)
	}
}
