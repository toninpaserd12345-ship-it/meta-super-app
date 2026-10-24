package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
		lockedPrice := 0
		var currentPeriodEnd *time.Time

		if err := r.db.WithContext(ctx).Preload("Plan").Where("account_id = ?", acc.ID).First(&sub).Error; err == nil {
			planID = sub.PlanID
			if sub.Plan.Name != "" {
				planName = sub.Plan.Name
			}
			planStatus = sub.Status
			lockedPrice = sub.LockedPrice
			if lockedPrice == 0 && sub.Plan.Price > 0 {
				lockedPrice = sub.Plan.Price
			}
			end := sub.CurrentPeriodEnd
			currentPeriodEnd = &end
		}

		status := acc.Status
		if status == "" {
			status = "active"
		}

		// Get users in this tenant
		var memberships []database.MembershipModel
		var tenantUsers []domain.TenantUser
		if err := r.db.WithContext(ctx).Preload("User").Where("account_id = ?", acc.ID).Order("created_at asc").Find(&memberships).Error; err == nil {
			for _, m := range memberships {
				tenantUsers = append(tenantUsers, domain.TenantUser{
					ID:        m.UserID,
					Name:      m.User.Name,
					Email:     m.User.Email,
					Role:      m.Role,
					CreatedAt: m.CreatedAt,
				})
			}
		}

		// Get connected pages in this tenant
		var pageModels []database.MetaPageTokenModel
		var tenantPages []domain.TenantPage
		if err := r.db.WithContext(ctx).Where("account_id = ?", acc.ID).Order("created_at asc").Find(&pageModels).Error; err == nil {
			for _, p := range pageModels {
				tenantPages = append(tenantPages, domain.TenantPage{
					PageID:      p.PageID,
					Name:        p.Name,
					Category:    p.Category,
					PictureURL:  p.PictureURL,
					IsConnected: p.IsConnected,
					CreatedAt:   p.CreatedAt,
				})
			}
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
			LockedPrice:         lockedPrice,
			CurrentPeriodEnd:    currentPeriodEnd,
			Users:               tenantUsers,
			Pages:               tenantPages,
			CreatedAt:           acc.CreatedAt,
		})
	}

	return result, nil
}

func (r *GormRepository) ListAllPages(ctx context.Context) ([]domain.GlobalPageItem, error) {
	var pages []database.MetaPageTokenModel
	if err := r.db.WithContext(ctx).Preload("Account").Order("created_at desc").Find(&pages).Error; err != nil {
		return nil, fmt.Errorf("list all pages: %w", err)
	}

	result := make([]domain.GlobalPageItem, 0, len(pages))
	for _, p := range pages {
		storeName := p.Account.Name
		if storeName == "" {
			storeName = "Unassigned Store"
		}
		result = append(result, domain.GlobalPageItem{
			PageID:      p.PageID,
			PageName:    p.Name,
			Category:    p.Category,
			PictureURL:  p.PictureURL,
			IsConnected: p.IsConnected,
			AccountID:   p.AccountID,
			StoreName:   storeName,
			CreatedAt:   p.CreatedAt,
		})
	}
	return result, nil
}

func (r *GormRepository) ListAllUsers(ctx context.Context) ([]domain.GlobalUserItem, error) {
	var memberships []database.MembershipModel
	if err := r.db.WithContext(ctx).Preload("User").Preload("Account").Order("created_at desc").Find(&memberships).Error; err != nil {
		return nil, fmt.Errorf("list all users: %w", err)
	}

	result := make([]domain.GlobalUserItem, 0, len(memberships))
	for _, m := range memberships {
		storeName := m.Account.Name
		if storeName == "" {
			storeName = "Unassigned Store"
		}
		result = append(result, domain.GlobalUserItem{
			UserID:       m.UserID,
			UserName:     m.User.Name,
			Email:        m.User.Email,
			IsSuperAdmin: m.User.IsSuperAdmin,
			AccountID:    m.AccountID,
			StoreName:    storeName,
			Role:         m.Role,
			CreatedAt:    m.CreatedAt,
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

func (r *GormRepository) ListAdminPlans(ctx context.Context) ([]domain.AdminPlanItem, error) {
	var models []database.PlanModel
	if err := r.db.WithContext(ctx).Order("price asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list admin plans: %w", err)
	}

	result := make([]domain.AdminPlanItem, 0, len(models))
	for _, m := range models {
		var subCount int64
		_ = r.db.WithContext(ctx).Model(&database.SubscriptionModel{}).
			Where("plan_id = ? AND (status = 'active' OR status = 'trialing')", m.ID).
			Count(&subCount).Error

		result = append(result, domain.AdminPlanItem{
			ID:              m.ID,
			Name:            m.Name,
			Price:           m.Price,
			MaxUsers:        m.MaxUsers,
			MaxPages:        m.MaxPages,
			IsActive:        m.IsActive,
			SubscriberCount: subCount,
		})
	}
	return result, nil
}

func (r *GormRepository) SavePlan(ctx context.Context, plan domain.Plan) error {
	m := database.PlanModel{
		ID:        plan.ID,
		Name:      plan.Name,
		Price:     plan.Price,
		MaxUsers:  plan.MaxUsers,
		MaxPages:  plan.MaxPages,
		IsActive:  plan.IsActive,
		UpdatedAt: time.Now(),
	}

	var existing database.PlanModel
	err := r.db.WithContext(ctx).Where("id = ?", plan.ID).First(&existing).Error
	if err != nil {
		m.CreatedAt = time.Now()
		if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
			return fmt.Errorf("create plan: %w", err)
		}
		return nil
	}

	if err := r.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
		"name":       m.Name,
		"price":      m.Price,
		"max_users":  m.MaxUsers,
		"max_pages":  m.MaxPages,
		"is_active":  m.IsActive,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("update plan: %w", err)
	}
	return nil
}

func (r *GormRepository) DeletePlan(ctx context.Context, planID string) error {
	var subCount int64
	_ = r.db.WithContext(ctx).Model(&database.SubscriptionModel{}).Where("plan_id = ?", planID).Count(&subCount).Error
	if subCount > 0 {
		return r.db.WithContext(ctx).Model(&database.PlanModel{}).Where("id = ?", planID).Update("is_active", false).Error
	}
	return r.db.WithContext(ctx).Where("id = ?", planID).Delete(&database.PlanModel{}).Error
}

func (r *GormRepository) SetTenantPlan(ctx context.Context, accountID, planID string, days, lockedPrice int, status string) error {
	if status == "" {
		status = "active"
	}

	var periodEnd time.Time
	if days > 0 {
		periodEnd = time.Now().AddDate(0, 0, days)
	} else {
		periodEnd = time.Now().AddDate(100, 0, 0)
	}

	var plan database.PlanModel
	_ = r.db.WithContext(ctx).Where("id = ?", planID).First(&plan).Error
	if lockedPrice <= 0 && plan.Price > 0 {
		lockedPrice = plan.Price
	}

	var sub database.SubscriptionModel
	err := r.db.WithContext(ctx).Where("account_id = ?", accountID).First(&sub).Error
	if err != nil {
		newSub := database.SubscriptionModel{
			ID:               uuid.NewString(),
			AccountID:        accountID,
			PlanID:           planID,
			Status:           status,
			LockedPrice:      lockedPrice,
			CurrentPeriodEnd: periodEnd,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		return r.db.WithContext(ctx).Create(&newSub).Error
	}

	return r.db.WithContext(ctx).Model(&sub).Updates(map[string]interface{}{
		"plan_id":            planID,
		"status":             status,
		"locked_price":       lockedPrice,
		"current_period_end": periodEnd,
		"updated_at":         time.Now(),
	}).Error
}
