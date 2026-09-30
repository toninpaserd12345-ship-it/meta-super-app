package repository

import (
	"context"
	"testing"

	"github.com/meta-super-app/backend/internal/domain"
)

func TestMemoryAutomationRuleCRUDAndFind(t *testing.T) {
	ctx := context.Background()
	repo := NewMemory("")
	rule := &domain.AutomationRule{
		AccountID:    "account-1",
		PageID:       "page-1",
		TriggerType:  "post",
		TriggerValue: "post-1",
		ProductID:    "product-1",
		ReplySetID:   "reply-1",
	}

	if err := repo.CreateRule(ctx, rule); err != nil {
		t.Fatalf("CreateRule() error = %v", err)
	}
	if rule.ID == "" || !rule.IsActive || rule.CreatedAt.IsZero() || rule.UpdatedAt.IsZero() {
		t.Fatalf("CreateRule() did not populate defaults: %#v", rule)
	}
	if rule.ProductID != "product-1" {
		t.Fatalf("CreateRule() ProductID = %q, want product-1", rule.ProductID)
	}

	other := &domain.AutomationRule{AccountID: "account-2", PageID: "page-1", TriggerType: "post", TriggerValue: "post-2", ProductID: "product-2", ReplySetID: "reply-2"}
	if err := repo.CreateRule(ctx, other); err != nil {
		t.Fatalf("CreateRule(other) error = %v", err)
	}
	rules, err := repo.GetRulesByAccountID(ctx, "account-1")
	if err != nil {
		t.Fatalf("GetRulesByAccountID() error = %v", err)
	}
	if len(rules) != 1 || rules[0].ID != rule.ID || rules[0].ProductID != "product-1" {
		t.Fatalf("GetRulesByAccountID() = %#v", rules)
	}

	found, err := repo.GetRulesByPageAndTrigger(ctx, "page-1", "post", "post-1")
	if err != nil {
		t.Fatalf("GetRulesByPageAndTrigger() error = %v", err)
	}
	if len(found) != 1 || found[0].ID != rule.ID {
		t.Fatalf("GetRulesByPageAndTrigger() = %#v", found)
	}
	found, err = repo.GetRulesByPageAndType(ctx, "page-1", "post")
	if err != nil {
		t.Fatalf("GetRulesByPageAndType() error = %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("GetRulesByPageAndType() count = %d, want 2 active rules", len(found))
	}

	rule.ProductID = "product-updated"
	rule.ReplySetID = "reply-updated"
	rule.IsActive = false
	if err = repo.UpdateRule(ctx, rule); err != nil {
		t.Fatalf("UpdateRule() error = %v", err)
	}
	updated, err := repo.GetRuleByID(ctx, rule.ID)
	if err != nil {
		t.Fatalf("GetRuleByID() error = %v", err)
	}
	if updated.ProductID != "product-updated" || updated.ReplySetID != "reply-updated" || updated.IsActive {
		t.Fatalf("UpdateRule() did not persist fields: %#v", updated)
	}
	found, err = repo.GetRulesByPageAndTrigger(ctx, "page-1", "post", "post-1")
	if err != nil {
		t.Fatalf("GetRulesByPageAndTrigger(inactive) error = %v", err)
	}
	if len(found) != 0 {
		t.Fatalf("inactive rule was returned: %#v", found)
	}

	if err = repo.DeleteRule(ctx, rule.ID); err != nil {
		t.Fatalf("DeleteRule() error = %v", err)
	}
	if _, err = repo.GetRuleByID(ctx, rule.ID); err == nil {
		t.Fatal("GetRuleByID() after delete returned no error")
	}
}

func TestMemoryAutomationRuleMissingMutationsReturnError(t *testing.T) {
	ctx := context.Background()
	repo := NewMemory("")

	if err := repo.UpdateRule(ctx, &domain.AutomationRule{ID: "missing"}); err == nil {
		t.Fatal("UpdateRule(missing) returned no error")
	}
	if err := repo.DeleteRule(ctx, "missing"); err == nil {
		t.Fatal("DeleteRule(missing) returned no error")
	}
}
