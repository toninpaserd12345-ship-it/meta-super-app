package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/meta-super-app/backend/internal/domain"
)

var (
	ErrInvalidAutomationRule  = errors.New("invalid automation rule")
	ErrAutomationRuleConflict = errors.New("an active automation already uses this trigger")
)

type Automation struct {
	repo domain.AutomationRepository
}

func NewAutomation(repo domain.AutomationRepository) *Automation {
	return &Automation{repo: repo}
}

func (u *Automation) CreateRule(ctx context.Context, accountID, pageID, triggerType, triggerValue, triggerName, productID, replySetID string) (*domain.AutomationRule, error) {
	accountID = strings.TrimSpace(accountID)
	pageID = strings.TrimSpace(pageID)
	triggerType = strings.ToLower(strings.TrimSpace(triggerType))
	triggerValue = strings.TrimSpace(triggerValue)
	triggerName = strings.TrimSpace(triggerName)
	productID = strings.TrimSpace(productID)
	replySetID = strings.TrimSpace(replySetID)
	if pageID == "" || triggerType == "" || triggerValue == "" || replySetID == "" {
		return nil, fmt.Errorf("%w: pageId, triggerType, triggerValue, and replySetId are required", ErrInvalidAutomationRule)
	}
	if !validLegacyTriggerType(triggerType) {
		return nil, fmt.Errorf("%w: unsupported triggerType", ErrInvalidAutomationRule)
	}
	existing, err := u.repo.GetRulesByPageAndTrigger(ctx, pageID, triggerType, triggerValue)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, ErrAutomationRuleConflict
	}

	rule := &domain.AutomationRule{
		AccountID:    accountID,
		PageID:       pageID,
		TriggerType:  triggerType,
		TriggerName:  triggerName,
		TriggerValue: triggerValue,
		ProductID:    productID,
		ReplySetID:   replySetID,
		IsActive:     true,
	}

	if err := u.repo.CreateRule(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

func (u *Automation) GetRules(ctx context.Context, accountID string) ([]domain.AutomationRule, error) {
	return u.repo.GetRulesByAccountID(ctx, accountID)
}

func (u *Automation) GetRule(ctx context.Context, id string, accountID string) (*domain.AutomationRule, error) {
	rule, err := u.repo.GetRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if rule.AccountID != accountID {
		return nil, fmt.Errorf("unauthorized to access this automation rule")
	}

	return rule, nil
}

func (u *Automation) UpdateRule(ctx context.Context, id, accountID, triggerType, triggerValue, triggerName, productID, replySetID string, isActive bool) error {
	rule, err := u.GetRule(ctx, id, accountID)
	if err != nil {
		return err
	}
	triggerType = strings.ToLower(strings.TrimSpace(triggerType))
	triggerValue = strings.TrimSpace(triggerValue)
	triggerName = strings.TrimSpace(triggerName)
	productID = strings.TrimSpace(productID)
	replySetID = strings.TrimSpace(replySetID)
	if triggerValue == "" || replySetID == "" {
		return fmt.Errorf("%w: triggerValue and replySetId are required", ErrInvalidAutomationRule)
	}
	if !validLegacyTriggerType(triggerType) {
		return fmt.Errorf("%w: unsupported triggerType", ErrInvalidAutomationRule)
	}
	if isActive {
		existing, findErr := u.repo.GetRulesByPageAndTrigger(ctx, rule.PageID, triggerType, triggerValue)
		if findErr != nil {
			return findErr
		}
		for _, candidate := range existing {
			if candidate.ID != rule.ID {
				return ErrAutomationRuleConflict
			}
		}
	}

	rule.TriggerType = triggerType
	rule.TriggerValue = triggerValue
	rule.ProductID = productID
	rule.ReplySetID = replySetID
	rule.IsActive = isActive

	return u.repo.UpdateRule(ctx, rule)
}

func (u *Automation) DeleteRule(ctx context.Context, id string, accountID string) error {
	_, err := u.GetRule(ctx, id, accountID)
	if err != nil {
		return err
	}

	return u.repo.DeleteRule(ctx, id)
}

func (u *Automation) FindActiveRuleForTrigger(ctx context.Context, pageID, triggerType, triggerValue string) (*domain.AutomationRule, error) {
	rules, err := u.repo.GetRulesByPageAndTrigger(ctx, pageID, triggerType, triggerValue)
	if err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, nil // No rule found
	}

	return &rules[0], nil
}

func (u *Automation) GetKeywordRules(ctx context.Context, pageID string) ([]domain.AutomationRule, error) {
	return u.repo.GetRulesByPageAndType(ctx, pageID, "keyword")
}

func validTriggerType(value string) bool {
	switch value {
	case "post", "ad", "campaign", "adset", "keyword":
		return true
	}
	return false
}

func validLegacyTriggerType(value string) bool {
	switch value {
	case "post", "ad", "keyword":
		return true
	}
	return false
}

func (u *Automation) CreateFlow(ctx context.Context, flow *domain.AutomationFlow) error {
	if err := u.prepareFlow(ctx, flow, ""); err != nil {
		return err
	}
	flow.IsActive = true
	return u.repo.CreateFlow(ctx, flow)
}

func (u *Automation) prepareFlow(ctx context.Context, flow *domain.AutomationFlow, ignoredFlowID string) error {
	flow.AccountID = strings.TrimSpace(flow.AccountID)
	flow.Name = strings.TrimSpace(flow.Name)
	flow.PageID = strings.TrimSpace(flow.PageID)
	flow.ProductID = strings.TrimSpace(flow.ProductID)
	flow.ReplySetID = strings.TrimSpace(flow.ReplySetID)
	if flow.Name == "" || flow.PageID == "" || flow.ProductID == "" || flow.ReplySetID == "" || len(flow.Targets) == 0 {
		return fmt.Errorf("%w: name, pageId, productId, replySetId, and at least one target are required", ErrInvalidAutomationRule)
	}
	if flow.CooldownSeconds < 0 || flow.CooldownSeconds > 86400 {
		return fmt.Errorf("%w: cooldownSeconds must be between 0 and 86400", ErrInvalidAutomationRule)
	}
	seen := map[string]bool{}
	clean := make([]domain.AutomationTarget, 0, len(flow.Targets))
	for _, target := range flow.Targets {
		target.Type = strings.ToLower(strings.TrimSpace(target.Type))
		target.Value = strings.TrimSpace(target.Value)
		target.Name = strings.TrimSpace(target.Name)
		if !validTriggerType(target.Type) || target.Value == "" {
			return fmt.Errorf("%w: every target needs a valid type and value", ErrInvalidAutomationRule)
		}
		key := target.Type + ":" + target.Value
		if seen[key] {
			continue
		}
		seen[key] = true
		existing, err := u.repo.GetRulesByPageAndTrigger(ctx, flow.PageID, target.Type, target.Value)
		if err != nil {
			return err
		}
		for _, rule := range existing {
			ruleFlowID := rule.FlowID
			if ruleFlowID == "" {
				ruleFlowID = rule.ID
			}
			if ignoredFlowID == "" || ruleFlowID != ignoredFlowID {
				return ErrAutomationRuleConflict
			}
		}
		clean = append(clean, target)
	}
	flow.Targets = clean
	return nil
}

func (u *Automation) GetFlows(ctx context.Context, accountID string) ([]domain.AutomationFlow, error) {
	return u.repo.GetFlowsByAccountID(ctx, accountID)
}

func (u *Automation) UpdateFlow(ctx context.Context, flow *domain.AutomationFlow) error {
	existing, err := u.repo.GetFlowByID(ctx, flow.ID)
	if err != nil {
		return err
	}
	if existing.AccountID != flow.AccountID {
		return fmt.Errorf("unauthorized to access this automation")
	}
	if err := u.prepareFlow(ctx, flow, existing.ID); err != nil {
		return err
	}
	flow.ID = existing.ID
	flow.IsActive = existing.IsActive
	flow.CreatedAt = existing.CreatedAt
	return u.repo.UpdateFlow(ctx, flow)
}

func (u *Automation) SetFlowStatus(ctx context.Context, id, accountID string, active bool) error {
	flow, err := u.repo.GetFlowByID(ctx, id)
	if err != nil {
		return err
	}
	if flow.AccountID != accountID {
		return fmt.Errorf("unauthorized to access this automation")
	}
	return u.repo.UpdateFlowStatus(ctx, id, active)
}

func (u *Automation) DeleteFlow(ctx context.Context, id, accountID string) error {
	flow, err := u.repo.GetFlowByID(ctx, id)
	if err != nil {
		return err
	}
	if flow.AccountID != accountID {
		return fmt.Errorf("unauthorized to access this automation")
	}
	return u.repo.DeleteFlow(ctx, id)
}
