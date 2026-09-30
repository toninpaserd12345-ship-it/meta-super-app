package repository

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/meta-super-app/backend/internal/domain"
)

// MemoryRepository is a development adapter implementing the same domain
// contracts as GormRepository. Use cases do not know which adapter is active.
type MemoryRepository struct {
	mu              sync.RWMutex
	users           map[string]domain.User
	replySets       map[string]domain.ReplySet
	automationRules map[string]domain.AutomationRule
}

func NewMemory(passwordHash string) *MemoryRepository {
	// Stable development IDs keep local JWT sessions valid across restarts.
	userID, accountID := "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002"
	account := domain.Account{ID: accountID, Name: "Demo Company", Slug: "demo-company", CreatedAt: time.Now()}
	user := domain.User{ID: userID, Name: "System Admin", Email: "admin@example.com", Password: passwordHash, Accounts: []domain.Membership{{Account: account, Role: domain.RoleOwner, Claims: append([]domain.Permission(nil), domain.DefaultRoleClaims[domain.RoleOwner]...)}}}
	return &MemoryRepository{users: map[string]domain.User{userID: user}, replySets: make(map[string]domain.ReplySet), automationRules: make(map[string]domain.AutomationRule)}
}

func (r *MemoryRepository) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if strings.EqualFold(user.Email, email) {
			copy := user
			copy.Accounts = nil
			return &copy, nil
		}
	}
	return nil, errors.New("user not found")
}
func (r *MemoryRepository) FindByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	copy := user
	copy.Accounts = nil
	return &copy, nil
}
func (r *MemoryRepository) ListForUser(_ context.Context, userID string) ([]domain.Membership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[userID]
	if !ok {
		return nil, errors.New("user not found")
	}
	return append([]domain.Membership(nil), user.Accounts...), nil
}
func (r *MemoryRepository) FindMembership(ctx context.Context, userID, accountID string) (*domain.Membership, error) {
	memberships, err := r.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, membership := range memberships {
		if membership.Account.ID == accountID {
			copy := membership
			return &copy, nil
		}
	}
	return nil, errors.New("membership not found")
}

func (r *MemoryRepository) CreateUser(ctx context.Context, user *domain.User) error { return nil }
func (r *MemoryRepository) ListMembers(ctx context.Context, accountID string) ([]domain.MembershipUser, error) {
	return nil, nil
}
func (r *MemoryRepository) AddMember(ctx context.Context, accountID, userID string, role domain.Role, claims []domain.Permission) error {
	return nil
}
func (r *MemoryRepository) UpdateMemberRole(ctx context.Context, accountID, userID string, role domain.Role, claims []domain.Permission) error {
	return nil
}
func (r *MemoryRepository) RemoveMember(ctx context.Context, accountID, userID string) error {
	return nil
}

func (r *MemoryRepository) ListPlans(ctx context.Context) ([]domain.Plan, error) { return nil, nil }
func (r *MemoryRepository) GetSubscription(ctx context.Context, accountID string) (*domain.Subscription, error) {
	return nil, nil
}
func (r *MemoryRepository) UpsertSubscription(ctx context.Context, sub *domain.Subscription) error {
	return nil
}
func (r *MemoryRepository) CreateTransaction(ctx context.Context, txn *domain.Transaction) error {
	return nil
}
func (r *MemoryRepository) UpdateTransactionStatus(ctx context.Context, txnID string, status string) error {
	return nil
}
func (r *MemoryRepository) ListTransactions(ctx context.Context, accountID string) ([]domain.Transaction, error) {
	return nil, nil
}
