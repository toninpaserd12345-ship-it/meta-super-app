package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
)

type Billing struct {
	billing domain.BillingRepository
}

func NewBilling(billing domain.BillingRepository) *Billing {
	return &Billing{billing: billing}
}

func (u *Billing) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	return u.billing.ListPlans(ctx)
}

func (u *Billing) GetSubscription(ctx context.Context, accountID string) (*domain.Subscription, error) {
	sub, err := u.billing.GetSubscription(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Check if trial expired
	if sub.Status == "trialing" && time.Now().After(sub.CurrentPeriodEnd) {
		sub.Status = "expired"
		_ = u.billing.UpsertSubscription(ctx, sub)
	}

	if sub.LockedPrice == 0 && sub.Plan != nil && sub.Plan.Price > 0 {
		sub.LockedPrice = sub.Plan.Price
	}

	return sub, nil
}

func (u *Billing) StartTrial(ctx context.Context, accountID string) (*domain.Subscription, error) {
	plans, err := u.billing.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	var proPlan *domain.Plan
	for _, p := range plans {
		if p.ID == "pro" {
			pCopy := p
			proPlan = &pCopy
			break
		}
	}
	if proPlan == nil {
		return nil, errors.New("pro plan not found")
	}

	sub, _ := u.billing.GetSubscription(ctx, accountID)
	subID := uuid.NewString()
	if sub != nil && sub.ID != "" {
		subID = sub.ID
		if sub.Status == "active" {
			return sub, nil
		}
	}

	trialSub := domain.Subscription{
		ID:               subID,
		AccountID:        accountID,
		PlanID:           proPlan.ID,
		Plan:             proPlan,
		Status:           "trialing",
		LockedPrice:      proPlan.Price,
		CurrentPeriodEnd: time.Now().AddDate(0, 0, 7), // 7 days free trial
		CreatedAt:        time.Now(),
	}
	if err := u.billing.UpsertSubscription(ctx, &trialSub); err != nil {
		return nil, err
	}
	return &trialSub, nil
}

func (u *Billing) SelectFree(ctx context.Context, accountID string) (*domain.Subscription, error) {
	plans, err := u.billing.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	var freePlan *domain.Plan
	for _, p := range plans {
		if p.ID == "free" || p.Price == 0 {
			pCopy := p
			freePlan = &pCopy
			break
		}
	}
	if freePlan == nil {
		return nil, errors.New("free plan not found")
	}

	sub, _ := u.billing.GetSubscription(ctx, accountID)
	subID := uuid.NewString()
	if sub != nil && sub.ID != "" {
		subID = sub.ID
	}

	freeSub := domain.Subscription{
		ID:               subID,
		AccountID:        accountID,
		PlanID:           freePlan.ID,
		Plan:             freePlan,
		Status:           "active",
		LockedPrice:      0,
		CurrentPeriodEnd: time.Now().AddDate(100, 0, 0),
		CreatedAt:        time.Now(),
	}
	if err := u.billing.UpsertSubscription(ctx, &freeSub); err != nil {
		return nil, err
	}
	return &freeSub, nil
}

type CheckoutInput struct {
	PlanID        string `json:"planId"`
	PaymentMethod string `json:"paymentMethod"`
	Reference     string `json:"reference"` // BCEL One / OnePay transfer slip URL or ref
}

func (u *Billing) Checkout(ctx context.Context, accountID string, in CheckoutInput) error {
	plans, err := u.billing.ListPlans(ctx)
	if err != nil {
		return err
	}

	var selectedPlan *domain.Plan
	for _, p := range plans {
		if p.ID == in.PlanID {
			selectedPlan = &p
			break
		}
	}
	if selectedPlan == nil {
		return errors.New("invalid plan id")
	}

	// Grandfathered / locked price check:
	// If the store already has a lockedPrice > 0 for this plan, keep their locked price!
	amount := selectedPlan.Price
	existingSub, _ := u.billing.GetSubscription(ctx, accountID)
	if existingSub != nil && existingSub.PlanID == selectedPlan.ID && existingSub.LockedPrice > 0 {
		amount = existingSub.LockedPrice
	}

	// Free plan: instant activation
	if selectedPlan.Price == 0 {
		newSub := domain.Subscription{
			ID:               uuid.NewString(),
			AccountID:        accountID,
			PlanID:           selectedPlan.ID,
			Status:           "active",
			LockedPrice:      0,
			CurrentPeriodEnd: time.Now().AddDate(100, 0, 0),
		}
		if existingSub != nil && existingSub.ID != "" {
			newSub.ID = existingSub.ID
		}
		return u.billing.UpsertSubscription(ctx, &newSub)
	}

	// Paid plan (e.g. Pro via BCEL One):
	// Merchant submits slip, transaction stays "pending" until Super Admin reviews and approves.
	method := in.PaymentMethod
	if method == "" {
		method = "bcel_one"
	}

	txn := domain.Transaction{
		ID:            uuid.NewString(),
		AccountID:     accountID,
		PlanID:        selectedPlan.ID,
		Amount:        amount,
		Currency:      "LAK",
		Status:        "pending",
		PaymentMethod: method,
		Reference:     in.Reference,
	}

	return u.billing.CreateTransaction(ctx, &txn)
}
