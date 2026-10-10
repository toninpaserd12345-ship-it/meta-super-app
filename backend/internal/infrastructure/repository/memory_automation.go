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

func (r *MemoryRepository) CreateFlow(ctx context.Context, flow *domain.AutomationFlow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if flow.ID == "" {
		flow.ID = uuid.New().String()
	}
	now := time.Now()
	flow.CreatedAt, flow.UpdatedAt = now, now
	for index := range flow.Targets {
		target := &flow.Targets[index]
		target.ID = uuid.New().String()
		r.automationRules[target.ID] = domain.AutomationRule{ID: target.ID, AccountID: flow.AccountID, PageID: flow.PageID, TriggerType: target.Type, TriggerValue: target.Value, TriggerName: target.Name, ProductID: flow.ProductID, ReplySetID: flow.ReplySetID, FlowID: flow.ID, FlowName: flow.Name, FirstMessageOnly: flow.FirstMessageOnly, CooldownSeconds: flow.CooldownSeconds, IsActive: flow.IsActive, CreatedAt: now, UpdatedAt: now}
	}
	return nil
}

func (r *MemoryRepository) GetFlowsByAccountID(ctx context.Context, accountID string) ([]domain.AutomationFlow, error) {
	rules, err := r.GetRulesByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	groups := map[string]*domain.AutomationFlow{}
	for _, rule := range rules {
		id := rule.FlowID
		if id == "" {
			id = rule.ID
		}
		flow := groups[id]
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
			flow = &domain.AutomationFlow{ID: id, AccountID: rule.AccountID, Name: name, PageID: rule.PageID, ProductID: rule.ProductID, ReplySetID: rule.ReplySetID, FirstMessageOnly: firstMessageOnly, CooldownSeconds: rule.CooldownSeconds, IsActive: rule.IsActive, CreatedAt: rule.CreatedAt, UpdatedAt: rule.UpdatedAt}
			groups[id] = flow
		}
		flow.Targets = append(flow.Targets, domain.AutomationTarget{ID: rule.ID, Type: rule.TriggerType, Value: rule.TriggerValue, Name: rule.TriggerName})
	}
	flows := make([]domain.AutomationFlow, 0, len(groups))
	for _, flow := range groups {
		flows = append(flows, *flow)
	}
	sort.Slice(flows, func(i, j int) bool { return flows[i].UpdatedAt.After(flows[j].UpdatedAt) })
	return flows, nil
}

func (r *MemoryRepository) GetFlowByID(ctx context.Context, id string) (*domain.AutomationFlow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var flow *domain.AutomationFlow
	for _, rule := range r.automationRules {
		flowID := rule.FlowID
		if flowID == "" {
			flowID = rule.ID
		}
		if flowID != id {
			continue
		}
		if flow == nil {
			name := rule.FlowName
			if name == "" {
				name = rule.TriggerName
			}
			firstMessageOnly := rule.FirstMessageOnly
			if rule.FlowID == "" {
				firstMessageOnly = true
			}
			flow = &domain.AutomationFlow{ID: id, AccountID: rule.AccountID, Name: name, PageID: rule.PageID, ProductID: rule.ProductID, ReplySetID: rule.ReplySetID, FirstMessageOnly: firstMessageOnly, CooldownSeconds: rule.CooldownSeconds, IsActive: rule.IsActive, CreatedAt: rule.CreatedAt, UpdatedAt: rule.UpdatedAt}
		}
		flow.Targets = append(flow.Targets, domain.AutomationTarget{ID: rule.ID, Type: rule.TriggerType, Value: rule.TriggerValue, Name: rule.TriggerName})
	}
	if flow == nil {
		return nil, errors.New("automation flow not found")
	}
	return flow, nil
}

func (r *MemoryRepository) UpdateFlow(ctx context.Context, flow *domain.AutomationFlow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	found := false
	for key, rule := range r.automationRules {
		flowID := rule.FlowID
		if flowID == "" {
			flowID = rule.ID
		}
		if flowID == flow.ID {
			delete(r.automationRules, key)
			found = true
		}
	}
	if !found {
		return errors.New("automation flow not found")
	}
	now := time.Now()
	flow.UpdatedAt = now
	for index := range flow.Targets {
		target := &flow.Targets[index]
		target.ID = uuid.New().String()
		r.automationRules[target.ID] = domain.AutomationRule{ID: target.ID, AccountID: flow.AccountID, PageID: flow.PageID, TriggerType: target.Type, TriggerValue: target.Value, TriggerName: target.Name, ProductID: flow.ProductID, ReplySetID: flow.ReplySetID, FlowID: flow.ID, FlowName: flow.Name, FirstMessageOnly: flow.FirstMessageOnly, CooldownSeconds: flow.CooldownSeconds, IsActive: flow.IsActive, CreatedAt: flow.CreatedAt, UpdatedAt: now}
	}
	return nil
}

func (r *MemoryRepository) UpdateFlowStatus(ctx context.Context, id string, active bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	found := false
	for key, rule := range r.automationRules {
		flowID := rule.FlowID
		if flowID == "" {
			flowID = rule.ID
		}
		if flowID == id {
			rule.IsActive = active
			rule.UpdatedAt = time.Now()
			r.automationRules[key] = rule
			found = true
		}
	}
	if !found {
		return errors.New("automation flow not found")
	}
	return nil
}

func (r *MemoryRepository) DeleteFlow(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	found := false
	for key, rule := range r.automationRules {
		flowID := rule.FlowID
		if flowID == "" {
			flowID = rule.ID
		}
		if flowID == id {
			delete(r.automationRules, key)
			found = true
		}
	}
	if !found {
		return errors.New("automation flow not found")
	}
	return nil
}
