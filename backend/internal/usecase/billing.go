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
		// If no subscription found, return a default Free plan (graceful fallback)
		plans, planErr := u.billing.ListPlans(ctx)
		if planErr == nil && len(plans) > 0 {
			var freePlan *domain.Plan
			for _, p := range plans {
				if p.Price == 0 {
					pCopy := p
					freePlan = &pCopy
					break
				}
			}
			if freePlan != nil {
				return &domain.Subscription{
					AccountID:        accountID,
					PlanID:           freePlan.ID,
					Plan:             freePlan,
					Status:           "active",
					CurrentPeriodEnd: time.Now().AddDate(100, 0, 0), // Lifetime free
				}, nil
			}
		}
		return nil, err
	}
	return sub, nil
}

type CheckoutInput struct {
	PlanID        string `json:"planId"`
	PaymentMethod string `json:"paymentMethod"`
	Reference     string `json:"reference"` // E.g., Bank transfer receipt ID
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

	// Create a transaction record (In a real system, this might be 'pending' until admin approves)
	txn := domain.Transaction{
		ID:            uuid.NewString(),
		AccountID:     accountID,
		Amount:        selectedPlan.Price,
		Currency:      "LAK", // Defaulting to local currency
		Status:        "pending",
		PaymentMethod: in.PaymentMethod,
		Reference:     in.Reference,
	}
	
	if err := u.billing.CreateTransaction(ctx, &txn); err != nil {
		return err
	}

	// For manual billing (like BCEL transfers), the subscription status would normally only update
	// after an Admin verifies the transaction. But for MVP simulation, we'll auto-approve it.
	
	// Ensure we extend from CurrentPeriodEnd if active, else from Now
	sub, _ := u.billing.GetSubscription(ctx, accountID)
	var newEnd time.Time
	if sub != nil && sub.Status == "active" && sub.CurrentPeriodEnd.After(time.Now()) {
		newEnd = sub.CurrentPeriodEnd.AddDate(0, 1, 0) // Add 1 month
	} else {
		newEnd = time.Now().AddDate(0, 1, 0)
	}

	newSub := domain.Subscription{
		ID:               uuid.NewString(),
		AccountID:        accountID,
		PlanID:           selectedPlan.ID,
		Status:           "active",
		CurrentPeriodEnd: newEnd,
	}
	
	if sub != nil {
		newSub.ID = sub.ID // Update existing
	}

	if err := u.billing.UpsertSubscription(ctx, &newSub); err != nil {
		return err
	}
	
	// Mark txn as completed (simulation)
	return u.billing.UpdateTransactionStatus(ctx, txn.ID, "completed")
}
