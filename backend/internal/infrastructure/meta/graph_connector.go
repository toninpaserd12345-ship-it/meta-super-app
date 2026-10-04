package meta

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/usecase"
)

type GraphConfig struct {
	AppID, AppSecret, RedirectURI, Version, WebhookFields, WebhookVerifyToken, StateFile string
}

type oauthState struct {
	UserID, AccountID string
	Scopes            []string
	ExpiresAt         time.Time
}

type graphSession struct {
	UserToken                           string
	TokenExpiresAt, DataAccessExpiresAt int64
	GrantedPermissions                  []string
	Pages                               map[string]graphPage
}

type graphPage struct {
	domain.MetaPage
	AccessToken string
}

type GraphConnector struct {
	chatStream *usecase.ChatStream
	cfg           GraphConfig
	client        *http.Client
	mu            sync.RWMutex
	states        map[string]oauthState
	sessions      map[string]*graphSession
	recentEvents  map[string]time.Time
	bindings      map[string]domain.ProductBinding
	products      map[string]map[string]domain.Product
	replyFlows    map[string]map[string]domain.ReplyFlow
	firstReplies  map[string]time.Time
	conversations map[string]string
	deliverySteps map[string]bool
	inFlight      map[string]bool
	automation    domain.AutomationProvider
}

func (g *GraphConnector) SetChatStream(c *usecase.ChatStream) {
	g.chatStream = c
}

func (g *GraphConnector) SetAutomationProvider(p domain.AutomationProvider) {
	g.automation = p
}

func NewGraphConnector(cfg GraphConfig) *GraphConnector {
	g := &GraphConnector{cfg: cfg, client: &http.Client{Timeout: 15 * time.Second}, states: make(map[string]oauthState), sessions: make(map[string]*graphSession), recentEvents: make(map[string]time.Time), bindings: make(map[string]domain.ProductBinding), products: make(map[string]map[string]domain.Product), replyFlows: make(map[string]map[string]domain.ReplyFlow), firstReplies: make(map[string]time.Time), conversations: make(map[string]string), deliverySteps: make(map[string]bool), inFlight: make(map[string]bool)}
	if err := g.loadState(); err != nil {
		slog.Warn("unable to load Meta state", "error", err)
	}
	return g
}

type persistentGraphState struct {
	Sessions   map[string]*graphSession               `json:"sessions"`
	Bindings   map[string]domain.ProductBinding       `json:"bindings"`
	Products   map[string]map[string]domain.Product   `json:"products"`
	ReplyFlows map[string]map[string]domain.ReplyFlow `json:"replyFlows"`
}

func (g *GraphConnector) loadState() error {
	if strings.TrimSpace(g.cfg.StateFile) == "" {
		return nil
	}
	data, err := os.ReadFile(g.cfg.StateFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var state persistentGraphState
	if err = json.Unmarshal(data, &state); err != nil {
		return err
	}
	if state.Sessions != nil {
		g.sessions = state.Sessions
	}
	if state.Bindings != nil {
		g.bindings = state.Bindings
	}
	if state.Products != nil {
		g.products = state.Products
	}
	if state.ReplyFlows != nil {
		g.replyFlows = state.ReplyFlows
	}
	return nil
}

// persistStateLocked atomically saves durable configuration and OAuth tokens.
// The caller must hold g.mu. The file is user-readable only.
func (g *GraphConnector) persistStateLocked() error {
	if strings.TrimSpace(g.cfg.StateFile) == "" {
		return nil
	}
	state := persistentGraphState{Sessions: g.sessions, Bindings: g.bindings, Products: g.products, ReplyFlows: g.replyFlows}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	dir := filepath.Dir(g.cfg.StateFile)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".meta-state-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, g.cfg.StateFile)
}

func (g *GraphConnector) VerifyWebhook(token string) bool {
	return g.cfg.WebhookVerifyToken != "" && hmac.Equal([]byte(token), []byte(g.cfg.WebhookVerifyToken))
}

func (g *GraphConnector) ReceiveWebhook(ctx context.Context, body []byte, signature string) error {
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return errors.New("webhook signature is missing")
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
	if err != nil {
		return errors.New("webhook signature is invalid")
	}
	mac := hmac.New(sha256.New, []byte(g.cfg.AppSecret))
	_, _ = mac.Write(body)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return errors.New("webhook signature does not match")
	}
	var objectProbe struct {
		Object string `json:"object"`
	}
	_ = json.Unmarshal(body, &objectProbe)
	if objectProbe.Object == "whatsapp_business_account" {
		return g.processWhatsAppWebhook(ctx, body)
	}

	var event struct {
		Object string `json:"object"`
		Entry  []struct {
			ID        string `json:"id"`
			Messaging []struct {
				Sender struct {
					ID string `json:"id"`
				} `json:"sender"`
				Message struct {
					MID      string           `json:"mid"`
					Text     string           `json:"text"`
					Referral *webhookReferral `json:"referral"`
				} `json:"message"`
				Referral *webhookReferral `json:"referral"`
				Postback struct {
					Referral *webhookReferral `json:"referral"`
				} `json:"postback"`
			} `json:"messaging"`
			Changes []struct {
				Field string `json:"field"`
				Value struct {
					Item      string `json:"item"`
					Verb      string `json:"verb"`
					PostID    string `json:"post_id"`
					CommentID string `json:"comment_id"`
					Message   string `json:"message"`
					From      struct {
						ID string `json:"id"`
					} `json:"from"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}
	if err = json.Unmarshal(body, &event); err != nil {
		return errors.New("webhook payload is invalid")
	}
	if event.Object != "page" {
		return errors.New("unsupported webhook object")
	}
	slog.Info("Webhook received from Meta:")
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
		fmt.Println(prettyJSON.String())
	} else {
		fmt.Println(string(body))
	}
	digest := sha256.Sum256(body)
	key := hex.EncodeToString(digest[:])
	g.mu.Lock()
	now := time.Now()
	for id, expires := range g.recentEvents {
		if now.After(expires) {
			delete(g.recentEvents, id)
		}
	}
	if _, duplicate := g.recentEvents[key]; duplicate {
		g.mu.Unlock()
		return nil
	}
	g.recentEvents[key] = now.Add(10 * time.Minute)
	g.mu.Unlock()
	for _, entry := range event.Entry {
		for _, change := range entry.Changes {
			if change.Field != "feed" || change.Value.Item != "comment" || change.Value.Verb != "add" || change.Value.PostID == "" || change.Value.CommentID == "" || change.Value.From.ID == "" {
				continue
			}
			seenKey := entry.ID + ":" + change.Value.From.ID
			var rule *domain.AutomationRule
			if g.automation != nil {
				rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, "post", change.Value.PostID)
			}
			found := rule != nil

			g.mu.Lock()
			lastReplyTime, alreadyReplied := g.firstReplies[seenKey]
			isSpamming := false
			if alreadyReplied {
				isSpamming = time.Since(lastReplyTime) < 24*time.Hour
			}

			busy := g.inFlight[seenKey]
			if found && !isSpamming && !busy {
				g.inFlight[seenKey] = true
			}
			g.mu.Unlock()
			_, page, pageFound := g.findPage(entry.ID)
			if !found || isSpamming || busy || !pageFound {
				continue
			}

			var items []domain.ReplyItem
			if g.automation != nil {
				items, _ = g.automation.GetReplySetItems(ctx, rule.ReplySetID, rule.AccountID)
			}
			items = g.resolveReplyItems(rule, items)

			texts := []string{}
			for _, step := range items {
				if step.IsEnabled && step.Type == "text" {
					texts = append(texts, step.Content)
				}
			}
			if len(texts) == 0 {
				g.releaseReply(seenKey)
				continue
			}
			text := strings.Join(texts, "\n\n")

			// Reply to comment publicly
			if err := g.replyToComment(ctx, page, change.Value.CommentID, text); err != nil {
				slog.Error("automatic comment reply failed", "page_id", entry.ID, "post_id", change.Value.PostID, "comment_id", change.Value.CommentID, "error", err)
			}

			// Reply privately
			if err := g.sendPrivateReply(ctx, page, change.Value.CommentID, text); err != nil {
				slog.Error("automatic private product reply failed", "page_id", entry.ID, "post_id", change.Value.PostID, "comment_id", change.Value.CommentID, "error", err)
				g.releaseReply(seenKey)
				continue
			}
			g.mu.Lock()
			g.firstReplies[seenKey] = time.Now()
			delete(g.inFlight, seenKey)
			g.mu.Unlock()
			slog.Info("automatic private product reply sent", "page_id", entry.ID, "post_id", change.Value.PostID, "comment_id", change.Value.CommentID)
		}
		for _, message := range entry.Messaging {
			ref := message.Message.Referral
			if ref == nil {
				ref = message.Referral
			}
			if ref == nil {
				ref = message.Postback.Referral
			}
			if message.Sender.ID == "" {
				continue
			}
			seenKey := entry.ID + ":" + message.Sender.ID
			sourceType, sourceID, resolvedKey := "", "", ""
			if ref != nil {
				sourceType, sourceID = "post", ref.Ref
				if ref.AdID != "" {
					sourceType, sourceID = "ad", ref.AdID
				}
				if sourceID != "" {
					resolvedKey = bindingKey(entry.ID, sourceType, sourceID)
				}
			} else {
				g.mu.RLock()
				resolvedKey = g.conversations[seenKey]
				g.mu.RUnlock()
			}

			var rule *domain.AutomationRule
			if g.automation != nil {
				if ref != nil {
					// Try fetching by post or ad
					rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, sourceType, sourceID)
				} else {
					// Check for keyword trigger
					msgText := strings.TrimSpace(message.Message.Text)
					if msgText != "" {
						keywordRules, _ := g.automation.GetKeywordRules(ctx, entry.ID)
						for _, kr := range keywordRules {
							if matchesKeywords(msgText, []string{kr.TriggerValue}) {
								rule = &kr
								break
							}
						}
					}
				}
			}

			found := rule != nil
			if !found && resolvedKey == "" {
				continue
			}

			g.mu.Lock()
			if found && ref != nil {
				g.conversations[seenKey] = resolvedKey
			}
			lastReplyTime, alreadyReplied := g.firstReplies[seenKey]

			// For keyword triggers, we only rate-limit to once every 1 minute to avoid spamming
			// For ad/post triggers (first-message), we rate limit to once every 24 hours.
			isSpamming := false
			if alreadyReplied {
				if rule != nil && rule.TriggerType == "keyword" {
					isSpamming = time.Since(lastReplyTime) < time.Minute
				} else {
					isSpamming = time.Since(lastReplyTime) < 24*time.Hour
				}
			}

			busy := g.inFlight[seenKey]
			if found && !isSpamming && !busy {
				g.inFlight[seenKey] = true
			}
			g.mu.Unlock()
			_, page, pageFound := g.findPage(entry.ID)
			if !found || isSpamming || busy || !pageFound {
				continue
			}

			var items []domain.ReplyItem
			if g.automation != nil {
				items, _ = g.automation.GetReplySetItems(ctx, rule.ReplySetID, rule.AccountID)
			}
			items = g.resolveReplyItems(rule, items)

			sendFailed := false
			for index, step := range items {
				if !step.IsEnabled {
					continue
				}
				stepKey := seenKey + ":" + resolvedKey + ":" + step.ID
				g.mu.RLock()
				stepDone := g.deliverySteps[stepKey]
				g.mu.RUnlock()
				if stepDone {
					continue
				}

				// Add a slight delay between sending multiple steps to simulate typing/make it feel natural
				if index > 0 {
					time.Sleep(1500 * time.Millisecond)
				}

				var err error
				if step.Type == "text" {
					err = g.sendMessage(ctx, page, message.Sender.ID, step.Content)
				} else {
					err = g.sendMedia(ctx, page, message.Sender.ID, step.Type, step.Content)
				}
				if err != nil {
					slog.Error("automatic flow message failed", "page_id", entry.ID, "sender_id", message.Sender.ID, "source_id", sourceID, "step", index+1, "error", err)
					sendFailed = true
					break
				}
				g.mu.Lock()
				g.deliverySteps[stepKey] = true
				g.mu.Unlock()
			}
			if sendFailed {
				g.releaseReply(seenKey)
				continue
			}
			g.mu.Lock()
			g.firstReplies[seenKey] = time.Now()
			delete(g.inFlight, seenKey)
			g.mu.Unlock()
			slog.Info("automatic product reply sent", "page_id", entry.ID, "sender_id", message.Sender.ID, "source_type", sourceType, "source_id", sourceID)
		}
	}
	return nil
}

type webhookReferral struct {
	Ref    string `json:"ref"`
	Source string `json:"source"`
	AdID   string `json:"ad_id"`
}

func bindingKey(pageID, sourceType, sourceID string) string {
	return pageID + ":" + sourceType + ":" + sourceID
}

func (g *GraphConnector) releaseReply(seenKey string) {
	g.mu.Lock()
	delete(g.inFlight, seenKey)
	g.mu.Unlock()
}

func (g *GraphConnector) resolveReplyItems(rule *domain.AutomationRule, items []domain.ReplyItem) []domain.ReplyItem {
	if rule == nil || rule.ProductID == "" {
		return append([]domain.ReplyItem(nil), items...)
	}
	g.mu.RLock()
	product, found := g.products[rule.AccountID][rule.ProductID]
	g.mu.RUnlock()
	resolved := make([]domain.ReplyItem, len(items))
	copy(resolved, items)
	if !found {
		return resolved
	}
	for index := range resolved {
		resolved[index].Content = strings.ReplaceAll(resolved[index].Content, "{{product.name}}", product.Name)
		resolved[index].Content = strings.ReplaceAll(resolved[index].Content, "{{product.price}}", product.Price)
		resolved[index].Content = strings.ReplaceAll(resolved[index].Content, "{{product.description}}", product.Description)
	}
	return resolved
}

func productReply(binding domain.ProductBinding) string {
	text := binding.ProductName + "\nPrice: " + binding.Price
	if strings.TrimSpace(binding.Description) != "" {
		text += "\n" + binding.Description
	}
	return text
}

func replySteps(binding domain.ProductBinding) []domain.ReplyStep {
	if len(binding.Messages) == 0 {
		return []domain.ReplyStep{{Code: "product", Type: "text", Content: productReply(binding), Enabled: true}}
	}
	result := make([]domain.ReplyStep, 0, len(binding.Messages))
	for _, step := range binding.Messages {
		if !step.Enabled {
			continue
		}
		step.Content = strings.ReplaceAll(step.Content, "{{product.name}}", binding.ProductName)
		step.Content = strings.ReplaceAll(step.Content, "{{product.price}}", binding.Price)
		step.Content = strings.ReplaceAll(step.Content, "{{product.description}}", binding.Description)
		result = append(result, step)
	}
	return result
}

func matchesKeywords(text string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	text = strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(strings.TrimSpace(keyword))) {
			return true
		}
	}
	return false
}

func (g *GraphConnector) findPage(pageID string) (string, graphPage, bool) {
	for accID, session := range g.sessions {
		if page, ok := session.Pages[pageID]; ok {
			return accID, page, true
		}
	}
	return "", graphPage{}, false
}

func (g *GraphConnector) AuthorizationURL(_ context.Context, userID, accountID string, requested []string) (string, error) {
	allowed := map[string]bool{"pages_show_list": true, "pages_manage_metadata": true, "pages_read_engagement": true, "pages_messaging": true, "pages_manage_posts": true, "read_insights": true, "ads_read": true, "business_management": true, "whatsapp_business_management": true, "whatsapp_business_messaging": true}
	base := []string{"pages_show_list", "pages_manage_metadata", "pages_read_engagement", "pages_messaging", "ads_read", "business_management", "whatsapp_business_management", "whatsapp_business_messaging"}
	seen := map[string]bool{}
	scopes := make([]string, 0, len(base)+len(requested))
	for _, scope := range append(base, requested...) {
		if allowed[scope] && !seen[scope] {
			seen[scope] = true
			scopes = append(scopes, scope)
		}
	}
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	g.mu.Lock()
	for key, item := range g.states {
		if time.Now().After(item.ExpiresAt) {
			delete(g.states, key)
		}
	}
	g.states[state] = oauthState{UserID: userID, AccountID: accountID, Scopes: scopes, ExpiresAt: time.Now().Add(30 * time.Minute)}
	g.mu.Unlock()
	query := url.Values{
		"client_id": {g.cfg.AppID}, "redirect_uri": {g.cfg.RedirectURI}, "state": {state},
		"response_type": {"code"}, "scope": {strings.Join(scopes, ",")},
	}
	return fmt.Sprintf("https://www.facebook.com/%s/dialog/oauth?%s", g.cfg.Version, query.Encode()), nil
}

func (g *GraphConnector) CompleteAuthorization(ctx context.Context, state, code string) (*domain.MetaOAuthResult, error) {
	g.mu.Lock()
	pending, ok := g.states[state]
	delete(g.states, state)
	g.mu.Unlock()
	if !ok || time.Now().After(pending.ExpiresAt) {
		return nil, errors.New("oauth state is invalid or expired")
	}
	query := url.Values{"client_id": {g.cfg.AppID}, "client_secret": {g.cfg.AppSecret}, "redirect_uri": {g.cfg.RedirectURI}, "code": {code}}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/oauth/access_token?"+query.Encode(), &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, errors.New("meta returned an empty access token")
	}
	longQuery := url.Values{"grant_type": {"fb_exchange_token"}, "client_id": {g.cfg.AppID}, "client_secret": {g.cfg.AppSecret}, "fb_exchange_token": {token.AccessToken}}
	var longToken struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/oauth/access_token?"+longQuery.Encode(), &longToken); err != nil {
		return nil, fmt.Errorf("exchange long-lived token: %w", err)
	}
	if longToken.AccessToken != "" {
		token.AccessToken = longToken.AccessToken
	}
	debug, err := g.debugToken(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("debug access token: %w", err)
	}
	if !debug.Data.IsValid {
		return nil, errors.New("meta access token is invalid")
	}
	pages, err := g.fetchPages(ctx, token.AccessToken)
	if err != nil {
		return nil, err
	}
	g.mu.Lock()
	for id, page := range pages {
		page.TokenExpiresAt = debug.Data.ExpiresAt
		page.DataAccessExpiresAt = debug.Data.DataAccessExpiresAt
		page.GrantedPermissions = append([]string(nil), debug.Data.Scopes...)
		pages[id] = page
	}
	var profile struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	_ = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me?fields=id,name,email&access_token="+token.AccessToken, &profile)
	if pending.AccountID != "" {
		g.sessions[pending.AccountID] = &graphSession{UserToken: token.AccessToken, TokenExpiresAt: debug.Data.ExpiresAt, DataAccessExpiresAt: debug.Data.DataAccessExpiresAt, GrantedPermissions: append([]string(nil), debug.Data.Scopes...), Pages: pages}
		_ = g.persistStateLocked()
	}
	g.mu.Unlock()
	return &domain.MetaOAuthResult{UserID: pending.UserID, AccountID: pending.AccountID, AccessToken: token.AccessToken, FacebookID: profile.ID, Name: profile.Name, Email: profile.Email}, nil
}

func (g *GraphConnector) ListPages(ctx context.Context, _ string, accountID string) ([]domain.MetaPage, error) {
	g.mu.RLock()
	session := g.sessions[accountID]
	g.mu.RUnlock()
	if session == nil {
		return []domain.MetaPage{}, nil
	}
	pages, err := g.fetchPages(ctx, session.UserToken)
	if err != nil {
		return nil, err
	}
	for id, page := range pages {
		page.TokenExpiresAt = session.TokenExpiresAt
		page.DataAccessExpiresAt = session.DataAccessExpiresAt
		page.GrantedPermissions = append([]string(nil), session.GrantedPermissions...)
		pages[id] = page
	}
	// Meta is the source of truth for webhook subscriptions. The local
	// connector may restart while subscriptions remain active on Meta, so sync
	// every Page instead of relying only on the in-memory Connected flag.
	for id, page := range pages {
		connected, subscriptionErr := g.isPageSubscribed(ctx, page)
		if subscriptionErr != nil {
			return nil, subscriptionErr
		}
		page.Connected = connected
		if connected {
			page.WebhookStatus = "subscribed"
		} else {
			page.WebhookStatus = "disconnected"
		}
		pages[id] = page
	}
	g.mu.Lock()
	for id, page := range pages {
		session.Pages[id] = page
	}
	if err := g.persistStateLocked(); err != nil {
		slog.Warn("unable to persist refreshed Meta Pages", "error", err)
	}
	g.mu.Unlock()
	result := make([]domain.MetaPage, 0, len(pages))
	for _, page := range pages {
		result = append(result, page.MetaPage)
	}
	return result, nil
}

type debugTokenResponse struct {
	Data struct {
		IsValid             bool     `json:"is_valid"`
		ExpiresAt           int64    `json:"expires_at"`
		DataAccessExpiresAt int64    `json:"data_access_expires_at"`
		Scopes              []string `json:"scopes"`
	} `json:"data"`
}

func (g *GraphConnector) debugToken(ctx context.Context, token string) (*debugTokenResponse, error) {
	query := url.Values{"input_token": {token}, "access_token": {g.cfg.AppID + "|" + g.cfg.AppSecret}}
	var result debugTokenResponse
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/debug_token?"+query.Encode(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (g *GraphConnector) isPageSubscribed(ctx context.Context, page graphPage) (bool, error) {
	if page.Category == "WhatsApp" {
		// WhatsApp webhooks are configured at the App level, not per phone number.
		// If it's in the list, it's virtually connected or we can just say true if we have the token.
		return page.Connected, nil
	}
	query := url.Values{"access_token": {page.AccessToken}}
	var response struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(page.ID)+"/subscribed_apps?"+query.Encode(), &response); err != nil {
		return false, err
	}
	for _, app := range response.Data {
		if app.ID == g.cfg.AppID {
			return true, nil
		}
	}
	return false, nil
}

func (g *GraphConnector) ConnectPage(ctx context.Context, _ string, accountID, pageID string) (*domain.MetaPage, error) {
	g.mu.RLock()
	session := g.sessions[accountID]
	var page graphPage
	var ok bool
	if session != nil {
		page, ok = session.Pages[pageID]
	}
	g.mu.RUnlock()
	if !ok {
		return nil, errors.New("meta page not found")
	}

	if page.Category != "WhatsApp" {
		form := url.Values{"access_token": {page.AccessToken}}
		if fields := strings.TrimSpace(g.cfg.WebhookFields); fields != "" {
			form.Set("subscribed_fields", fields)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(pageID)+"/subscribed_apps", strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := g.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		var envelope struct {
			Success bool        `json:"success"`
			Error   *graphError `json:"error"`
		}
		_ = json.Unmarshal(body, &envelope)
		if resp.StatusCode >= 400 || envelope.Error != nil {
			return nil, graphResponseError(resp.StatusCode, envelope.Error)
		}
	}

	g.mu.Lock()
	page.Connected = true
	if page.Category != "WhatsApp" {
		page.WebhookStatus = "subscribed"
	} else {
		page.WebhookStatus = "app_level"
	}
	session.Pages[pageID] = page
	_ = g.persistStateLocked()
	g.mu.Unlock()
	
	ret := page.MetaPage
	return &ret, nil
}

func (g *GraphConnector) DisconnectPage(ctx context.Context, _ string, accountID, pageID string) (*domain.MetaPage, error) {
	g.mu.RLock()
	session := g.sessions[accountID]
	var page graphPage
	var ok bool
	if session != nil {
		page, ok = session.Pages[pageID]
	}
	g.mu.RUnlock()
	if !ok {
		return nil, errors.New("meta page not found")
	}

	if page.Category != "WhatsApp" {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(pageID)+"/subscribed_apps?access_token="+url.QueryEscape(page.AccessToken), nil)
		if err != nil {
			return nil, err
		}
		resp, err := g.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		var envelope struct {
			Success bool        `json:"success"`
			Error   *graphError `json:"error"`
		}
		_ = json.Unmarshal(body, &envelope)
		if resp.StatusCode >= 400 || envelope.Error != nil {
			return nil, graphResponseError(resp.StatusCode, envelope.Error)
		}
	}

	g.mu.Lock()
	page.Connected = false
	page.WebhookStatus = ""
	session.Pages[pageID] = page
	_ = g.persistStateLocked()
	g.mu.Unlock()
	
	ret := page.MetaPage
	return &ret, nil
}

func (g *GraphConnector) ListPosts(ctx context.Context, _ string, accountID, pageID string) ([]domain.MetaPost, error) {
	g.mu.RLock()
	session := g.sessions[accountID]
	var page graphPage
	var ok bool
	if session != nil {
		page, ok = session.Pages[pageID]
	}
	g.mu.RUnlock()
	if !ok {
		return nil, errors.New("meta page not found")
	}
	query := url.Values{"fields": {"id,message,created_time,permalink_url,full_picture"}, "access_token": {page.AccessToken}, "limit": {"50"}}
	var response struct {
		Data []struct {
			ID           string `json:"id"`
			Message      string `json:"message"`
			CreatedTime  string `json:"created_time"`
			PermalinkURL string `json:"permalink_url"`
			FullPicture  string `json:"full_picture"`
		} `json:"data"`
	}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(pageID)+"/published_posts?"+query.Encode(), &response); err != nil {
		return nil, err
	}
	posts := make([]domain.MetaPost, 0, len(response.Data))
	for _, item := range response.Data {
		posts = append(posts, domain.MetaPost{ID: item.ID, Message: item.Message, CreatedTime: item.CreatedTime, PermalinkURL: item.PermalinkURL, PictureURL: item.FullPicture})
	}
	return posts, nil
}

func (g *GraphConnector) userToken(accountID string) (string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	session := g.sessions[accountID]
	if session == nil || session.UserToken == "" {
		return "", errors.New("facebook connection not found")
	}
	return session.UserToken, nil
}

func (g *GraphConnector) ListAdAccounts(ctx context.Context, _ string, accountID string) ([]domain.MetaAdAccount, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		return nil, err
	}
	query := url.Values{"fields": {"id,name,account_status,business{id,name}"}, "limit": {"100"}, "access_token": {token}}
	var response struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			AccountStatus int    `json:"account_status"`
			Business      *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"business"`
		} `json:"data"`
	}
	if err = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me/adaccounts?"+query.Encode(), &response); err != nil {
		return nil, err
	}
	items := make([]domain.MetaAdAccount, 0, len(response.Data))
	for _, item := range response.Data {
		accountType, businessID, businessName := "personal", "", ""
		if item.Business != nil && item.Business.ID != "" {
			accountType, businessID, businessName = "business", item.Business.ID, item.Business.Name
		}
		items = append(items, domain.MetaAdAccount{ID: item.ID, Name: item.Name, AccountStatus: item.AccountStatus, AccountType: accountType, BusinessID: businessID, BusinessName: businessName})
	}
	return items, nil
}

func (g *GraphConnector) ListCampaigns(ctx context.Context, _ string, accountID, adAccountID string) ([]domain.MetaCampaign, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(adAccountID, "act_") {
		adAccountID = "act_" + adAccountID
	}
	query := url.Values{"fields": {"id,name,status,effective_status,objective,adsets.limit(0).summary(true),ads.limit(0).summary(true)"}, "limit": {"100"}, "access_token": {token}}
	var response struct {
		Data []struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			Status          string `json:"status"`
			EffectiveStatus string `json:"effective_status"`
			Objective       string `json:"objective"`
			AdSets          struct {
				Summary struct {
					TotalCount int `json:"total_count"`
				} `json:"summary"`
			} `json:"adsets"`
			Ads struct {
				Summary struct {
					TotalCount int `json:"total_count"`
				} `json:"summary"`
			} `json:"ads"`
		} `json:"data"`
	}
	if err = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(adAccountID)+"/campaigns?"+query.Encode(), &response); err != nil {
		return nil, err
	}
	items := make([]domain.MetaCampaign, 0, len(response.Data))
	for _, item := range response.Data {
		items = append(items, domain.MetaCampaign{ID: item.ID, Name: item.Name, Status: item.Status, EffectiveStatus: item.EffectiveStatus, Objective: item.Objective, AdSetCount: item.AdSets.Summary.TotalCount, AdCount: item.Ads.Summary.TotalCount})
	}
	return items, nil
}

func (g *GraphConnector) ListAds(ctx context.Context, _ string, accountID, campaignID string) ([]domain.MetaAd, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		return nil, err
	}
	query := url.Values{"fields": {"id,name,status,effective_status"}, "limit": {"200"}, "access_token": {token}}
	var response struct {
		Data []struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			Status          string `json:"status"`
			EffectiveStatus string `json:"effective_status"`
		} `json:"data"`
	}
	if err = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(campaignID)+"/ads?"+query.Encode(), &response); err != nil {
		return nil, err
	}
	items := make([]domain.MetaAd, 0, len(response.Data))
	for _, item := range response.Data {
		items = append(items, domain.MetaAd{ID: item.ID, Name: item.Name, Status: item.Status, EffectiveStatus: item.EffectiveStatus})
	}
	return items, nil
}

func (g *GraphConnector) ListProductBindings(_ context.Context, _, accountID string) ([]domain.ProductBinding, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := []domain.ProductBinding{}
	session := g.sessions[accountID]
	if session == nil {
		return result, nil
	}
	for _, item := range g.bindings {
		if _, ok := session.Pages[item.PageID]; ok {
			result = append(result, item)
		}
	}
	return result, nil
}

func (g *GraphConnector) SaveProductBinding(_ context.Context, _, accountID string, binding domain.ProductBinding) (*domain.ProductBinding, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	session := g.sessions[accountID]
	if session == nil {
		return nil, errors.New("facebook connection not found")
	}
	if _, ok := session.Pages[binding.PageID]; !ok {
		return nil, errors.New("meta page not found")
	}
	flow, flowFound := g.replyFlows[accountID][binding.FlowID]
	if !flowFound {
		return nil, errors.New("reply flow not found")
	}
	if binding.ProductID != "" {
		product, productFound := g.products[accountID][binding.ProductID]
		if !productFound {
			return nil, errors.New("product not found")
		}
		binding.ProductName, binding.Price, binding.Description = product.Name, product.Price, product.Description
	}
	binding.Messages = append([]domain.ReplyStep(nil), flow.Messages...)
	binding.Keywords = append([]string(nil), flow.Keywords...)
	g.bindings[bindingKey(binding.PageID, binding.SourceType, binding.SourceID)] = binding
	if err := g.persistStateLocked(); err != nil {
		return nil, fmt.Errorf("persist automation: %w", err)
	}
	copy := binding
	return &copy, nil
}

func (g *GraphConnector) SaveProductBindings(_ context.Context, _, accountID string, bindings []domain.ProductBinding) ([]domain.ProductBinding, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	session := g.sessions[accountID]
	if session == nil {
		return nil, errors.New("facebook connection not found")
	}
	prepared := make([]domain.ProductBinding, len(bindings))
	for index, binding := range bindings {
		if _, ok := session.Pages[binding.PageID]; !ok {
			return nil, fmt.Errorf("page %s not found", binding.PageID)
		}
		product, ok := g.products[accountID][binding.ProductID]
		if !ok {
			return nil, errors.New("product not found")
		}
		flow, ok := g.replyFlows[accountID][binding.FlowID]
		if !ok {
			return nil, errors.New("reply flow not found")
		}
		binding.ProductName, binding.Price, binding.Description = product.Name, product.Price, product.Description
		binding.Messages = append([]domain.ReplyStep(nil), flow.Messages...)
		binding.Keywords = append([]string(nil), flow.Keywords...)
		prepared[index] = binding
	}
	for _, binding := range prepared {
		g.bindings[bindingKey(binding.PageID, binding.SourceType, binding.SourceID)] = binding
	}
	if err := g.persistStateLocked(); err != nil {
		return nil, fmt.Errorf("persist automations: %w", err)
	}
	return prepared, nil
}


func (g *GraphConnector) ListReplyFlows(_ context.Context, _, accountID string) ([]domain.ReplyFlow, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := []domain.ReplyFlow{}
	for _, item := range g.replyFlows[accountID] {
		result = append(result, item)
	}
	return result, nil
}

func (g *GraphConnector) SaveReplyFlow(_ context.Context, _, accountID string, flow domain.ReplyFlow) (*domain.ReplyFlow, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if flow.ID == "" {
		value := make([]byte, 12)
		if _, err := rand.Read(value); err != nil {
			return nil, err
		}
		flow.ID = base64.RawURLEncoding.EncodeToString(value)
	}
	if g.replyFlows[accountID] == nil {
		g.replyFlows[accountID] = map[string]domain.ReplyFlow{}
	}
	g.replyFlows[accountID][flow.ID] = flow
	if err := g.persistStateLocked(); err != nil {
		return nil, fmt.Errorf("persist reply flow: %w", err)
	}
	copy := flow
	return &copy, nil
}

func (g *GraphConnector) sendMessage(ctx context.Context, page graphPage, recipientID, text string) error {
	if page.Category == "WhatsApp" {
		return g.sendWhatsAppMessage(ctx, page.ID, page.AccessToken, recipientID, text)
	}
	payload, err := json.Marshal(map[string]any{"recipient": map[string]string{"id": recipientID}, "messaging_type": "RESPONSE", "message": map[string]string{"text": text}})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(page.ID)+"/messages?access_token="+url.QueryEscape(page.AccessToken), strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var envelope struct {
		Error *graphError `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	if resp.StatusCode >= 400 || envelope.Error != nil {
		return graphResponseError(resp.StatusCode, envelope.Error)
	}
	return nil
}

func (g *GraphConnector) sendMedia(ctx context.Context, page graphPage, recipientID, mediaType, mediaURL string) error {
	if page.Category == "WhatsApp" {
		return g.sendWhatsAppMedia(ctx, page.ID, page.AccessToken, recipientID, mediaType, mediaURL)
	}
	payload, err := json.Marshal(map[string]any{"recipient": map[string]string{"id": recipientID}, "messaging_type": "RESPONSE", "message": map[string]any{"attachment": map[string]any{"type": mediaType, "payload": map[string]any{"url": mediaURL, "is_reusable": true}}}})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(page.ID)+"/messages?access_token="+url.QueryEscape(page.AccessToken), strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var envelope struct {
		Error *graphError `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	if resp.StatusCode >= 400 || envelope.Error != nil {
		return graphResponseError(resp.StatusCode, envelope.Error)
	}
	return nil
}

func (g *GraphConnector) sendPrivateReply(ctx context.Context, page graphPage, commentID, text string) error {
	form := url.Values{"message": {text}, "access_token": {page.AccessToken}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(commentID)+"/private_replies", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var envelope struct {
		Error *graphError `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	if resp.StatusCode >= 400 || envelope.Error != nil {
		return graphResponseError(resp.StatusCode, envelope.Error)
	}
	return nil
}

func (g *GraphConnector) replyToComment(ctx context.Context, page graphPage, commentID, text string) error {
	form := url.Values{"message": {text}, "access_token": {page.AccessToken}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(commentID)+"/comments", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var envelope struct {
		Error *graphError `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	if resp.StatusCode >= 400 || envelope.Error != nil {
		return graphResponseError(resp.StatusCode, envelope.Error)
	}
	return nil
}

func (g *GraphConnector) fetchPages(ctx context.Context, token string) (map[string]graphPage, error) {
	query := url.Values{"fields": {"id,name,category,picture{url},access_token"}, "access_token": {token}, "limit": {"100"}}
	var response struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Category    string `json:"category"`
			AccessToken string `json:"access_token"`
			Picture     struct {
				Data struct {
					URL string `json:"url"`
				} `json:"data"`
			} `json:"picture"`
		} `json:"data"`
		Error *graphError `json:"error"`
	}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me/accounts?"+query.Encode(), &response); err != nil {
		return nil, err
	}
	pages := make(map[string]graphPage)
	for _, item := range response.Data {
		pages[item.ID] = graphPage{MetaPage: domain.MetaPage{ID: item.ID, Name: item.Name, Category: item.Category, PictureURL: item.Picture.Data.URL, TokenReady: item.AccessToken != ""}, AccessToken: item.AccessToken}
	}

	// Also fetch WhatsApp Business Accounts (WABAs)
	var bizResponse struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me/businesses?access_token="+token, &bizResponse); err == nil {
		for _, biz := range bizResponse.Data {
			var wabaResponse struct {
				Data []struct {
					ID                string `json:"id"`
					Name              string `json:"name"`
					ProfilePictureUrl string `json:"profile_picture_url"`
				} `json:"data"`
			}
			if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+biz.ID+"/owned_whatsapp_business_accounts?fields=id,name,profile_picture_url&access_token="+token, &wabaResponse); err == nil {
				for _, waba := range wabaResponse.Data {
					var phoneResponse struct {
						Data []struct {
							ID                 string `json:"id"`
							DisplayPhoneNumber string `json:"display_phone_number"`
							VerifiedName       string `json:"verified_name"`
						} `json:"data"`
					}
					if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+waba.ID+"/phone_numbers?fields=id,display_phone_number,verified_name&access_token="+token, &phoneResponse); err == nil {
						for _, phone := range phoneResponse.Data {
							name := phone.VerifiedName
							if name == "" {
								name = waba.Name
							}
							if name == "" {
								name = phone.DisplayPhoneNumber
							}
							pages[phone.ID] = graphPage{
								MetaPage: domain.MetaPage{
									ID:         phone.ID,
									Name:       name,
									Category:   "WhatsApp",
									PictureURL: waba.ProfilePictureUrl,
									TokenReady: true,
								},
								AccessToken: token,
							}
						}
					}
				}
			}
		}
	}

	return pages, nil
}

type graphError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

func (g *GraphConnector) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var envelope struct {
		Error *graphError `json:"error"`
	}
	if err = json.Unmarshal(body, &envelope); err != nil {
		return err
	}
	if resp.StatusCode >= 400 || envelope.Error != nil {
		return graphResponseError(resp.StatusCode, envelope.Error)
	}
	return json.Unmarshal(body, target)
}
func graphResponseError(status int, value *graphError) error {
	if value == nil {
		return fmt.Errorf("meta graph request failed with status %d", status)
	}
	return fmt.Errorf("meta graph error %d: %s", value.Code, value.Message)
}
