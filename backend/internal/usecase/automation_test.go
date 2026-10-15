package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/repository"
	"github.com/meta-super-app/backend/internal/usecase"
)

func TestAutomationFlowGroupsTargetsAndUpdatesTogether(t *testing.T) {
	ctx := context.Background()
	service := usecase.NewAutomation(repository.NewMemory(""))
	flow := &domain.AutomationFlow{AccountID: "account-1", Name: "Campaign replies", PageID: "page-1", ProductID: "product-1", ReplySetID: "reply-1", FirstMessageOnly: true, Targets: []domain.AutomationTarget{{Type: "campaign", Value: "campaign-1", Name: "One"}, {Type: "campaign", Value: "campaign-2", Name: "Two"}}}
	if err := service.CreateFlow(ctx, flow); err != nil {
		t.Fatalf("CreateFlow() error = %v", err)
	}
	flows, err := service.GetFlows(ctx, "account-1")
	if err != nil || len(flows) != 1 || len(flows[0].Targets) != 2 {
		t.Fatalf("GetFlows() = %#v, %v", flows, err)
	}
	if err = service.SetFlowStatus(ctx, flow.ID, "account-1", false); err != nil {
		t.Fatalf("SetFlowStatus() error = %v", err)
	}
	rules, _ := service.GetRules(ctx, "account-1")
	for _, rule := range rules {
		if rule.IsActive {
			t.Fatalf("rule remained active: %#v", rule)
		}
	}
	flow.Name = "Updated replies"
	flow.ProductID = "product-2"
	flow.ReplySetID = "reply-2"
	flow.CooldownSeconds = 90
	flow.Targets = []domain.AutomationTarget{{Type: "post", Value: "post-1", Name: "Updated target"}}
	if err = service.UpdateFlow(ctx, flow); err != nil {
		t.Fatalf("UpdateFlow() error = %v", err)
	}
	updatedFlows, _ := service.GetFlows(ctx, "account-1")
	if len(updatedFlows) != 1 || updatedFlows[0].Name != "Updated replies" || updatedFlows[0].ProductID != "product-2" || updatedFlows[0].CooldownSeconds != 90 || len(updatedFlows[0].Targets) != 1 || updatedFlows[0].Targets[0].Value != "post-1" || updatedFlows[0].IsActive {
		t.Fatalf("UpdateFlow() did not persist details: %#v", updatedFlows)
	}
	if err = service.DeleteFlow(ctx, flow.ID, "account-1"); err != nil {
		t.Fatalf("DeleteFlow() error = %v", err)
	}
	flows, _ = service.GetFlows(ctx, "account-1")
	if len(flows) != 0 {
		t.Fatalf("flow was not deleted: %#v", flows)
	}
}

func TestAutomationFlowMixedTargetsLongCaptionAndCustomCommentReply(t *testing.T) {
	ctx := context.Background()
	service := usecase.NewAutomation(repository.NewMemory(""))
	longCaption := strings.TrimSpace(strings.Repeat("🔥 ໂປຣໂມຊັ່ນພິເສດ! ເກີບ ກິໂຕ ແທ້ 100% ", 20)) // > 600 chars
	customComment := "ຂອບໃຈທີ່ສົນໃຈ! ພວກເຮົາໄດ້ສົ່ງຂໍ້ມູນໃຫ້ທາງ Inbox ແລ້ວເດີ້"

	flow := &domain.AutomationFlow{
		AccountID:        "account-1",
		Name:             "Kito Shoes Promotion",
		PageID:           "page-107",
		ProductID:        "prod-1",
		ReplySetID:       "reply-1",
		CommentReplyText: customComment,
		FirstMessageOnly: true,
		Targets: []domain.AutomationTarget{
			{Type: "post", Value: "post_107_999", Name: longCaption},
			{Type: "campaign", Value: "camp_888", Name: "Kito Brand Campaign"},
		},
	}

	if err := service.CreateFlow(ctx, flow); err != nil {
		t.Fatalf("CreateFlow() error = %v", err)
	}

	fetched, err := service.GetFlow(ctx, flow.ID, "account-1")
	if err != nil {
		t.Fatalf("GetFlow() error = %v", err)
	}
	if fetched.CommentReplyText != customComment {
		t.Fatalf("CommentReplyText = %q, want %q", fetched.CommentReplyText, customComment)
	}
	if len(fetched.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(fetched.Targets))
	}

	// Verify both targets exist with correct types
	hasPost := false
	hasCampaign := false
	for _, target := range fetched.Targets {
		if target.Type == "post" && target.Name == longCaption {
			hasPost = true
		}
		if target.Type == "campaign" && target.Value == "camp_888" {
			hasCampaign = true
		}
	}
	if !hasPost || !hasCampaign {
		t.Fatalf("expected both post and campaign targets, got: %#v", fetched.Targets)
	}

	// Test updating flow with new custom comment and targets
	fetched.CommentReplyText = "ຂໍ້ຄວາມໃໝ່"
	fetched.Targets = append(fetched.Targets, domain.AutomationTarget{Type: "ad", Value: "ad_777", Name: "Ad 777"})
	if err := service.UpdateFlow(ctx, fetched); err != nil {
		t.Fatalf("UpdateFlow() error = %v", err)
	}

	updated, err := service.GetFlow(ctx, flow.ID, "account-1")
	if err != nil {
		t.Fatalf("GetFlow() after update error = %v", err)
	}
	if updated.CommentReplyText != "ຂໍ້ຄວາມໃໝ່" {
		t.Fatalf("CommentReplyText after update = %q, want %q", updated.CommentReplyText, "ຂໍ້ຄວາມໃໝ່")
	}
	if len(updated.Targets) != 3 {
		t.Fatalf("expected 3 targets after update, got %d", len(updated.Targets))
	}
}

func TestAutomationCreateRuleValidation(t *testing.T) {
	tests := []struct {
		name         string
		accountID    string
		pageID       string
		triggerType  string
		triggerValue string
		triggerName  string
		productID    string
		replySetID   string
	}{
		{name: "missing page", accountID: "account-1", triggerType: "post", triggerValue: "post-1", productID: "product-1", replySetID: "reply-1"},
		{name: "missing trigger type", accountID: "account-1", pageID: "page-1", triggerValue: "post-1", productID: "product-1", replySetID: "reply-1"},
		{name: "missing trigger value", accountID: "account-1", pageID: "page-1", triggerType: "post", productID: "product-1", replySetID: "reply-1"},
		{name: "missing reply set", accountID: "account-1", pageID: "page-1", triggerType: "post", triggerValue: "post-1", productID: "product-1"},
		{name: "unsupported trigger", accountID: "account-1", pageID: "page-1", triggerType: "campaign", triggerValue: "campaign-1", productID: "product-1", replySetID: "reply-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := usecase.NewAutomation(repository.NewMemory(""))
			if _, err := service.CreateRule(context.Background(), tt.accountID, tt.pageID, tt.triggerType, tt.triggerValue, tt.triggerName, tt.productID, tt.replySetID); err == nil {
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
		rule, err := service.CreateRule(ctx, "account-1", "page-1", triggerType, triggerType+"-value", "", "product-1", "reply-1")
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
	if err = service.UpdateRule(ctx, rule.ID, "account-1", rule.TriggerType, rule.TriggerValue, "", "product-2", "reply-2", false); err != nil {
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
	rule, err := service.CreateRule(ctx, "account-1", "page-1", "post", "post-1", "", "product-1", "reply-1")
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
