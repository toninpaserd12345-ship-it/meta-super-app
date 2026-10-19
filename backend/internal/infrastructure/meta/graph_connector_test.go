package meta

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/meta-super-app/backend/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestGraphStateSurvivesRestartWithoutPersistingTokens(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "meta-state.json")
	first := NewGraphConnector(GraphConfig{StateFile: stateFile}, nil)
	first.sessions["account-1"] = &graphSession{UserToken: "user-token", Pages: map[string]graphPage{"page-1": {MetaPage: domain.MetaPage{ID: "page-1", Connected: true}, AccessToken: "page-token"}}}
	first.products["account-1"] = map[string]domain.Product{"product-1": {ID: "product-1", Name: "Durian"}}
	first.replyFlows["account-1"] = map[string]domain.ReplyFlow{"flow-1": {ID: "flow-1", Name: "Price reply"}}
	first.bindings["page-1:post:post-1"] = domain.ProductBinding{PageID: "page-1", SourceType: "post", SourceID: "post-1", ProductID: "product-1", FlowID: "flow-1"}
	if err := first.persistStateLocked(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "user-token") || strings.Contains(string(data), "page-token") || strings.Contains(string(data), "sessions") {
		t.Fatalf("state file contains OAuth credentials: %s", data)
	}

	restarted := NewGraphConnector(GraphConfig{StateFile: stateFile}, nil)
	if restarted.sessions["account-1"] != nil {
		t.Fatal("OAuth session must not be restored from the state file")
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

func TestLegacyGraphStateScrubsPlaintextTokens(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "meta-state.json")
	legacy := `{"sessions":{"account-1":{"userToken":"legacy-user-token","pages":{"page-1":{"accessToken":"legacy-page-token"}}}},"bindings":{},"products":{},"replyFlows":{}}`
	if err := os.WriteFile(stateFile, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}

	connector := NewGraphConnector(GraphConfig{StateFile: stateFile}, nil)
	if len(connector.sessions) != 0 {
		t.Fatal("legacy OAuth sessions must be discarded")
	}
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "legacy-user-token") || strings.Contains(string(data), "legacy-page-token") || strings.Contains(string(data), "sessions") {
		t.Fatalf("legacy credentials were not scrubbed: %s", data)
	}
}

func TestTokenEncryptionRoundTrip(t *testing.T) {
	connector := NewGraphConnector(GraphConfig{EncryptionKey: "test-encryption-key-that-is-long-enough"}, nil)
	const token = "EAAB-secret-facebook-token"

	encrypted := connector.encryptToken(token)
	if encrypted == token || !strings.HasPrefix(encrypted, "enc:") {
		t.Fatalf("token was not encrypted: %q", encrypted)
	}
	if strings.Contains(encrypted, token) {
		t.Fatal("ciphertext contains the plaintext token")
	}
	if decrypted := connector.decryptToken(encrypted); decrypted != token {
		t.Fatalf("decryptToken() = %q, want original token", decrypted)
	}
}

func TestTokenEncryptionFailsClosedWithoutKey(t *testing.T) {
	connector := NewGraphConnector(GraphConfig{}, nil)
	if encrypted := connector.encryptToken("must-not-be-plaintext"); encrypted != "" {
		t.Fatalf("encryptToken() = %q, want empty result when no key is configured", encrypted)
	}
}

func TestAuthorizationURLCopiesAccountIDBeforeRetainingOAuthState(t *testing.T) {
	connector := NewGraphConnector(GraphConfig{AppID: "app", RedirectURI: "https://example.com/callback", Version: "v23.0"}, nil)
	requestBuffer := []byte("573f8d7a-789c-452e-b16c-309e1370aef4")
	accountID := unsafe.String(unsafe.SliceData(requestBuffer), len(requestBuffer))

	authorizationURL, err := connector.AuthorizationURL(context.Background(), "user-1", accountID, nil)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	state := parsed.Query().Get("state")
	copy(requestBuffer, []byte("text/html,application/xhtml+xml,appl"))

	if got := connector.states[state].AccountID; got != "573f8d7a-789c-452e-b16c-309e1370aef4" {
		t.Fatalf("retained account ID changed with request buffer: %q", got)
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

func TestSelectedInsightResultPrefersBusinessOutcome(t *testing.T) {
	result := selectedInsightResult(
		[]graphInsightAction{{ActionType: "link_click", Value: "20"}, {ActionType: "lead", Value: "4"}},
		[]graphInsightAction{{ActionType: "link_click", Value: "0.50"}, {ActionType: "lead", Value: "2.50"}},
	)
	if result.Type != "lead" || result.Value == nil || *result.Value != 4 || result.Cost == nil || *result.Cost != 2.5 {
		t.Fatalf("unexpected normalized result: %#v", result)
	}
}

func TestInsightItemPreservesMissingMetrics(t *testing.T) {
	item := insightItem("ad", graphInsightRow{AdID: "ad-1", AdName: "Creative A", Spend: "not-a-number", InlineLinkClickCTR: "2.75"})
	if item.EntityID != "ad-1" || item.EntityName != "Creative A" {
		t.Fatalf("unexpected identity: %#v", item)
	}
	if item.Spend != nil {
		t.Fatalf("invalid spend must remain unavailable, got %v", *item.Spend)
	}
	if item.LinkCTR == nil || *item.LinkCTR != 2.75 {
		t.Fatalf("link CTR was not normalized: %#v", item.LinkCTR)
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

func TestUserTokenRenewsBeforeExpiry(t *testing.T) {
	connector := NewGraphConnector(GraphConfig{AppID: "app", AppSecret: "secret", Version: "v23.0"}, nil)
	connector.sessions["account-1"] = &graphSession{
		UserToken:      "current-token",
		TokenExpiresAt: time.Now().Add(time.Hour).Unix(),
		Pages:          make(map[string]graphPage),
	}
	connector.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `{"access_token":"renewed-token"}`
		if strings.Contains(request.URL.Path, "debug_token") {
			body = `{"data":{"is_valid":true,"expires_at":4102444800,"data_access_expires_at":4102444800,"scopes":["pages_show_list"]}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	token, err := connector.userToken(context.Background(), "account-1")
	if err != nil {
		t.Fatal(err)
	}
	if token != "renewed-token" {
		t.Fatalf("token = %q, want renewed-token", token)
	}
	if connector.sessions["account-1"].TokenExpiresAt != 4102444800 {
		t.Fatal("renewed expiry was not stored")
	}
}

func TestUserTokenRequiresReconnectAfterDataAccessExpiry(t *testing.T) {
	connector := NewGraphConnector(GraphConfig{}, nil)
	connector.sessions["account-1"] = &graphSession{
		UserToken:           "expired-token",
		TokenExpiresAt:      time.Now().Add(time.Hour).Unix(),
		DataAccessExpiresAt: time.Now().Add(-time.Minute).Unix(),
		Pages:               make(map[string]graphPage),
	}

	_, err := connector.userToken(context.Background(), "account-1")
	if !errors.Is(err, domain.ErrMetaReconnectRequired) {
		t.Fatalf("error = %v, want ErrMetaReconnectRequired", err)
	}
}

func TestListAdAccountsMergesBusinessPortfolioAccounts(t *testing.T) {
	connector := NewGraphConnector(GraphConfig{Version: "v23.0"}, nil)
	connector.sessions["account-1"] = &graphSession{UserToken: "token", TokenExpiresAt: time.Now().Add(24 * time.Hour).Unix(), Pages: make(map[string]graphPage)}
	connector.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `{"data":[]}`
		switch {
		case strings.HasSuffix(request.URL.Path, "/me/adaccounts"):
			body = `{"data":[{"id":"act_personal","name":"Personal Ads","account_status":1}]}`
		case strings.HasSuffix(request.URL.Path, "/me/businesses"):
			body = `{"data":[{"id":"business-1","name":"Shop Portfolio"}]}`
		case strings.HasSuffix(request.URL.Path, "/business-1/owned_ad_accounts"):
			body = `{"data":[{"id":"act_business","name":"Shop Ads","account_status":1}]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	items, err := connector.ListAdAccounts(context.Background(), "user-1", "account-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].AccountType != "business" || items[0].BusinessName != "Shop Portfolio" {
		t.Fatalf("business account metadata = %#v", items[0])
	}
	if items[1].AccountType != "personal" {
		t.Fatalf("personal account metadata = %#v", items[1])
	}
}

func TestPagePictureUsesWorkspacePageAndLargeGraphImage(t *testing.T) {
	connector := NewGraphConnector(GraphConfig{Version: "v23.0"}, nil)
	connector.sessions["account-1"] = &graphSession{Pages: map[string]graphPage{
		"page-1": {MetaPage: domain.MetaPage{ID: "page-1", Category: "Retail", PictureURL: "https://stale.example/avatar.jpg"}, AccessToken: "page-token"},
	}}
	connector.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/v23.0/page-1/picture" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if request.URL.Query().Get("type") != "large" || request.URL.Query().Get("access_token") != "page-token" {
			t.Fatalf("query = %q", request.URL.RawQuery)
		}
		header := make(http.Header)
		header.Set("Content-Type", "image/jpeg")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("jpeg-data")), Header: header}, nil
	})}

	picture, err := connector.PagePicture(context.Background(), "account-1", "page-1")
	if err != nil {
		t.Fatal(err)
	}
	if picture.ContentType != "image/jpeg" || string(picture.Data) != "jpeg-data" {
		t.Fatalf("picture = %#v", picture)
	}
}

func TestFetchPagesIncludesWhatsAppDisplayNumber(t *testing.T) {
	connector := NewGraphConnector(GraphConfig{Version: "v23.0"}, nil)
	connector.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `{"data":[]}`
		switch {
		case strings.HasSuffix(request.URL.Path, "/me/accounts"):
			body = `{"data":[]}`
		case strings.HasSuffix(request.URL.Path, "/me/businesses"):
			body = `{"data":[{"id":"business-1"}]}`
		case strings.HasSuffix(request.URL.Path, "/business-1/client_whatsapp_business_accounts"):
			if fields := request.URL.Query().Get("fields"); fields != "id,name" {
				t.Fatalf("WABA fields = %q", fields)
			}
			body = `{"data":[{"id":"waba-1","name":"Shop WhatsApp"}]}`
		case strings.HasSuffix(request.URL.Path, "/waba-1/phone_numbers"):
			body = `{"data":[{"id":"phone-1","display_phone_number":"+856 20 5555 1234","verified_name":"Tonxay Shop"}]}`
		case strings.HasSuffix(request.URL.Path, "/phone-1/whatsapp_business_profile"):
			body = `{"data":[{"profile_picture_url":"https://lookaside.fbsbx.com/avatar.jpg"}]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	pages, diagnostics, err := connector.fetchPages(context.Background(), "user-token")
	if err != nil {
		t.Fatal(err)
	}
	page := pages["phone-1"].MetaPage
	if page.PhoneNumber != "+856 20 5555 1234" || page.Name != "Tonxay Shop" || page.Category != "WhatsApp" || page.PictureURL != "https://lookaside.fbsbx.com/avatar.jpg" {
		t.Fatalf("WhatsApp page = %#v", page)
	}
	if diagnostics.State != "ready" || diagnostics.BusinessCount != 1 || diagnostics.WhatsAppBusinessAccountCount != 1 || diagnostics.PhoneNumberCount != 1 {
		t.Fatalf("WhatsApp diagnostics = %#v", diagnostics)
	}
}

func TestIsAskingPrice(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"User ລາຄາ ທໍ່ໃດ", true},
		{"ລາຄາເທົ່າໃດ", true},
		{"ສົນໃຈ ລາຂາ ທໍໃດນິ", true},
		{"ລາຄ່າ", true},
		{"ຈັກບາດ", true},
		{"ຈັກກີບນິ", true},
		{"ຂາຍແນວໃດ", true},
		{"how much is this?", true},
		{"price please", true},
		{"ib", true},
		{"inbox", true},
		{"ราคาเท่าไหร่ครับ", true},
		{"มีของไหม", true},
		{"ສະບາຍດີ", false},
		{"ok", false},
		{"hello", false},
	}

	for _, c := range cases {
		got := isAskingPrice(c.input)
		if got != c.expected {
			t.Errorf("isAskingPrice(%q) = %v, want %v", c.input, got, c.expected)
		}
	}
}

type mockCommentAutomationProvider struct {
	rule  *domain.AutomationRule
	items []domain.ReplyItem
}

func (m *mockCommentAutomationProvider) FindActiveRuleForTrigger(ctx context.Context, pageID, triggerType, triggerValue string) (*domain.AutomationRule, error) {
	if m.rule != nil && m.rule.TriggerType == triggerType && m.rule.TriggerValue == triggerValue {
		return m.rule, nil
	}
	return nil, nil
}

func (m *mockCommentAutomationProvider) GetKeywordRules(ctx context.Context, pageID string) ([]domain.AutomationRule, error) {
	if m.rule != nil && m.rule.TriggerType == "keyword" {
		return []domain.AutomationRule{*m.rule}, nil
	}
	return nil, nil
}

func (m *mockCommentAutomationProvider) FindReplySetByTrigger(ctx context.Context, accountID, trigger string) (*domain.ReplySet, error) {
	return nil, nil
}

func (m *mockCommentAutomationProvider) GetReplySetItems(ctx context.Context, replySetID string, accountID string) ([]domain.ReplyItem, error) {
	return m.items, nil
}

func (m *mockCommentAutomationProvider) GetRule(ctx context.Context, id, accountID string) (*domain.AutomationRule, error) {
	if m.rule != nil && m.rule.ID == id {
		return m.rule, nil
	}
	return nil, nil
}

func signPayload(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestCommentAutoReplyWithKeywordAndPostbackUnlock(t *testing.T) {
	var reqMu sync.Mutex
	var requestedURLs []string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		reqMu.Lock()
		requestedURLs = append(requestedURLs, req.Method+" "+req.URL.String())
		reqMu.Unlock()
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"success":true,"message_id":"mid.123"}`)),
			Header:     make(http.Header),
		}, nil
	})

	const appSecret = "test-secret"
	g := NewGraphConnector(GraphConfig{Version: "v22.0", AppSecret: appSecret}, nil)
	g.client = &http.Client{Transport: transport}
	g.sessions["acc-1"] = &graphSession{
		UserToken: "user-tok",
		Pages: map[string]graphPage{
			"page-1": {
				MetaPage:    domain.MetaPage{ID: "page-1", Connected: true},
				AccessToken: "page-tok",
			},
		},
	}

	testRule := &domain.AutomationRule{
		ID:           "rule-100",
		AccountID:    "acc-1",
		PageID:       "page-1",
		TriggerType:  "keyword",
		TriggerValue: "ສົນໃຈ",
		ReplySetID:   "set-1",
		IsActive:     true,
	}
	testItems := []domain.ReplyItem{
		{ID: "item-1", Type: "text", Content: "ເສື້ອລຸ້ນໃໝ່ 150,000 ກີບ", IsEnabled: true},
		{ID: "item-2", Type: "image", Content: "https://example.com/shirt.jpg", IsEnabled: true},
	}

	mockAuto := &mockCommentAutomationProvider{
		rule:  testRule,
		items: testItems,
	}
	g.SetAutomationProvider(mockAuto)

	// 1. Comment from page itself should be ignored
	selfCommentPayload := []byte(`{"object":"page","entry":[{"id":"page-1","changes":[{"field":"feed","value":{"item":"comment","verb":"add","post_id":"post-1","comment_id":"c-self","message":"ສົນໃຈ","from":{"id":"page-1"}}}]}]}`)
	if err := g.ReceiveWebhook(context.Background(), selfCommentPayload, signPayload(appSecret, selfCommentPayload)); err != nil {
		t.Fatalf("webhook error: %v", err)
	}
	reqMu.Lock()
	selfCount := len(requestedURLs)
	reqMu.Unlock()
	if selfCount != 0 {
		t.Fatalf("expected 0 requests for self comment, got %d", selfCount)
	}

	// 2. Customer comment with keyword "ສົນໃຈ" should trigger like, public reply, and private reply
	customerCommentPayload := []byte(`{"object":"page","entry":[{"id":"page-1","changes":[{"field":"feed","value":{"item":"comment","verb":"add","post_id":"post-1","comment_id":"c-cust-1","message":"ຂ້ອຍສົນໃຈໂຕນີ້","from":{"id":"user-999"}}}]}]}`)
	if err := g.ReceiveWebhook(context.Background(), customerCommentPayload, signPayload(appSecret, customerCommentPayload)); err != nil {
		t.Fatalf("webhook error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	reqMu.Lock()
	urls := make([]string, len(requestedURLs))
	copy(urls, requestedURLs)
	reqMu.Unlock()

	if len(urls) < 3 {
		t.Fatalf("expected at least 3 requests (like, comment reply, private reply), got %d: %v", len(urls), urls)
	}

	hasLike := false
	hasCommentReply := false
	hasPrivateMessage := false
	for _, u := range urls {
		if strings.Contains(u, "/likes") {
			hasLike = true
		}
		if strings.Contains(u, "/comments") {
			hasCommentReply = true
		}
		if strings.Contains(u, "/messages") || strings.Contains(u, "/private_replies") {
			hasPrivateMessage = true
		}
	}
	if !hasLike {
		t.Errorf("missing like request: %v", urls)
	}
	if !hasCommentReply {
		t.Errorf("missing public comment reply request: %v", urls)
	}
	if !hasPrivateMessage {
		t.Errorf("missing private reply request: %v", urls)
	}

	// 3. Customer clicks button which triggers postback COMMENT_UNLOCK
	postbackPayload := []byte(`{"object":"page","entry":[{"id":"page-1","messaging":[{"sender":{"id":"user-999"},"recipient":{"id":"page-1"},"timestamp":123456789,"postback":{"title":"📦 ເບິ່ງລາຍລະອຽດເຕັມ","payload":"COMMENT_UNLOCK:rule-100:c-cust-1"}}]}]}`)
	if err := g.ReceiveWebhook(context.Background(), postbackPayload, signPayload(appSecret, postbackPayload)); err != nil {
		t.Fatalf("postback webhook error: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
}

type mockMultiAutomationProvider struct {
	rules map[string]*domain.AutomationRule
	items map[string][]domain.ReplyItem
}

func (m *mockMultiAutomationProvider) FindActiveRuleForTrigger(ctx context.Context, pageID, triggerType, triggerValue string) (*domain.AutomationRule, error) {
	key := triggerType + ":" + triggerValue
	if r, ok := m.rules[key]; ok {
		return r, nil
	}
	return nil, nil
}

func (m *mockMultiAutomationProvider) GetKeywordRules(ctx context.Context, pageID string) ([]domain.AutomationRule, error) {
	var list []domain.AutomationRule
	for _, r := range m.rules {
		if r.TriggerType == "keyword" {
			list = append(list, *r)
		}
	}
	return list, nil
}

func (m *mockMultiAutomationProvider) FindReplySetByTrigger(ctx context.Context, accountID, trigger string) (*domain.ReplySet, error) {
	return nil, nil
}

func (m *mockMultiAutomationProvider) GetReplySetItems(ctx context.Context, replySetID string, accountID string) ([]domain.ReplyItem, error) {
	return m.items[replySetID], nil
}

func (m *mockMultiAutomationProvider) GetRule(ctx context.Context, id, accountID string) (*domain.AutomationRule, error) {
	for _, r := range m.rules {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}

func TestAdReferralPriorityAndCommentTextReply(t *testing.T) {
	var reqMu sync.Mutex
	var sentRequests []string
	var sentFormMessages []string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		reqMu.Lock()
		urlStr := req.Method + " " + req.URL.String()
		sentRequests = append(sentRequests, urlStr)
		if strings.Contains(urlStr, "/private_replies") {
			_ = req.ParseForm()
			sentFormMessages = append(sentFormMessages, req.Form.Get("message"))
		}
		reqMu.Unlock()
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"success":true,"message_id":"mid.456"}`)),
			Header:     make(http.Header),
		}, nil
	})

	const appSecret = "test-secret-2"
	g := NewGraphConnector(GraphConfig{Version: "v22.0", AppSecret: appSecret}, nil)
	g.client = &http.Client{Transport: transport}
	g.sessions["acc-2"] = &graphSession{
		UserToken: "user-tok",
		Pages: map[string]graphPage{
			"page-10": {
				MetaPage:    domain.MetaPage{ID: "page-10", Connected: true},
				AccessToken: "page-tok-10",
			},
		},
	}

	adRule := &domain.AutomationRule{
		ID:           "rule-ad-1",
		AccountID:    "acc-2",
		PageID:       "page-10",
		TriggerType:  "ad",
		TriggerValue: "12022334455",
		ProductID:    "prod-shoes",
		ReplySetID:   "reply-shoes",
		FlowID:       "flow-shoes",
		IsActive:     true,
	}
	postRule := &domain.AutomationRule{
		ID:               "rule-post-2",
		AccountID:        "acc-2",
		PageID:           "page-10",
		TriggerType:      "post",
		TriggerValue:     "post-777",
		ProductID:        "prod-shirt",
		ReplySetID:       "reply-shirt",
		FlowID:           "flow-shirt",
		CommentReplyText: "ສະບາຍດີ! ແຈ້ງລາຄາທາງ Inbox ແລ້ວເດີ້",
		IsActive:         true,
	}

	mockAuto := &mockMultiAutomationProvider{
		rules: map[string]*domain.AutomationRule{
			"ad:12022334455": adRule,
			"post:post-777":  postRule,
		},
		items: map[string][]domain.ReplyItem{
			"reply-shoes": {
				{ID: "s-1", Type: "text", Content: "ເກີບແຕະ Kito ແທ້ ລາຄາ 120,000 ກີບ", IsEnabled: true},
			},
			"reply-shirt": {
				{ID: "sh-1", Type: "text", Content: "ເສື້ອຍືດພຣີມ່ຽມ ລາຄາ 99,000 ກີບ ພ້ອມສົ່ງ", IsEnabled: true},
			},
		},
	}
	g.SetAutomationProvider(mockAuto)

	// Test 1: Customer clicks Facebook Ad and sends message "ສະບາຍດີ ລາຄາເທົ່າໃດ"
	// Ad referral must be matched and shoes reply set delivered!
	adPayload := []byte(`{"object":"page","entry":[{"id":"page-10","messaging":[{"sender":{"id":"customer-ad-user"},"recipient":{"id":"page-10"},"timestamp":123456789,"message":{"mid":"mid-ad-1","text":"ສະບາຍດີ ລາຄາເທົ່າໃດ"},"referral":{"source":"ADS","type":"OPEN_THREAD","ad_id":"12022334455"}}]}]}`)
	if err := g.ReceiveWebhook(context.Background(), adPayload, signPayload(appSecret, adPayload)); err != nil {
		t.Fatalf("webhook error: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Verify conversation context was bound to ad rule
	g.mu.RLock()
	boundContext := g.conversations["page-10:customer-ad-user"]
	g.mu.RUnlock()
	expectedBinding := "page-10:ad:12022334455"
	if boundContext != expectedBinding {
		t.Errorf("ad conversation binding = %q, want %q", boundContext, expectedBinding)
	}

	// Test 2: Comment on post-777 must send exact text reply via private_replies
	commentPayload := []byte(`{"object":"page","entry":[{"id":"page-10","changes":[{"field":"feed","value":{"item":"comment","verb":"add","post_id":"post-777","comment_id":"comm-123","message":"ສົນໃຈເສື້ອໂຕນີ້","from":{"id":"customer-commenter"}}}]}]}`)
	if err := g.ReceiveWebhook(context.Background(), commentPayload, signPayload(appSecret, commentPayload)); err != nil {
		t.Fatalf("webhook comment error: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	reqMu.Lock()
	requestsCopy := make([]string, len(sentRequests))
	copy(requestsCopy, sentRequests)
	messagesCopy := make([]string, len(sentFormMessages))
	copy(messagesCopy, sentFormMessages)
	reqMu.Unlock()

	foundPrivateReply := false
	for _, req := range requestsCopy {
		if strings.Contains(req, "/comm-123/private_replies") {
			foundPrivateReply = true
			break
		}
	}
	if !foundPrivateReply {
		t.Errorf("expected private_replies call for comm-123, got: %v", requestsCopy)
	}

	foundExactText := false
	for _, msg := range messagesCopy {
		if strings.Contains(msg, "ເສື້ອຍືດພຣີມ່ຽມ ລາຄາ 99,000 ກີບ") {
			foundExactText = true
			break
		}
	}
	if !foundExactText {
		t.Errorf("expected private reply to contain configured reply set text, got: %v", messagesCopy)
	}

	// Verify commenter's conversation context is bound to post-777
	g.mu.RLock()
	commenterBinding := g.conversations["page-10:customer-commenter"]
	g.mu.RUnlock()
	expectedCommentBinding := "page-10:post:post-777"
	if commenterBinding != expectedCommentBinding {
		t.Errorf("commenter conversation binding = %q, want %q", commenterBinding, expectedCommentBinding)
	}
}

func TestUnlinkedCampaignDoesNotTriggerAutoReply(t *testing.T) {
	var reqMu sync.Mutex
	var sentRequests []string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		reqMu.Lock()
		sentRequests = append(sentRequests, req.Method+" "+req.URL.String())
		reqMu.Unlock()
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"success":true,"message_id":"mid.789"}`)),
			Header:     make(http.Header),
		}, nil
	})

	const appSecret = "test-secret-unlinked"
	g := NewGraphConnector(GraphConfig{Version: "v22.0", AppSecret: appSecret}, nil)
	g.client = &http.Client{Transport: transport}
	g.sessions["acc-3"] = &graphSession{
		UserToken: "user-tok",
		Pages: map[string]graphPage{
			"page-20": {
				MetaPage:    domain.MetaPage{ID: "page-20", Connected: true},
				AccessToken: "page-tok-20",
			},
		},
	}

	// Database has an active rule for KITO shoes linked to campaign-kito
	kitoRule := &domain.AutomationRule{
		ID:           "rule-kito-1",
		AccountID:    "acc-3",
		PageID:       "page-20",
		TriggerType:  "campaign",
		TriggerValue: "campaign-kito",
		ProductID:    "prod-kito",
		ReplySetID:   "reply-kito",
		FlowID:       "flow-kito",
		IsActive:     true,
	}

	mockAuto := &mockMultiAutomationProvider{
		rules: map[string]*domain.AutomationRule{
			"campaign:campaign-kito": kitoRule,
		},
		items: map[string][]domain.ReplyItem{
			"reply-kito": {
				{ID: "k-1", Type: "image", Content: "https://example.com/kito.jpg", IsEnabled: true},
				{ID: "k-2", Type: "text", Content: "ເກີບ Kito ແທ້ 100%", IsEnabled: true},
			},
		},
	}
	g.SetAutomationProvider(mockAuto)

	// Customer interacts with a BRAND NEW UNLINKED campaign (campaign-brand-new-999)
	unlinkedAdPayload := []byte(`{"object":"page","entry":[{"id":"page-20","messaging":[{"sender":{"id":"customer-new-campaign"},"recipient":{"id":"page-20"},"timestamp":123456789,"message":{"mid":"mid-new-1","text":"ສະບາຍດີ ລາຄາເທົ່າໃດ"},"referral":{"source":"ADS","type":"OPEN_THREAD","campaign_id":"campaign-brand-new-999","ad_id":"ad-brand-new-888"}}]}]}`)
	if err := g.ReceiveWebhook(context.Background(), unlinkedAdPayload, signPayload(appSecret, unlinkedAdPayload)); err != nil {
		t.Fatalf("webhook error: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	reqMu.Lock()
	requestsCopy := make([]string, len(sentRequests))
	copy(requestsCopy, sentRequests)
	reqMu.Unlock()

	// Must NOT send any message! (Should be 0 outbound calls to /messages)
	for _, req := range requestsCopy {
		if strings.Contains(req, "/messages") {
			t.Fatalf("CRITICAL ERROR: Unlinked campaign triggered an automated message: %v", req)
		}
	}

	// Customer comments on an UNLINKED post (post-brand-new-777)
	unlinkedCommentPayload := []byte(`{"object":"page","entry":[{"id":"page-20","changes":[{"field":"feed","value":{"item":"comment","verb":"add","post_id":"post-brand-new-777","comment_id":"comm-new-555","message":"ສົນໃຈ","from":{"id":"commenter-new"}}}]}]}`)
	if err := g.ReceiveWebhook(context.Background(), unlinkedCommentPayload, signPayload(appSecret, unlinkedCommentPayload)); err != nil {
		t.Fatalf("comment webhook error: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	reqMu.Lock()
	requestsCopy = make([]string, len(sentRequests))
	copy(requestsCopy, sentRequests)
	reqMu.Unlock()

	// Must NOT send any private reply or comment reply for Kito!
	for _, req := range requestsCopy {
		if strings.Contains(req, "/private_replies") {
			t.Fatalf("CRITICAL ERROR: Unlinked post triggered a private reply: %v", req)
		}
	}
}



