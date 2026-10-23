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
		// New account: initialize 7-Day Free Trial of Pro Plan
		plans, planErr := u.billing.ListPlans(ctx)
		if planErr == nil && len(plans) > 0 {
			var proPlan *domain.Plan
			var freePlan *domain.Plan
			for _, p := range plans {
				if p.ID == "pro" {
					pCopy := p
					proPlan = &pCopy
				}
				if p.Price == 0 || p.ID == "free" {
					pCopy := p
					freePlan = &pCopy
				}
			}
			if proPlan != nil {
				trialSub := domain.Subscription{
					ID:               uuid.NewString(),
					AccountID:        accountID,
					PlanID:           proPlan.ID,
					Plan:             proPlan,
					Status:           "trialing",
					CurrentPeriodEnd: time.Now().AddDate(0, 0, 7), // 7 days free trial
					CreatedAt:        time.Now(),
				}
				_ = u.billing.UpsertSubscription(ctx, &trialSub)
				return &trialSub, nil
			}
			if freePlan != nil {
				return &domain.Subscription{
					AccountID:        accountID,
					PlanID:           freePlan.ID,
					Plan:             freePlan,
					Status:           "active",
					CurrentPeriodEnd: time.Now().AddDate(100, 0, 0),
				}, nil
			}
		}
		return nil, err
	}

	// Check if trial expired
	if sub.Status == "trialing" && time.Now().After(sub.CurrentPeriodEnd) {
		sub.Status = "expired"
		_ = u.billing.UpsertSubscription(ctx, sub)
	}

	return sub, nil
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

	// Free plan: instant activation
	if selectedPlan.Price == 0 {
		newSub := domain.Subscription{
			ID:               uuid.NewString(),
			AccountID:        accountID,
			PlanID:           selectedPlan.ID,
			Status:           "active",
			CurrentPeriodEnd: time.Now().AddDate(100, 0, 0),
		}
		if sub, _ := u.billing.GetSubscription(ctx, accountID); sub != nil && sub.ID != "" {
			newSub.ID = sub.ID
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
		Amount:        selectedPlan.Price,
		Currency:      "LAK",
		Status:        "pending",
		PaymentMethod: method,
		Reference:     in.Reference,
	}

	return u.billing.CreateTransaction(ctx, &txn)
}
