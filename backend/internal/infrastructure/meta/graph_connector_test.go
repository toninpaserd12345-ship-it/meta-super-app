package meta

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/meta-super-app/backend/internal/domain"
)

func TestGraphStateSurvivesRestart(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "meta-state.json")
	first := NewGraphConnector(GraphConfig{StateFile: stateFile}, nil)
	first.sessions["account-1"] = &graphSession{UserToken: "user-token", Pages: map[string]graphPage{"page-1": {MetaPage: domain.MetaPage{ID: "page-1", Connected: true}, AccessToken: "page-token"}}}
	first.products["account-1"] = map[string]domain.Product{"product-1": {ID: "product-1", Name: "Durian"}}
	first.replyFlows["account-1"] = map[string]domain.ReplyFlow{"flow-1": {ID: "flow-1", Name: "Price reply"}}
	first.bindings["page-1:post:post-1"] = domain.ProductBinding{PageID: "page-1", SourceType: "post", SourceID: "post-1", ProductID: "product-1", FlowID: "flow-1"}
	if err := first.persistStateLocked(); err != nil {
		t.Fatal(err)
	}

	restarted := NewGraphConnector(GraphConfig{StateFile: stateFile}, nil)
	if restarted.sessions["account-1"] == nil || restarted.sessions["account-1"].Pages["page-1"].AccessToken != "page-token" {
		t.Fatal("OAuth session was not restored")
	}
	if restarted.products["account-1"]["product-1"].Name != "Durian" {
		t.Fatal("product was not restored")
	}
	if restarted.replyFlows["account-1"]["flow-1"].Name != "Price reply" {
		t.Fatal("reply flow was not restored")
	}
	if restarted.bindings["page-1:post:post-1"].ProductID != "product-1" {
		t.Fatal("binding was not restored")
	}
	info, err := os.Stat(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("state permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestMatchesKeywords(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		keywords []string
		want     bool
	}{
		{name: "no filter", text: "anything", want: true},
		{name: "case insensitive", text: "How Much is this?", keywords: []string{"how much"}, want: true},
		{name: "unicode", text: "ລາຄາເທົ່າໃດ", keywords: []string{"ລາຄາ"}, want: true},
		{name: "trim keyword", text: "ราคาเท่าไหร่", keywords: []string{" ราคา "}, want: true},
		{name: "not matched", text: "hello", keywords: []string{"price"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesKeywords(tt.text, tt.keywords); got != tt.want {
				t.Fatalf("matchesKeywords() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplyStepsFiltersAndExpandsVariables(t *testing.T) {
	binding := domain.ProductBinding{
		ProductName: "Durian A",
		Price:       "250,000 LAK",
		Description: "Fresh today",
		Messages: []domain.ReplyStep{
			{Code: "intro", Type: "text", Content: "{{product.name}} costs {{product.price}}", Enabled: true},
			{Code: "disabled", Type: "image", Content: "https://example.com/old.jpg", Enabled: false},
			{Code: "detail", Type: "text", Content: "{{product.description}}", Enabled: true},
		},
	}

	want := []domain.ReplyStep{
		{Code: "intro", Type: "text", Content: "Durian A costs 250,000 LAK", Enabled: true},
		{Code: "detail", Type: "text", Content: "Fresh today", Enabled: true},
	}
	if got := replySteps(binding); !reflect.DeepEqual(got, want) {
		t.Fatalf("replySteps() = %#v, want %#v", got, want)
	}
}

func TestReplyStepsUsesProductFallback(t *testing.T) {
	binding := domain.ProductBinding{ProductName: "Durian A", Price: "250,000 LAK", Description: "Fresh today"}
	got := replySteps(binding)
	if len(got) != 1 || got[0].Code != "product" || got[0].Content != "Durian A\nPrice: 250,000 LAK\nFresh today" {
		t.Fatalf("unexpected fallback: %#v", got)
	}
}

func TestResolveReplyItemsExpandsProductVariablesWithoutMutatingSource(t *testing.T) {
	connector := NewGraphConnector(GraphConfig{}, nil)
	connector.products["account-1"] = map[string]domain.Product{
		"product-1": {
			ID:          "product-1",
			Name:        "Durian A",
			Price:       "250,000 LAK",
			Description: "Fresh today",
		},
	}
	rule := &domain.AutomationRule{AccountID: "account-1", ProductID: "product-1"}
	items := []domain.ReplyItem{{
		ID:        "item-1",
		Type:      "text",
		Content:   "{{product.name}} — {{product.price}} — {{product.description}}",
		IsEnabled: true,
	}}

	got := connector.resolveReplyItems(rule, items)
	if got[0].Content != "Durian A — 250,000 LAK — Fresh today" {
		t.Fatalf("resolved content = %q", got[0].Content)
	}
	if items[0].Content != "{{product.name}} — {{product.price}} — {{product.description}}" {
		t.Fatalf("source items were mutated: %#v", items)
	}
	if !got[0].IsEnabled {
		t.Fatal("enabled state was not preserved")
	}
}
