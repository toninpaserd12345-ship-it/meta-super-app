package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
)

// Ensure Gorm implements AutomationRepository
// var _ domain.AutomationRepository = (*GormRepository)(nil)

func (r *GormRepository) CreateRule(ctx context.Context, rule *domain.AutomationRule) error {
	id := uuid.New().String()
	rule.ID = id
	model := database.AutomationRuleModel{
		ID:           id,
		AccountID:    rule.AccountID,
		PageID:       rule.PageID,
		TriggerType:  rule.TriggerType,
		TriggerName:  rule.TriggerName,
		TriggerValue: rule.TriggerValue,
		ProductID:    func() *string {
			if rule.ProductID == "" {
				return nil
			}
			return &rule.ProductID
		}(),
		ReplySetID:   rule.ReplySetID,
		IsActive:     true,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}

	rule.CreatedAt = model.CreatedAt
	rule.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *GormRepository) GetRulesByAccountID(ctx context.Context, accountID string) ([]domain.AutomationRule, error) {
	var models []database.AutomationRuleModel
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Find(&models).Error; err != nil {
		return nil, err
	}

	var rules []domain.AutomationRule
	for _, m := range models {
		rules = append(rules, toDomainAutomationRule(m))
	}
	return rules, nil
}

func (r *GormRepository) GetRuleByID(ctx context.Context, id string) (*domain.AutomationRule, error) {
	var model database.AutomationRuleModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, err
	}
	rule := toDomainAutomationRule(model)
	return &rule, nil
}

func (r *GormRepository) GetRulesByPageAndTrigger(ctx context.Context, pageID, triggerType, triggerValue string) ([]domain.AutomationRule, error) {
	var models []database.AutomationRuleModel
	if err := r.db.WithContext(ctx).Where("page_id = ? AND trigger_type = ? AND trigger_value = ? AND is_active = ?", pageID, triggerType, triggerValue, true).Find(&models).Error; err != nil {
		return nil, err
	}

	var rules []domain.AutomationRule
	for _, m := range models {
		rules = append(rules, toDomainAutomationRule(m))
	}
	return rules, nil
}

func (r *GormRepository) GetRulesByPageAndType(ctx context.Context, pageID, triggerType string) ([]domain.AutomationRule, error) {
	var models []database.AutomationRuleModel
	if err := r.db.WithContext(ctx).Where("page_id = ? AND trigger_type = ? AND is_active = ?", pageID, triggerType, true).Find(&models).Error; err != nil {
		return nil, err
	}

	var rules []domain.AutomationRule
	for _, m := range models {
		rules = append(rules, toDomainAutomationRule(m))
	}
	return rules, nil
}

func (r *GormRepository) UpdateRule(ctx context.Context, rule *domain.AutomationRule) error {
	return r.db.WithContext(ctx).Model(&database.AutomationRuleModel{}).Where("id = ?", rule.ID).Updates(map[string]interface{}{
		"trigger_type":  rule.TriggerType,
		"trigger_name":  rule.TriggerName,
		"trigger_value": rule.TriggerValue,
		"product_id":    func() *string { if rule.ProductID == "" { return nil } else { return &rule.ProductID } }(),
		"reply_set_id":  rule.ReplySetID,
		"is_active":     rule.IsActive,
	}).Error
}

func (r *GormRepository) DeleteRule(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&database.AutomationRuleModel{}).Error
}

func toDomainAutomationRule(m database.AutomationRuleModel) domain.AutomationRule {
	return domain.AutomationRule{
		ID:           m.ID,
		AccountID:    m.AccountID,
		PageID:       m.PageID,
		TriggerType:  m.TriggerType,
		TriggerName:  m.TriggerName,
		TriggerValue: m.TriggerValue,
		ProductID:    func() string {
			if m.ProductID == nil {
				return ""
			}
			return *m.ProductID
		}(),
		ReplySetID:   m.ReplySetID,
		IsActive:     m.IsActive,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
