package domain

import (
	"context"
	"time"
)

type AutomationRule struct {
	ID               string    `json:"id"`
	AccountID        string    `json:"accountId"`
	PageID           string    `json:"pageId"`
	TriggerType      string    `json:"triggerType"`
	TriggerName      string    `json:"triggerName"`  // post, ad, keyword
	TriggerValue     string    `json:"triggerValue"` // PostID, AdID, Keyword
	ProductID        string    `json:"productId"`
	ReplySetID       string    `json:"replySetId"`
	FlowID           string    `json:"flowId"`
	FlowName         string    `json:"flowName"`
	FirstMessageOnly bool      `json:"firstMessageOnly"`
	CooldownSeconds  int       `json:"cooldownSeconds"`
	IsActive         bool      `json:"isActive"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type AutomationTarget struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
	Name  string `json:"name"`
}

// AutomationFlow is the user-facing automation aggregate. A flow owns the
// product and reply sequence once, and can have many Facebook targets.
type AutomationFlow struct {
	ID               string             `json:"id"`
	AccountID        string             `json:"accountId"`
	Name             string             `json:"name"`
	PageID           string             `json:"pageId"`
	ProductID        string             `json:"productId"`
	ReplySetID       string             `json:"replySetId"`
	FirstMessageOnly bool               `json:"firstMessageOnly"`
	CooldownSeconds  int                `json:"cooldownSeconds"`
	IsActive         bool               `json:"isActive"`
	Targets          []AutomationTarget `json:"targets"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
}

type AutomationRepository interface {
	CreateRule(ctx context.Context, rule *AutomationRule) error
	GetRulesByAccountID(ctx context.Context, accountID string) ([]AutomationRule, error)
	GetRuleByID(ctx context.Context, id string) (*AutomationRule, error)
	GetRulesByPageAndTrigger(ctx context.Context, pageID, triggerType, triggerValue string) ([]AutomationRule, error)
	GetRulesByPageAndType(ctx context.Context, pageID, triggerType string) ([]AutomationRule, error)
	UpdateRule(ctx context.Context, rule *AutomationRule) error
	DeleteRule(ctx context.Context, id string) error
	CreateFlow(ctx context.Context, flow *AutomationFlow) error
	GetFlowsByAccountID(ctx context.Context, accountID string) ([]AutomationFlow, error)
	GetFlowByID(ctx context.Context, id string) (*AutomationFlow, error)
	UpdateFlow(ctx context.Context, flow *AutomationFlow) error
	UpdateFlowStatus(ctx context.Context, id string, isActive bool) error
	DeleteFlow(ctx context.Context, id string) error
}
