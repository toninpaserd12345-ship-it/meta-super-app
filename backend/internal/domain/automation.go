package domain

import (
	"context"
	"time"
)

type AutomationRule struct {
	ID           string    `json:"id"`
	AccountID    string    `json:"accountId"`
	PageID       string    `json:"pageId"`
	TriggerType  string    `json:"triggerType"`
	TriggerName  string    `json:"triggerName"`  // post, ad, keyword
	TriggerValue string    `json:"triggerValue"` // PostID, AdID, Keyword
	ProductID    string    `json:"productId"`
	ReplySetID   string    `json:"replySetId"`
	IsActive     bool      `json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type AutomationRepository interface {
	CreateRule(ctx context.Context, rule *AutomationRule) error
	GetRulesByAccountID(ctx context.Context, accountID string) ([]AutomationRule, error)
	GetRuleByID(ctx context.Context, id string) (*AutomationRule, error)
	GetRulesByPageAndTrigger(ctx context.Context, pageID, triggerType, triggerValue string) ([]AutomationRule, error)
	GetRulesByPageAndType(ctx context.Context, pageID, triggerType string) ([]AutomationRule, error)
	UpdateRule(ctx context.Context, rule *AutomationRule) error
	DeleteRule(ctx context.Context, id string) error
}
