package repository

import (
	"context"
	"fmt"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
)

func (r *GormRepository) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	var models []database.PlanModel
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	result := make([]domain.Plan, 0, len(models))
	for _, m := range models {
		result = append(result, domain.Plan{
			ID:       m.ID,
			Name:     m.Name,
			Price:    m.Price,
			MaxUsers: m.MaxUsers,
			MaxPages: m.MaxPages,
			IsActive: m.IsActive,
		})
	}
	return result, nil
}

func (r *GormRepository) GetSubscription(ctx context.Context, accountID string) (*domain.Subscription, error) {
	var model database.SubscriptionModel
	if err := r.db.WithContext(ctx).Preload("Plan").Where("account_id = ?", accountID).First(&model).Error; err != nil {
		return nil, mapError(err)
	}

	plan := &domain.Plan{
		ID:       model.Plan.ID,
		Name:     model.Plan.Name,
		Price:    model.Plan.Price,
		MaxUsers: model.Plan.MaxUsers,
		MaxPages: model.Plan.MaxPages,
		IsActive: model.Plan.IsActive,
	}

	return &domain.Subscription{
		ID:               model.ID,
		AccountID:        model.AccountID,
		PlanID:           model.PlanID,
		Plan:             plan,
		Status:           model.Status,
		CurrentPeriodEnd: model.CurrentPeriodEnd,
		CreatedAt:        model.CreatedAt,
	}, nil
}

func (r *GormRepository) UpsertSubscription(ctx context.Context, sub *domain.Subscription) error {
	model := database.SubscriptionModel{
		ID:               sub.ID,
		AccountID:        sub.AccountID,
		PlanID:           sub.PlanID,
		Status:           sub.Status,
		CurrentPeriodEnd: sub.CurrentPeriodEnd,
	}

	// Create or Update (Upsert based on AccountID which is a unique index or ID)
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return fmt.Errorf("upsert subscription: %w", err)
	}
	return nil
}

func (r *GormRepository) CreateTransaction(ctx context.Context, txn *domain.Transaction) error {
	model := database.TransactionModel{
		ID:            txn.ID,
		AccountID:     txn.AccountID,
		Amount:        txn.Amount,
		Currency:      txn.Currency,
		Status:        txn.Status,
		PaymentMethod: txn.PaymentMethod,
		Reference:     txn.Reference,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	return nil
}

func (r *GormRepository) UpdateTransactionStatus(ctx context.Context, txnID string, status string) error {
	if err := r.db.WithContext(ctx).Model(&database.TransactionModel{}).Where("id = ?", txnID).Update("status", status).Error; err != nil {
		return fmt.Errorf("update transaction: %w", err)
	}
	return nil
}

func (r *GormRepository) ListTransactions(ctx context.Context, accountID string) ([]domain.Transaction, error) {
	var models []database.TransactionModel
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("created_at desc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	result := make([]domain.Transaction, 0, len(models))
	for _, m := range models {
		result = append(result, domain.Transaction{
			ID:            m.ID,
			AccountID:     m.AccountID,
			Amount:        m.Amount,
			Currency:      m.Currency,
			Status:        m.Status,
			PaymentMethod: m.PaymentMethod,
			Reference:     m.Reference,
			CreatedAt:     m.CreatedAt,
		})
	}
	return result, nil
}
