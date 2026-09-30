package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
)

// Ensure MemoryRepository implements AutomationRepository
// var _ domain.AutomationRepository = (*MemoryRepository)(nil)

func (r *MemoryRepository) CreateRule(ctx context.Context, rule *domain.AutomationRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := uuid.New().String()
	rule.ID = id
	rule.IsActive = true
	rule.CreatedAt, rule.UpdatedAt = time.Now(), time.Now()
	r.automationRules[id] = *rule
	return nil
}

func (r *MemoryRepository) GetRulesByAccountID(ctx context.Context, accountID string) ([]domain.AutomationRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rules := make([]domain.AutomationRule, 0)
	for _, rule := range r.automationRules {
		if rule.AccountID == accountID {
			rules = append(rules, rule)
		}
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].UpdatedAt.After(rules[j].UpdatedAt) })
	return rules, nil
}

func (r *MemoryRepository) GetRuleByID(ctx context.Context, id string) (*domain.AutomationRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.automationRules[id]
	if !ok {
		return nil, errors.New("automation rule not found")
	}
	return &rule, nil
}

func (r *MemoryRepository) GetRulesByPageAndTrigger(ctx context.Context, pageID, triggerType, triggerValue string) ([]domain.AutomationRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rules := []domain.AutomationRule{}
	for _, rule := range r.automationRules {
		if rule.PageID == pageID && rule.TriggerType == triggerType && rule.TriggerValue == triggerValue && rule.IsActive {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func (r *MemoryRepository) GetRulesByPageAndType(ctx context.Context, pageID, triggerType string) ([]domain.AutomationRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rules := []domain.AutomationRule{}
	for _, rule := range r.automationRules {
		if rule.PageID == pageID && rule.TriggerType == triggerType && rule.IsActive {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func (r *MemoryRepository) UpdateRule(ctx context.Context, rule *domain.AutomationRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.automationRules[rule.ID]; !ok {
		return errors.New("automation rule not found")
	}
	rule.UpdatedAt = time.Now()
	r.automationRules[rule.ID] = *rule
	return nil
}

func (r *MemoryRepository) DeleteRule(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.automationRules[id]; !ok {
		return errors.New("automation rule not found")
	}
	delete(r.automationRules, id)
	return nil
}
