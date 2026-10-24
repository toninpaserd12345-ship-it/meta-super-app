package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
)

type Admin struct {
	adminRepo   domain.AdminRepository
	billingRepo domain.BillingRepository
	usersRepo   domain.UserRepository
	passwords   domain.PasswordHasher
	tokens      domain.TokenService
}

func NewAdmin(
	adminRepo domain.AdminRepository,
	billingRepo domain.BillingRepository,
	usersRepo domain.UserRepository,
	passwords domain.PasswordHasher,
	tokens domain.TokenService,
) *Admin {
	return &Admin{
		adminRepo:   adminRepo,
		billingRepo: billingRepo,
		usersRepo:   usersRepo,
		passwords:   passwords,
		tokens:      tokens,
	}
}

func (u *Admin) Overview(ctx context.Context) (*domain.PlatformOverview, error) {
	return u.adminRepo.GetPlatformOverview(ctx)
}

func (u *Admin) ListTenants(ctx context.Context, search string) ([]domain.TenantDetail, error) {
	return u.adminRepo.ListTenants(ctx, search)
}

func (u *Admin) ListAllPages(ctx context.Context) ([]domain.GlobalPageItem, error) {
	return u.adminRepo.ListAllPages(ctx)
}

func (u *Admin) ListAllUsers(ctx context.Context) ([]domain.GlobalUserItem, error) {
	return u.adminRepo.ListAllUsers(ctx)
}

func (u *Admin) SetTenantStatus(ctx context.Context, accountID, status string) error {
	return u.adminRepo.SetTenantStatus(ctx, accountID, status)
}

func (u *Admin) DeleteTenant(ctx context.Context, accountID string) error {
	return u.adminRepo.DeleteTenant(ctx, accountID)
}

func (u *Admin) ListTransactions(ctx context.Context, status string) ([]domain.Transaction, error) {
	return u.adminRepo.ListAdminTransactions(ctx, status)
}

func (u *Admin) ApproveTransaction(ctx context.Context, txnID string) error {
	txn, err := u.adminRepo.GetTransaction(ctx, txnID)
	if err != nil {
		return err
	}
	if txn.Status != "pending" {
		return errors.New("transaction is not pending")
	}

	// 1. Mark transaction as completed
	if err := u.adminRepo.ApproveTransaction(ctx, txnID); err != nil {
		return err
	}

	// 2. Fetch existing subscription to extend
	planID := txn.PlanID
	if planID == "" {
		planID = "pro"
	}

	sub, _ := u.billingRepo.GetSubscription(ctx, txn.AccountID)
	var newEnd time.Time
	now := time.Now()

	if sub != nil && (sub.Status == "active" || sub.Status == "trialing") && sub.CurrentPeriodEnd.After(now) {
		newEnd = sub.CurrentPeriodEnd.AddDate(0, 1, 0) // +30 days / 1 month from current end
	} else {
		newEnd = now.AddDate(0, 1, 0) // +30 days / 1 month from now
	}

	subID := uuid.NewString()
	lockedPrice := txn.Amount
	if sub != nil && sub.ID != "" {
		subID = sub.ID
		if sub.LockedPrice > 0 {
			lockedPrice = sub.LockedPrice
		}
	}

	newSub := domain.Subscription{
		ID:               subID,
		AccountID:        txn.AccountID,
		PlanID:           planID,
		Status:           "active",
		LockedPrice:      lockedPrice,
		CurrentPeriodEnd: newEnd,
	}

	if err := u.billingRepo.UpsertSubscription(ctx, &newSub); err != nil {
		return err
	}

	// 3. Reactivate account if it was suspended
	_ = u.adminRepo.SetTenantStatus(ctx, txn.AccountID, "active")

	return nil
}

func (u *Admin) RejectTransaction(ctx context.Context, txnID, reason string) error {
	txn, err := u.adminRepo.GetTransaction(ctx, txnID)
	if err != nil {
		return err
	}
	if txn.Status != "pending" {
		return errors.New("transaction is not pending")
	}
	return u.adminRepo.RejectTransaction(ctx, txnID, reason)
}

func (u *Admin) IsSuperAdmin(ctx context.Context, userID string) (bool, error) {
	return u.adminRepo.IsUserSuperAdmin(ctx, userID)
}

func (u *Admin) Login(ctx context.Context, email, password string) (*LoginOutput, error) {
	user, err := u.usersRepo.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || u.passwords.Compare(user.Password, password) != nil {
		return nil, ErrInvalidCredentials
	}

	isSuper, _ := u.adminRepo.IsUserSuperAdmin(ctx, user.ID)
	if !isSuper && !user.IsSuperAdmin {
		return nil, errors.New("user does not have super admin privileges")
	}

	token, err := u.tokens.Issue(user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken: token,
		User:        user,
	}, nil
}

func (u *Admin) ListPlans(ctx context.Context) ([]domain.AdminPlanItem, error) {
	return u.adminRepo.ListAdminPlans(ctx)
}

func (u *Admin) SavePlan(ctx context.Context, plan domain.Plan) error {
	if strings.TrimSpace(plan.ID) == "" {
		return errors.New("plan id is required")
	}
	if strings.TrimSpace(plan.Name) == "" {
		return errors.New("plan name is required")
	}
	if plan.Price < 0 {
		return errors.New("price cannot be negative")
	}
	return u.adminRepo.SavePlan(ctx, plan)
}

func (u *Admin) DeletePlan(ctx context.Context, planID string) error {
	if strings.TrimSpace(planID) == "" {
		return errors.New("plan id is required")
	}
	return u.adminRepo.DeletePlan(ctx, planID)
}

func (u *Admin) SetTenantPlan(ctx context.Context, accountID, planID string, days, lockedPrice int, status string) error {
	if strings.TrimSpace(accountID) == "" {
		return errors.New("account id is required")
	}
	if strings.TrimSpace(planID) == "" {
		return errors.New("plan id is required")
	}
	return u.adminRepo.SetTenantPlan(ctx, accountID, planID, days, lockedPrice, status)
}

