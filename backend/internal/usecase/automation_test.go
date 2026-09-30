package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/meta-super-app/backend/internal/infrastructure/repository"
	"github.com/meta-super-app/backend/internal/usecase"
)

func TestAutomationCreateRuleValidation(t *testing.T) {
	tests := []struct {
		name         string
		accountID    string
		pageID       string
		triggerType  string
		triggerValue string
		productID    string
		replySetID   string
	}{
		{name: "missing page", accountID: "account-1", triggerType: "post", triggerValue: "post-1", productID: "product-1", replySetID: "reply-1"},
		{name: "missing trigger type", accountID: "account-1", pageID: "page-1", triggerValue: "post-1", productID: "product-1", replySetID: "reply-1"},
		{name: "missing trigger value", accountID: "account-1", pageID: "page-1", triggerType: "post", productID: "product-1", replySetID: "reply-1"},
		{name: "missing product", accountID: "account-1", pageID: "page-1", triggerType: "post", triggerValue: "post-1", replySetID: "reply-1"},
		{name: "missing reply set", accountID: "account-1", pageID: "page-1", triggerType: "post", triggerValue: "post-1", productID: "product-1"},
		{name: "unsupported trigger", accountID: "account-1", pageID: "page-1", triggerType: "campaign", triggerValue: "campaign-1", productID: "product-1", replySetID: "reply-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := usecase.NewAutomation(repository.NewMemory(""))
			if _, err := service.CreateRule(context.Background(), tt.accountID, tt.pageID, tt.triggerType, tt.triggerValue, tt.productID, tt.replySetID); err == nil {
				t.Fatal("CreateRule() returned no validation error")
			}
		})
	}
}

func TestAutomationCreateAndUpdatePropagateProductID(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory("")
	service := usecase.NewAutomation(repo)

	for _, triggerType := range []string{"post", "ad", "keyword"} {
		rule, err := service.CreateRule(ctx, "account-1", "page-1", triggerType, triggerType+"-value", "product-1", "reply-1")
		if err != nil {
			t.Fatalf("CreateRule(%s) error = %v", triggerType, err)
		}
		if rule.ProductID != "product-1" || rule.ReplySetID != "reply-1" || !rule.IsActive {
			t.Fatalf("CreateRule(%s) = %#v", triggerType, rule)
		}
	}

	rules, err := service.GetRules(ctx, "account-1")
	if err != nil {
		t.Fatalf("GetRules() error = %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("GetRules() count = %d, want 3", len(rules))
	}
	rule := rules[0]
	if err = service.UpdateRule(ctx, rule.ID, "account-1", rule.TriggerType, rule.TriggerValue, "product-2", "reply-2", false); err != nil {
		t.Fatalf("UpdateRule() error = %v", err)
	}
	updated, err := service.GetRule(ctx, rule.ID, "account-1")
	if err != nil {
		t.Fatalf("GetRule() error = %v", err)
	}
	if updated.ProductID != "product-2" || updated.ReplySetID != "reply-2" || updated.IsActive {
		t.Fatalf("UpdateRule() did not propagate fields: %#v", updated)
	}
}

func TestAutomationRejectsCrossAccountAccess(t *testing.T) {
	ctx := context.Background()
	service := usecase.NewAutomation(repository.NewMemory(""))
	rule, err := service.CreateRule(ctx, "account-1", "page-1", "post", "post-1", "product-1", "reply-1")
	if err != nil {
		t.Fatalf("CreateRule() error = %v", err)
	}

	if _, err = service.GetRule(ctx, rule.ID, "account-2"); err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("GetRule(cross-account) error = %v", err)
	}
	if err = service.DeleteRule(ctx, rule.ID, "account-2"); err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("DeleteRule(cross-account) error = %v", err)
	}
	if _, err = service.GetRule(ctx, rule.ID, "account-1"); err != nil {
		t.Fatalf("cross-account delete changed rule: %v", err)
	}
}
