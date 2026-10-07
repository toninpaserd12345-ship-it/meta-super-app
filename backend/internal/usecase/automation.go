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
	if triggerType != "post" && triggerType != "ad" && triggerType != "keyword" {
		return nil, fmt.Errorf("%w: triggerType must be post, ad, or keyword", ErrInvalidAutomationRule)
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
	if triggerType != "post" && triggerType != "ad" && triggerType != "keyword" {
		return fmt.Errorf("%w: triggerType must be post, ad, or keyword", ErrInvalidAutomationRule)
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
