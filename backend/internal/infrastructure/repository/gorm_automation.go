package repository

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
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
		ProductID: func() *string {
			if rule.ProductID == "" {
				return nil
			}
			return &rule.ProductID
		}(),
		ReplySetID:       rule.ReplySetID,
		FlowID:           rule.FlowID,
		FlowName:         rule.FlowName,
		FirstMessageOnly: rule.FirstMessageOnly,
		CooldownSeconds:  rule.CooldownSeconds,
		IsActive:         rule.IsActive,
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
		"product_id": func() *string {
			if rule.ProductID == "" {
				return nil
			} else {
				return &rule.ProductID
			}
		}(),
		"reply_set_id":       rule.ReplySetID,
		"flow_name":          rule.FlowName,
		"first_message_only": rule.FirstMessageOnly,
		"cooldown_seconds":   rule.CooldownSeconds,
		"is_active":          rule.IsActive,
	}).Error
}

func (r *GormRepository) CreateFlow(ctx context.Context, flow *domain.AutomationFlow) error {
	if flow.ID == "" {
		flow.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index := range flow.Targets {
			target := &flow.Targets[index]
			target.ID = uuid.New().String()
			productID := (*string)(nil)
			if flow.ProductID != "" {
				value := flow.ProductID
				productID = &value
			}
			model := database.AutomationRuleModel{
				ID: target.ID, AccountID: flow.AccountID, PageID: flow.PageID,
				TriggerType: target.Type, TriggerName: target.Name, TriggerValue: target.Value,
				ProductID: productID, ReplySetID: flow.ReplySetID, FlowID: flow.ID,
				FlowName: flow.Name, FirstMessageOnly: flow.FirstMessageOnly,
				CooldownSeconds: flow.CooldownSeconds, IsActive: flow.IsActive,
			}
			if err := tx.Create(&model).Error; err != nil {
				return err
			}
			if index == 0 {
				flow.CreatedAt, flow.UpdatedAt = model.CreatedAt, model.UpdatedAt
			}
		}
		return nil
	})
}

func (r *GormRepository) GetFlowsByAccountID(ctx context.Context, accountID string) ([]domain.AutomationFlow, error) {
	rules, err := r.GetRulesByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	groups := make(map[string]*domain.AutomationFlow)
	order := make([]string, 0)
	for _, rule := range rules {
		flowID := rule.FlowID
		if flowID == "" {
			flowID = rule.ID
		}
		flow := groups[flowID]
		if flow == nil {
			name := rule.FlowName
			if name == "" {
				name = rule.TriggerName
				if name == "" {
					name = "Imported automation"
				}
			}
			firstMessageOnly := rule.FirstMessageOnly
			if rule.FlowID == "" {
				firstMessageOnly = true
			}
			flow = &domain.AutomationFlow{ID: flowID, AccountID: rule.AccountID, Name: name, PageID: rule.PageID, ProductID: rule.ProductID, ReplySetID: rule.ReplySetID, FirstMessageOnly: firstMessageOnly, CooldownSeconds: rule.CooldownSeconds, IsActive: rule.IsActive, CreatedAt: rule.CreatedAt, UpdatedAt: rule.UpdatedAt}
			groups[flowID] = flow
			order = append(order, flowID)
		}
		flow.Targets = append(flow.Targets, domain.AutomationTarget{ID: rule.ID, Type: rule.TriggerType, Value: rule.TriggerValue, Name: rule.TriggerName})
		if rule.UpdatedAt.After(flow.UpdatedAt) {
			flow.UpdatedAt = rule.UpdatedAt
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return groups[order[i]].CreatedAt.After(groups[order[j]].CreatedAt) })
	flows := make([]domain.AutomationFlow, 0, len(order))
	for _, id := range order {
		flows = append(flows, *groups[id])
	}
	return flows, nil
}

func (r *GormRepository) GetFlowByID(ctx context.Context, id string) (*domain.AutomationFlow, error) {
	var model database.AutomationRuleModel
	query := r.db.WithContext(ctx).Where("flow_id = ?", id).First(&model)
	if query.Error != nil {
		query = r.db.WithContext(ctx).Where("id = ? AND (flow_id = '' OR flow_id IS NULL)", id).First(&model)
	}
	if query.Error != nil {
		return nil, query.Error
	}
	flows, err := r.GetFlowsByAccountID(ctx, model.AccountID)
	if err != nil {
		return nil, err
	}
	for index := range flows {
		if flows[index].ID == id {
			return &flows[index], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *GormRepository) UpdateFlowStatus(ctx context.Context, id string, isActive bool) error {
	result := r.db.WithContext(ctx).Model(&database.AutomationRuleModel{}).Where("flow_id = ?", id).Update("is_active", isActive)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return r.db.WithContext(ctx).Model(&database.AutomationRuleModel{}).Where("id = ? AND (flow_id = '' OR flow_id IS NULL)", id).Update("is_active", isActive).Error
	}
	return nil
}

func (r *GormRepository) DeleteFlow(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("flow_id = ?", id).Delete(&database.AutomationRuleModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return r.db.WithContext(ctx).Where("id = ? AND (flow_id = '' OR flow_id IS NULL)", id).Delete(&database.AutomationRuleModel{}).Error
	}
	return nil
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
		ProductID: func() string {
			if m.ProductID == nil {
				return ""
			}
			return *m.ProductID
		}(),
		ReplySetID:       m.ReplySetID,
		FlowID:           m.FlowID,
		FlowName:         m.FlowName,
		FirstMessageOnly: m.FirstMessageOnly,
		CooldownSeconds:  m.CooldownSeconds,
		IsActive:         m.IsActive,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
