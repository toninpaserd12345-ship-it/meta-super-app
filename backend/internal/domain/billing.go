package domain

import (
	"context"
	"time"
)

type Plan struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Price    int    `json:"price"` // local currency
	MaxUsers int    `json:"maxUsers"`
	MaxPages int    `json:"maxPages"`
	IsActive bool   `json:"isActive"`
}

type Subscription struct {
	ID               string    `json:"id"`
	AccountID        string    `json:"accountId"`
	PlanID           string    `json:"planId"`
	Plan             *Plan     `json:"plan,omitempty"`
	Status           string    `json:"status"` // active, past_due, canceled
	CurrentPeriodEnd time.Time `json:"currentPeriodEnd"`
	CreatedAt        time.Time `json:"createdAt"`
}

type Transaction struct {
	ID            string    `json:"id"`
	AccountID     string    `json:"accountId"`
	Amount        int       `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"` // pending, completed, failed
	PaymentMethod string    `json:"paymentMethod"`
	Reference     string    `json:"reference"`
	CreatedAt     time.Time `json:"createdAt"`
}

type BillingRepository interface {
	ListPlans(ctx context.Context) ([]Plan, error)
	GetSubscription(ctx context.Context, accountID string) (*Subscription, error)
	UpsertSubscription(ctx context.Context, sub *Subscription) error
	CreateTransaction(ctx context.Context, txn *Transaction) error
	UpdateTransactionStatus(ctx context.Context, txnID string, status string) error
	ListTransactions(ctx context.Context, accountID string) ([]Transaction, error)
}
