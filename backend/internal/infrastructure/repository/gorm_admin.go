package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
)

func (r *GormRepository) GetPlatformOverview(ctx context.Context) (*domain.PlatformOverview, error) {
	var totalTenants, activeTenants, suspendedTenants int64
	var proSubscribers, trialSubscribers, pendingApprovals int64
	var totalRevenue int64

	now := time.Now()

	// Tenant stats
	_ = r.db.WithContext(ctx).Model(&database.AccountModel{}).Count(&totalTenants).Error
	_ = r.db.WithContext(ctx).Model(&database.AccountModel{}).Where("status = ?", "active").Count(&activeTenants).Error
	_ = r.db.WithContext(ctx).Model(&database.AccountModel{}).Where("status = ?", "suspended").Count(&suspendedTenants).Error

	// Subscription stats
	_ = r.db.WithContext(ctx).Model(&database.SubscriptionModel{}).
		Where("plan_id = ? AND status = ? AND current_period_end > ?", "pro", "active", now).
		Count(&proSubscribers).Error

	_ = r.db.WithContext(ctx).Model(&database.SubscriptionModel{}).
		Where("status = ? AND current_period_end > ?", "trialing", now).
		Count(&trialSubscribers).Error

	freeSubscribers := totalTenants - (proSubscribers + trialSubscribers)
	if freeSubscribers < 0 {
		freeSubscribers = 0
	}

	// Transaction stats
	_ = r.db.WithContext(ctx).Model(&database.TransactionModel{}).Where("status = ?", "pending").Count(&pendingApprovals).Error
	_ = r.db.WithContext(ctx).Model(&database.TransactionModel{}).Where("status = ?", "completed").Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue).Error

	return &domain.PlatformOverview{
		TotalTenants:     totalTenants,
		ActiveTenants:    activeTenants,
		SuspendedTenants: suspendedTenants,
		ProSubscribers:   proSubscribers,
		TrialSubscribers: trialSubscribers,
		FreeSubscribers:  freeSubscribers,
		PendingApprovals: pendingApprovals,
		TotalRevenue:     totalRevenue,
	}, nil
}

func (r *GormRepository) ListTenants(ctx context.Context, search string) ([]domain.TenantDetail, error) {
	query := r.db.WithContext(ctx).Model(&database.AccountModel{})
	if strings.TrimSpace(search) != "" {
		searchTerm := "%" + strings.TrimSpace(search) + "%"
		query = query.Where("name ILIKE ? OR slug ILIKE ?", searchTerm, searchTerm)
	}

	var accounts []database.AccountModel
	if err := query.Order("created_at desc").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}

	result := make([]domain.TenantDetail, 0, len(accounts))
	for _, acc := range accounts {
		// Find Owner
		var ownerMembership database.MembershipModel
		var ownerName, ownerEmail string
		if err := r.db.WithContext(ctx).Preload("User").
			Where("account_id = ? AND role = ?", acc.ID, string(domain.RoleOwner)).
			First(&ownerMembership).Error; err == nil {
			ownerName = ownerMembership.User.Name
			ownerEmail = ownerMembership.User.Email
		}

		// Count users
		var userCount int64
		_ = r.db.WithContext(ctx).Model(&database.MembershipModel{}).Where("account_id = ?", acc.ID).Count(&userCount).Error

		// Count connected pages
		var connectedPagesCount int64
		_ = r.db.WithContext(ctx).Model(&database.MetaPageTokenModel{}).Where("account_id = ?", acc.ID).Count(&connectedPagesCount).Error

		// Get subscription
		var sub database.SubscriptionModel
		planID := "free"
		planName := "Free Plan"
		planStatus := "active"
		var currentPeriodEnd *time.Time

		if err := r.db.WithContext(ctx).Preload("Plan").Where("account_id = ?", acc.ID).First(&sub).Error; err == nil {
			planID = sub.PlanID
			if sub.Plan.Name != "" {
				planName = sub.Plan.Name
			}
			planStatus = sub.Status
			end := sub.CurrentPeriodEnd
			currentPeriodEnd = &end
		}

		status := acc.Status
		if status == "" {
			status = "active"
		}

		result = append(result, domain.TenantDetail{
			ID:                  acc.ID,
			Name:                acc.Name,
			Slug:                acc.Slug,
			Status:              status,
			OwnerName:           ownerName,
			OwnerEmail:          ownerEmail,
			UserCount:           userCount,
			ConnectedPagesCount: connectedPagesCount,
			PlanID:              planID,
			PlanName:            planName,
			PlanStatus:          planStatus,
			CurrentPeriodEnd:    currentPeriodEnd,
			CreatedAt:           acc.CreatedAt,
		})
	}

	return result, nil
}

func (r *GormRepository) SetTenantStatus(ctx context.Context, accountID, status string) error {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "active" && status != "suspended" {
		return fmt.Errorf("invalid tenant status: %s", status)
	}
	if err := r.db.WithContext(ctx).Model(&database.AccountModel{}).Where("id = ?", accountID).Update("status", status).Error; err != nil {
		return fmt.Errorf("set tenant status: %w", err)
	}
	return nil
}

func (r *GormRepository) DeleteTenant(ctx context.Context, accountID string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", accountID).Delete(&database.AccountModel{}).Error; err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	return nil
}

func (r *GormRepository) ListAdminTransactions(ctx context.Context, status string) ([]domain.Transaction, error) {
	query := r.db.WithContext(ctx).Preload("Account")
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}

	var models []database.TransactionModel
	if err := query.Order("created_at desc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list admin transactions: %w", err)
	}

	result := make([]domain.Transaction, 0, len(models))
	for _, m := range models {
		accName := m.Account.Name
		result = append(result, domain.Transaction{
			ID:              m.ID,
			AccountID:       m.AccountID,
			AccountName:     accName,
			PlanID:          m.PlanID,
			Amount:          m.Amount,
			Currency:        m.Currency,
			Status:          m.Status,
			PaymentMethod:   m.PaymentMethod,
			Reference:       m.Reference,
			RejectionReason: m.RejectionReason,
			CreatedAt:       m.CreatedAt,
		})
	}
	return result, nil
}

func (r *GormRepository) GetTransaction(ctx context.Context, txnID string) (*domain.Transaction, error) {
	var m database.TransactionModel
	if err := r.db.WithContext(ctx).Preload("Account").Where("id = ?", txnID).First(&m).Error; err != nil {
		return nil, mapError(err)
	}

	return &domain.Transaction{
		ID:              m.ID,
		AccountID:       m.AccountID,
		AccountName:     m.Account.Name,
		PlanID:          m.PlanID,
		Amount:          m.Amount,
		Currency:        m.Currency,
		Status:          m.Status,
		PaymentMethod:   m.PaymentMethod,
		Reference:       m.Reference,
		RejectionReason: m.RejectionReason,
		CreatedAt:       m.CreatedAt,
	}, nil
}

func (r *GormRepository) ApproveTransaction(ctx context.Context, txnID string) error {
	if err := r.db.WithContext(ctx).Model(&database.TransactionModel{}).
		Where("id = ?", txnID).
		Updates(map[string]interface{}{
			"status": "completed",
		}).Error; err != nil {
		return fmt.Errorf("approve transaction: %w", err)
	}
	return nil
}

func (r *GormRepository) RejectTransaction(ctx context.Context, txnID, reason string) error {
	if err := r.db.WithContext(ctx).Model(&database.TransactionModel{}).
		Where("id = ?", txnID).
		Updates(map[string]interface{}{
			"status":           "failed",
			"rejection_reason": reason,
		}).Error; err != nil {
		return fmt.Errorf("reject transaction: %w", err)
	}
	return nil
}

func (r *GormRepository) IsUserSuperAdmin(ctx context.Context, userID string) (bool, error) {
	var user database.UserModel
	if err := r.db.WithContext(ctx).Select("id, is_super_admin").Where("id = ?", userID).First(&user).Error; err != nil {
		return false, mapError(err)
	}
	return user.IsSuperAdmin, nil
}
