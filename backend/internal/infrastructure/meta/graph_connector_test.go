package meta

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
		case strings.HasSuffix(request.URL.Path, "/business-1/owned_whatsapp_business_accounts"):
			body = `{"data":[{"id":"waba-1","name":"Shop WhatsApp","profile_picture_url":"https://lookaside.fbsbx.com/avatar.jpg"}]}`
		case strings.HasSuffix(request.URL.Path, "/waba-1/phone_numbers"):
			body = `{"data":[{"id":"phone-1","display_phone_number":"+856 20 5555 1234","verified_name":"Tonxay Shop"}]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	pages, err := connector.fetchPages(context.Background(), "user-token")
	if err != nil {
		t.Fatal(err)
	}
	page := pages["phone-1"].MetaPage
	if page.PhoneNumber != "+856 20 5555 1234" || page.Name != "Tonxay Shop" || page.Category != "WhatsApp" {
		t.Fatalf("WhatsApp page = %#v", page)
	}
}
