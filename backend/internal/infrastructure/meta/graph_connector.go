package meta

import (
	"crypto/aes"
	"crypto/cipher"

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
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/usecase"
)

type GraphConfig struct {
	AppID, AppSecret, RedirectURI, Version, WebhookFields, WebhookVerifyToken, StateFile, EncryptionKey string
	WhatsAppBusinessAccountIDs, WhatsAppConfigID                                                        string
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
	WhatsAppDiagnostics                 domain.MetaWhatsAppDiagnostics
	Pages                               map[string]graphPage
}

type graphPage struct {
	domain.MetaPage
	AccessToken string
}

type GraphConnector struct {
	db *gorm.DB

	chatStream    *usecase.ChatStream
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

func NewGraphConnector(cfg GraphConfig, db *gorm.DB) *GraphConnector {
	g := &GraphConnector{db: db, cfg: cfg, client: &http.Client{Timeout: 15 * time.Second}, states: make(map[string]oauthState), sessions: make(map[string]*graphSession), recentEvents: make(map[string]time.Time), bindings: make(map[string]domain.ProductBinding), products: make(map[string]map[string]domain.Product), replyFlows: make(map[string]map[string]domain.ReplyFlow), firstReplies: make(map[string]time.Time), conversations: make(map[string]string), deliverySteps: make(map[string]bool), inFlight: make(map[string]bool)}
	if err := g.loadState(); err != nil {
		slog.Warn("unable to load Meta state", "error", err)
	}
	return g
}

type persistentGraphState struct {
	// Sessions is retained only so older state files can be read and scrubbed.
	// OAuth and Page access tokens belong exclusively in the encrypted database
	// columns; they must never be written to the state file.
	Sessions   map[string]*graphSession               `json:"sessions,omitempty"`
	Bindings   map[string]domain.ProductBinding       `json:"bindings"`
	Products   map[string]map[string]domain.Product   `json:"products"`
	ReplyFlows map[string]map[string]domain.ReplyFlow `json:"replyFlows"`
}

func (g *GraphConnector) encryptToken(token string) string {
	if token == "" {
		return ""
	}
	// Fail closed: a configuration or cryptographic failure must never cause a
	// live credential to be written as plaintext.
	if g.cfg.EncryptionKey == "" {
		return ""
	}
	key := sha256.Sum256([]byte(g.cfg.EncryptionKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return ""
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)
	return "enc:" + hex.EncodeToString(ciphertext)
}

func (g *GraphConnector) decryptToken(cipherHex string) string {
	if !strings.HasPrefix(cipherHex, "enc:") || g.cfg.EncryptionKey == "" {
		return cipherHex
	}
	cipherHex = strings.TrimPrefix(cipherHex, "enc:")
	key := sha256.Sum256([]byte(g.cfg.EncryptionKey))
	ciphertext, err := hex.DecodeString(cipherHex)
	if err != nil {
		return ""
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return ""
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ""
	}
	return string(plaintext)
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
	if state.Bindings != nil {
		g.bindings = state.Bindings
	}
	if state.Products != nil {
		g.products = state.Products
	}
	if state.ReplyFlows != nil {
		g.replyFlows = state.ReplyFlows
	}
	// Previous versions persisted live OAuth credentials in this file. Ignore
	// them and immediately rewrite the file without the legacy sessions field.
	// Database-backed sessions are restored lazily by userToken/findPage.
	if state.Sessions != nil {
		state.Sessions = nil
		return g.writePersistentState(state)
	}
	return nil
}

// persistStateLocked atomically saves durable configuration and OAuth tokens.
// The caller must hold g.mu. The file is user-readable only.
func (g *GraphConnector) persistStateLocked() error {
	if strings.TrimSpace(g.cfg.StateFile) == "" {
		return nil
	}
	state := persistentGraphState{Bindings: g.bindings, Products: g.products, ReplyFlows: g.replyFlows}
	return g.writePersistentState(state)
}

func (g *GraphConnector) writePersistentState(state persistentGraphState) error {
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
					err = g.SendMessage(ctx, rule.AccountID, page.ID, message.Sender.ID, step.Content)
				} else {
					err = g.SendMedia(ctx, rule.AccountID, page.ID, message.Sender.ID, step.Type, step.Content)
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
	g.mu.RLock()
	for accountID, session := range g.sessions {
		if page, ok := session.Pages[pageID]; ok {
			g.mu.RUnlock()
			return accountID, page, true
		}
	}
	g.mu.RUnlock()

	if g.db != nil {
		var whatsApp database.MetaWhatsAppConnectionModel
		if err := g.db.First(&whatsApp, "phone_number_id = ?", pageID).Error; err == nil {
			decrypted := g.decryptToken(whatsApp.AccessToken)
			page := graphPage{MetaPage: domain.MetaPage{
				ID:            whatsApp.PhoneNumberID,
				Name:          firstNonEmpty(whatsApp.VerifiedName, whatsApp.DisplayPhoneNumber, "WhatsApp"),
				Category:      "WhatsApp",
				PictureURL:    whatsApp.PictureURL,
				PhoneNumber:   whatsApp.DisplayPhoneNumber,
				Connected:     whatsApp.IsConnected,
				TokenReady:    decrypted != "",
				WebhookStatus: map[bool]string{true: "subscribed", false: "disconnected"}[whatsApp.IsConnected],
			}, AccessToken: decrypted}
			g.cachePage(whatsApp.AccountID, page)
			return whatsApp.AccountID, page, true
		}
		var model database.MetaPageTokenModel
		if err := g.db.First(&model, "page_id = ?", pageID).Error; err == nil {
			decrypted := g.decryptToken(model.AccessToken)
			page := graphPage{
				MetaPage: domain.MetaPage{
					ID:         model.PageID,
					Name:       model.Name,
					Category:   model.Category,
					PictureURL: model.PictureURL,
					Connected:  model.IsConnected,
					TokenReady: decrypted != "",
				},
				AccessToken: decrypted,
			}
			g.mu.Lock()
			if g.sessions[model.AccountID] == nil {
				g.sessions[model.AccountID] = &graphSession{Pages: make(map[string]graphPage)}
			}
			g.sessions[model.AccountID].Pages[model.PageID] = page
			g.mu.Unlock()
			return model.AccountID, page, true
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
	// Copy request-derived strings before retaining them beyond the request.
	// Fiber/fasthttp may reuse its request buffer as soon as the handler exits.
	g.states[state] = oauthState{
		UserID:    strings.Clone(userID),
		AccountID: strings.Clone(accountID),
		Scopes:    append([]string(nil), scopes...),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
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
		return nil, fmt.Errorf("exchange authorization code: %w", err)
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
	pages, whatsAppDiagnostics, err := g.fetchPages(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("load Facebook Pages: %w", err)
	}
	applyWhatsAppPermissions(&whatsAppDiagnostics, debug.Data.Scopes)
	encryptedUserToken := g.encryptToken(token.AccessToken)
	if encryptedUserToken == "" {
		return nil, errors.New("unable to encrypt Meta access token")
	}
	// Persist the freshly authorized credential before completing the callback.
	// Previously this error was ignored, so the UI could report a successful
	// reconnect while the next request still loaded the old expired token.
	if pending.AccountID != "" && g.db != nil {
		if err := g.db.Save(&database.MetaConnectionModel{
			AccountID:           pending.AccountID,
			UserToken:           encryptedUserToken,
			TokenExpiresAt:      debug.Data.ExpiresAt,
			DataAccessExpiresAt: debug.Data.DataAccessExpiresAt,
			GrantedPermissions:  strings.Join(debug.Data.Scopes, ","),
		}).Error; err != nil {
			return nil, fmt.Errorf("save encrypted Meta authorization: %w", err)
		}
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
		g.sessions[pending.AccountID] = &graphSession{UserToken: token.AccessToken, TokenExpiresAt: debug.Data.ExpiresAt, DataAccessExpiresAt: debug.Data.DataAccessExpiresAt, GrantedPermissions: append([]string(nil), debug.Data.Scopes...), WhatsAppDiagnostics: whatsAppDiagnostics, Pages: pages}
		_ = g.persistStateLocked()
	}
	g.mu.Unlock()
	return &domain.MetaOAuthResult{UserID: pending.UserID, AccountID: pending.AccountID, AccessToken: token.AccessToken, FacebookID: profile.ID, Name: profile.Name, Email: profile.Email}, nil
}

func (g *GraphConnector) ListPages(ctx context.Context, _ string, accountID string) ([]domain.MetaPage, error) {
	persistedWhatsApp, persistedErr := g.persistedWhatsAppPages(accountID)
	if persistedErr != nil {
		return nil, fmt.Errorf("load saved WhatsApp connections: %w", persistedErr)
	}
	// Always resolve the token through userToken. In production the process can
	// restart while the encrypted OAuth token remains in PostgreSQL. Looking at
	// the in-memory session only made every Page appear to disappear after a
	// deployment until the user completed OAuth again.
	token, err := g.userToken(ctx, accountID)
	if err != nil {
		if len(persistedWhatsApp) > 0 {
			result := make([]domain.MetaPage, 0, len(persistedWhatsApp))
			for _, page := range persistedWhatsApp {
				g.cachePage(accountID, page)
				result = append(result, page.MetaPage)
			}
			return result, nil
		}
		return nil, fmt.Errorf("facebook connection not found or invalid: %w", err)
	}
	g.mu.RLock()
	session := g.sessions[accountID]
	var tokenExpiresAt, dataAccessExpiresAt int64
	var grantedPermissions []string
	if session != nil {
		tokenExpiresAt = session.TokenExpiresAt
		dataAccessExpiresAt = session.DataAccessExpiresAt
		grantedPermissions = append([]string(nil), session.GrantedPermissions...)
	}
	g.mu.RUnlock()

	pages, whatsAppDiagnostics, err := g.fetchPages(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("facebook api error: %w", err)
	}
	applyWhatsAppPermissions(&whatsAppDiagnostics, grantedPermissions)
	for id, page := range persistedWhatsApp {
		pages[id] = page
	}
	if len(persistedWhatsApp) > 0 {
		whatsAppDiagnostics.State = "ready"
		whatsAppDiagnostics.Message = "WhatsApp is connected through Embedded Signup."
		whatsAppDiagnostics.PhoneNumberCount = len(persistedWhatsApp)
	}
	for id, page := range pages {
		page.TokenExpiresAt = tokenExpiresAt
		page.DataAccessExpiresAt = dataAccessExpiresAt
		page.GrantedPermissions = append([]string(nil), grantedPermissions...)
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
	if g.sessions[accountID] == nil {
		g.sessions[accountID] = &graphSession{UserToken: token, Pages: make(map[string]graphPage)}
	}
	session = g.sessions[accountID]
	session.WhatsAppDiagnostics = whatsAppDiagnostics
	if session.Pages == nil {
		session.Pages = make(map[string]graphPage)
	}
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

func (g *GraphConnector) WhatsAppDiagnostics(ctx context.Context, accountID string) (*domain.MetaWhatsAppDiagnostics, error) {
	if pages, err := g.persistedWhatsAppPages(accountID); err == nil && len(pages) > 0 {
		return &domain.MetaWhatsAppDiagnostics{
			State: "ready", Message: "WhatsApp is connected through Embedded Signup.",
			RequiredPermissions:          []string{"whatsapp_business_management", "whatsapp_business_messaging"},
			WhatsAppBusinessAccountCount: 1, PhoneNumberCount: len(pages),
		}, nil
	}
	if _, err := g.userToken(ctx, accountID); err != nil {
		return &domain.MetaWhatsAppDiagnostics{State: "not_connected", Message: "Connect WhatsApp with Embedded Signup."}, nil
	}
	g.mu.RLock()
	session := g.sessions[accountID]
	if session == nil {
		g.mu.RUnlock()
		return nil, errors.New("meta session not loaded")
	}
	diagnostics := session.WhatsAppDiagnostics
	diagnostics.RequiredPermissions = append([]string(nil), diagnostics.RequiredPermissions...)
	diagnostics.GrantedPermissions = append([]string(nil), diagnostics.GrantedPermissions...)
	diagnostics.MissingPermissions = append([]string(nil), diagnostics.MissingPermissions...)
	diagnostics.Issues = append([]string(nil), diagnostics.Issues...)
	g.mu.RUnlock()
	return &diagnostics, nil
}

func (g *GraphConnector) PagePicture(ctx context.Context, accountID, pageID string) (*domain.MetaPagePicture, error) {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return nil, usecase.ErrPageNotFound
	}

	// Resolve the Page from this workspace before fetching media. This prevents
	// the endpoint from being used as an unauthorised Graph/CDN image proxy.
	g.mu.RLock()
	session := g.sessions[accountID]
	page, found := graphPage{}, false
	if session != nil {
		page, found = session.Pages[pageID]
	}
	g.mu.RUnlock()
	if !found {
		foundAccountID, savedPage, saved := g.findPage(pageID)
		if saved && foundAccountID == accountID {
			page, found = savedPage, true
		}
	}
	if !found {
		token, err := g.userToken(ctx, accountID)
		if err != nil {
			return nil, domain.ErrMetaReconnectRequired
		}
		pages, _, err := g.fetchPages(ctx, token)
		if err != nil {
			return nil, err
		}
		page, found = pages[pageID]
		if !found {
			return nil, usecase.ErrPageNotFound
		}
	}

	endpoint := strings.TrimSpace(page.PictureURL)
	if !strings.EqualFold(page.Category, "WhatsApp") {
		query := url.Values{
			"type":         {"large"},
			"redirect":     {"true"},
			"access_token": {page.AccessToken},
		}
		endpoint = "https://graph.facebook.com/" + g.cfg.Version + "/" + url.PathEscape(page.ID) + "/picture?" + query.Encode()
	}
	if endpoint == "" {
		return nil, usecase.ErrPageNotFound
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/*")
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Meta Page picture returned status %d", resp.StatusCode)
	}
	contentType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return nil, errors.New("Meta Page picture returned invalid content type")
	}
	const maxPictureBytes = 5 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPictureBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maxPictureBytes {
		return nil, errors.New("Meta Page picture is empty or too large")
	}
	return &domain.MetaPagePicture{ContentType: contentType, Data: data}, nil
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
	encryptedPageToken := g.encryptToken(page.AccessToken)
	if page.AccessToken != "" && encryptedPageToken == "" {
		return nil, errors.New("unable to encrypt Meta Page access token")
	}

	g.mu.Lock()
	page.Connected = true
	if page.Category != "WhatsApp" {
		page.WebhookStatus = "subscribed"
	} else {
		page.WebhookStatus = "app_level"
	}
	session.Pages[pageID] = page
	if g.db != nil && page.Category == "WhatsApp" {
		g.db.Model(&database.MetaWhatsAppConnectionModel{}).Where("phone_number_id = ? AND account_id = ?", pageID, accountID).Update("is_connected", true)
	} else if g.db != nil {
		g.db.Save(&database.MetaPageTokenModel{
			PageID:      pageID,
			AccountID:   accountID,
			Name:        page.Name,
			Category:    page.Category,
			PictureURL:  page.PictureURL,
			AccessToken: encryptedPageToken,
			IsConnected: page.Connected,
		})
	}
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
	encryptedPageToken := g.encryptToken(page.AccessToken)
	if page.AccessToken != "" && encryptedPageToken == "" {
		return nil, errors.New("unable to encrypt Meta Page access token")
	}

	g.mu.Lock()
	page.Connected = false
	page.WebhookStatus = ""
	session.Pages[pageID] = page
	if g.db != nil && page.Category == "WhatsApp" {
		g.db.Model(&database.MetaWhatsAppConnectionModel{}).Where("phone_number_id = ? AND account_id = ?", pageID, accountID).Update("is_connected", false)
	} else if g.db != nil {
		g.db.Save(&database.MetaPageTokenModel{
			PageID:      pageID,
			AccountID:   accountID,
			Name:        page.Name,
			Category:    page.Category,
			PictureURL:  page.PictureURL,
			AccessToken: encryptedPageToken,
			IsConnected: page.Connected,
		})
	}
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

func (g *GraphConnector) userToken(ctx context.Context, accountID string) (string, error) {
	g.mu.RLock()
	session := g.sessions[accountID]
	var token string
	var tokenExpiresAt, dataAccessExpiresAt int64
	if session != nil {
		token = session.UserToken
		tokenExpiresAt = session.TokenExpiresAt
		dataAccessExpiresAt = session.DataAccessExpiresAt
	}
	g.mu.RUnlock()

	if token == "" && g.db != nil {
		var model database.MetaConnectionModel
		if err := g.db.First(&model, "account_id = ?", accountID).Error; err == nil {
			token = g.decryptToken(model.UserToken)
			tokenExpiresAt = model.TokenExpiresAt
			dataAccessExpiresAt = model.DataAccessExpiresAt
			permissions := splitPermissions(model.GrantedPermissions)
			g.mu.Lock()
			if g.sessions[accountID] == nil {
				g.sessions[accountID] = &graphSession{Pages: make(map[string]graphPage)}
			}
			g.sessions[accountID].UserToken = token
			g.sessions[accountID].TokenExpiresAt = tokenExpiresAt
			g.sessions[accountID].DataAccessExpiresAt = dataAccessExpiresAt
			g.sessions[accountID].GrantedPermissions = permissions
			g.mu.Unlock()
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("load encrypted Meta token: %w", err)
		}
	}
	if token == "" {
		return "", domain.ErrMetaReconnectRequired
	}

	now := time.Now().Unix()
	if dataAccessExpiresAt > 0 && now >= dataAccessExpiresAt {
		return "", domain.ErrMetaReconnectRequired
	}
	if tokenExpiresAt > 0 && now >= tokenExpiresAt {
		return "", domain.ErrMetaReconnectRequired
	}

	// Older rows did not persist expiry metadata. Debug once, then store the
	// result so subsequent requests do not need an extra Graph API call.
	if tokenExpiresAt == 0 && dataAccessExpiresAt == 0 {
		debug, err := g.debugToken(ctx, token)
		if err != nil || !debug.Data.IsValid {
			return "", domain.ErrMetaReconnectRequired
		}
		g.storeUserToken(accountID, token, debug)
		tokenExpiresAt = debug.Data.ExpiresAt
		dataAccessExpiresAt = debug.Data.DataAccessExpiresAt
		if dataAccessExpiresAt > 0 && now >= dataAccessExpiresAt {
			return "", domain.ErrMetaReconnectRequired
		}
	}

	// Meta does not issue a conventional refresh token. While the current user
	// token is still valid, opportunistically exchange it before it reaches its
	// expiry window. If Meta refuses the exchange, keep using the still-valid
	// token and ask the user to reconnect only when it actually expires.
	const refreshWindow = int64((7 * 24 * time.Hour) / time.Second)
	if tokenExpiresAt > 0 && tokenExpiresAt-now <= refreshWindow {
		refreshedToken, debug, err := g.exchangeLongLivedUserToken(ctx, token)
		if err != nil {
			slog.Warn("unable to renew Meta user token; current token remains active", "account_id", accountID, "error", err)
			return token, nil
		}
		token = refreshedToken
		g.storeUserToken(accountID, token, debug)
	}
	return token, nil
}

func splitPermissions(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return r == ',' })
}

func (g *GraphConnector) exchangeLongLivedUserToken(ctx context.Context, current string) (string, *debugTokenResponse, error) {
	query := url.Values{
		"grant_type":        {"fb_exchange_token"},
		"client_id":         {g.cfg.AppID},
		"client_secret":     {g.cfg.AppSecret},
		"fb_exchange_token": {current},
	}
	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/oauth/access_token?"+query.Encode(), &result); err != nil {
		return "", nil, err
	}
	if result.AccessToken == "" {
		return "", nil, errors.New("meta returned an empty renewed access token")
	}
	debug, err := g.debugToken(ctx, result.AccessToken)
	if err != nil {
		return "", nil, err
	}
	if !debug.Data.IsValid {
		return "", nil, errors.New("meta returned an invalid renewed access token")
	}
	return result.AccessToken, debug, nil
}

func (g *GraphConnector) storeUserToken(accountID, token string, debug *debugTokenResponse) {
	permissions := append([]string(nil), debug.Data.Scopes...)
	g.mu.Lock()
	if g.sessions[accountID] == nil {
		g.sessions[accountID] = &graphSession{Pages: make(map[string]graphPage)}
	}
	g.sessions[accountID].UserToken = token
	g.sessions[accountID].TokenExpiresAt = debug.Data.ExpiresAt
	g.sessions[accountID].DataAccessExpiresAt = debug.Data.DataAccessExpiresAt
	g.sessions[accountID].GrantedPermissions = permissions
	g.mu.Unlock()

	if g.db != nil {
		encryptedToken := g.encryptToken(token)
		if encryptedToken == "" {
			slog.Error("unable to encrypt renewed Meta token", "account_id", accountID)
			return
		}
		updates := map[string]any{
			"user_token":             encryptedToken,
			"token_expires_at":       debug.Data.ExpiresAt,
			"data_access_expires_at": debug.Data.DataAccessExpiresAt,
			"granted_permissions":    strings.Join(permissions, ","),
		}
		if err := g.db.Model(&database.MetaConnectionModel{}).Where("account_id = ?", accountID).Updates(updates).Error; err != nil {
			slog.Warn("unable to persist renewed Meta token", "account_id", accountID, "error", err)
		}
	}
}

func (g *GraphConnector) ListAdAccounts(ctx context.Context, _ string, accountID string) ([]domain.MetaAdAccount, error) {
	token, err := g.userToken(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("facebook connection not found or invalid: %w", err)
	}
	type graphBusiness struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type graphAdAccount struct {
		ID            string         `json:"id"`
		Name          string         `json:"name"`
		AccountStatus int            `json:"account_status"`
		Business      *graphBusiness `json:"business"`
	}
	query := url.Values{"fields": {"id,name,account_status,business{id,name}"}, "limit": {"100"}, "access_token": {token}}
	var response struct {
		Data []graphAdAccount `json:"data"`
	}
	if err = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me/adaccounts?"+query.Encode(), &response); err != nil {
		return nil, fmt.Errorf("facebook api error: %w", err)
	}

	// /me/adaccounts normally returns accounts explicitly assigned to the user,
	// but Business Portfolio owned/client accounts can be absent for some role
	// combinations. Read those edges too and merge them into one deduplicated
	// list so the UI works for both personal and business ad accounts.
	byID := make(map[string]domain.MetaAdAccount, len(response.Data))
	for _, item := range response.Data {
		accountType, businessID, businessName := "personal", "", ""
		if item.Business != nil && item.Business.ID != "" {
			accountType, businessID, businessName = "business", item.Business.ID, item.Business.Name
		}
		byID[item.ID] = domain.MetaAdAccount{ID: item.ID, Name: item.Name, AccountStatus: item.AccountStatus, AccountType: accountType, BusinessID: businessID, BusinessName: businessName}
	}

	var businesses struct {
		Data []graphBusiness `json:"data"`
	}
	businessQuery := url.Values{"fields": {"id,name"}, "limit": {"100"}, "access_token": {token}}
	if businessErr := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me/businesses?"+businessQuery.Encode(), &businesses); businessErr != nil {
		slog.Warn("unable to list Meta Business Portfolios", "error", businessErr)
	} else {
		for _, business := range businesses.Data {
			for _, edge := range []string{"owned_ad_accounts", "client_ad_accounts"} {
				edgeQuery := url.Values{"fields": {"id,name,account_status"}, "limit": {"100"}, "access_token": {token}}
				var edgeResponse struct {
					Data []graphAdAccount `json:"data"`
				}
				endpoint := "https://graph.facebook.com/" + g.cfg.Version + "/" + url.PathEscape(business.ID) + "/" + edge + "?" + edgeQuery.Encode()
				if edgeErr := g.getJSON(ctx, endpoint, &edgeResponse); edgeErr != nil {
					slog.Warn("unable to list Meta Business Portfolio ad accounts", "business_id", business.ID, "edge", edge, "error", edgeErr)
					continue
				}
				for _, item := range edgeResponse.Data {
					byID[item.ID] = domain.MetaAdAccount{ID: item.ID, Name: item.Name, AccountStatus: item.AccountStatus, AccountType: "business", BusinessID: business.ID, BusinessName: business.Name}
				}
			}
		}
	}

	items := make([]domain.MetaAdAccount, 0, len(byID))
	for _, item := range byID {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].AccountType != items[j].AccountType {
			return items[i].AccountType < items[j].AccountType
		}
		if items[i].BusinessName != items[j].BusinessName {
			return items[i].BusinessName < items[j].BusinessName
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (g *GraphConnector) ListCampaigns(ctx context.Context, _ string, accountID, adAccountID string) ([]domain.MetaCampaign, error) {
	token, err := g.userToken(ctx, accountID)
	if err != nil {
		return []domain.MetaCampaign{}, nil
	}
	if !strings.HasPrefix(adAccountID, "act_") {
		adAccountID = "act_" + adAccountID
	}
	query := url.Values{"fields": {"id,name,status,effective_status,objective,adsets.summary(1).limit(1),ads.summary(1).limit(1)"}, "limit": {"100"}, "access_token": {token}}
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
	token, err := g.userToken(ctx, accountID)
	if err != nil {
		return []domain.MetaAd{}, nil
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

func (g *GraphConnector) SendMessage(ctx context.Context, accountID, pageID, recipientID, text string) error {
	foundAccountID, page, found := g.findPage(pageID)
	if !found || foundAccountID != accountID {
		return errors.New("meta page not found or not connected")
	}

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
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("facebook send message failed: %s", string(body))
	}
	return nil
}

func (g *GraphConnector) SendMedia(ctx context.Context, accountID, pageID, recipientID, mediaType, mediaURL string) error {
	foundAccountID, page, found := g.findPage(pageID)
	if !found || foundAccountID != accountID {
		return errors.New("meta page not found")
	}
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

func (g *GraphConnector) fetchPages(ctx context.Context, token string) (map[string]graphPage, domain.MetaWhatsAppDiagnostics, error) {
	diagnostics := domain.MetaWhatsAppDiagnostics{
		RequiredPermissions: []string{"business_management", "whatsapp_business_management"},
		State:               "checking",
		Message:             "Checking WhatsApp Business access with Meta.",
	}
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
		return nil, diagnostics, err
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
	businessQuery := url.Values{"fields": {"id,name"}, "access_token": {token}, "limit": {"100"}}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me/businesses?"+businessQuery.Encode(), &bizResponse); err == nil {
		diagnostics.BusinessCount = len(bizResponse.Data)
		seenWABAs := make(map[string]bool)
		wabaErrors := 0
		phoneErrors := 0
		addWABA := func(businessID, wabaID, wabaName string) {
			wabaID = strings.TrimSpace(wabaID)
			if wabaID == "" || seenWABAs[wabaID] {
				return
			}
			seenWABAs[wabaID] = true
			diagnostics.WhatsAppBusinessAccountCount++
			var phoneResponse struct {
				Data []struct {
					ID                 string `json:"id"`
					DisplayPhoneNumber string `json:"display_phone_number"`
					VerifiedName       string `json:"verified_name"`
				} `json:"data"`
			}
			phoneQuery := url.Values{"fields": {"id,display_phone_number,verified_name"}, "access_token": {token}, "limit": {"100"}}
			if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(wabaID)+"/phone_numbers?"+phoneQuery.Encode(), &phoneResponse); err != nil {
				phoneErrors++
				diagnostics.Issues = append(diagnostics.Issues, diagnosticIssue("phone_numbers", err))
				slog.Warn("unable to discover WhatsApp phone numbers", "business_id", businessID, "waba_id", wabaID, "error", err)
				return
			}
			for _, phone := range phoneResponse.Data {
				diagnostics.PhoneNumberCount++
				name := phone.VerifiedName
				if name == "" {
					name = wabaName
				}
				if name == "" {
					name = phone.DisplayPhoneNumber
				}

				pictureURL := ""
				var profileResponse struct {
					Data []struct {
						ProfilePictureURL string `json:"profile_picture_url"`
					} `json:"data"`
				}
				profileQuery := url.Values{"fields": {"profile_picture_url"}, "access_token": {token}}
				if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(phone.ID)+"/whatsapp_business_profile?"+profileQuery.Encode(), &profileResponse); err != nil {
					slog.Warn("unable to load WhatsApp business profile picture", "phone_number_id", phone.ID, "error", err)
				} else if len(profileResponse.Data) > 0 {
					pictureURL = profileResponse.Data[0].ProfilePictureURL
				}

				pages[phone.ID] = graphPage{
					MetaPage: domain.MetaPage{
						ID:          phone.ID,
						Name:        name,
						Category:    "WhatsApp",
						PictureURL:  pictureURL,
						PhoneNumber: phone.DisplayPhoneNumber,
						TokenReady:  true,
					},
					AccessToken: token,
				}
			}
		}
		for _, biz := range bizResponse.Data {
			for _, edge := range []string{"owned_whatsapp_business_accounts", "client_whatsapp_business_accounts"} {
				var wabaResponse struct {
					Data []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"data"`
				}
				wabaQuery := url.Values{"fields": {"id,name"}, "access_token": {token}, "limit": {"100"}}
				if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+biz.ID+"/"+edge+"?"+wabaQuery.Encode(), &wabaResponse); err != nil {
					wabaErrors++
					diagnostics.Issues = append(diagnostics.Issues, diagnosticIssue(edge, err))
					slog.Warn("unable to discover WhatsApp Business Accounts", "business_id", biz.ID, "edge", edge, "error", err)
					continue
				}
				for _, waba := range wabaResponse.Data {
					addWABA(biz.ID, waba.ID, waba.Name)
				}
			}
		}
		// Meta can grant direct WABA access while denying the Business Portfolio
		// discovery edges. Explicit IDs keep Cloud API phone numbers available in
		// that valid setup and are safe to configure because WABA IDs are not secrets.
		for _, wabaID := range strings.Split(g.cfg.WhatsAppBusinessAccountIDs, ",") {
			addWABA("configured", wabaID, "")
		}
		switch {
		case diagnostics.PhoneNumberCount > 0:
			diagnostics.State = "ready"
			diagnostics.Message = "WhatsApp phone numbers are available."
		case diagnostics.BusinessCount == 0:
			diagnostics.State = "no_business"
			diagnostics.Message = "Meta returned no Business Portfolio for this Facebook account."
		case diagnostics.WhatsAppBusinessAccountCount == 0 && wabaErrors > 0:
			diagnostics.State = "waba_access_error"
			diagnostics.Message = "Meta found a Business Portfolio but denied access to its WhatsApp accounts."
		case diagnostics.WhatsAppBusinessAccountCount == 0:
			diagnostics.State = "no_waba"
			diagnostics.Message = "The accessible Business Portfolios do not contain a WhatsApp Business Account."
		case phoneErrors > 0:
			diagnostics.State = "phone_access_error"
			diagnostics.Message = "Meta found a WhatsApp Business Account but denied access to its phone numbers."
		default:
			diagnostics.State = "no_phone_number"
			diagnostics.Message = "The accessible WhatsApp Business Account has no registered phone number."
		}
	} else {
		diagnostics.State = "business_access_error"
		diagnostics.Message = "Meta denied access to Business Portfolios for this Facebook authorization."
		diagnostics.Issues = append(diagnostics.Issues, diagnosticIssue("businesses", err))
		slog.Warn("unable to discover Meta businesses for WhatsApp", "error", err)
	}

	return pages, diagnostics, nil
}

func diagnosticIssue(operation string, err error) string {
	issue := operation + ": " + err.Error()
	if len(issue) > 300 {
		return issue[:300]
	}
	return issue
}

func applyWhatsAppPermissions(diagnostics *domain.MetaWhatsAppDiagnostics, granted []string) {
	diagnostics.GrantedPermissions = append([]string(nil), granted...)
	grantedSet := make(map[string]bool, len(granted))
	for _, permission := range granted {
		grantedSet[permission] = true
	}
	diagnostics.MissingPermissions = diagnostics.MissingPermissions[:0]
	for _, permission := range diagnostics.RequiredPermissions {
		if !grantedSet[permission] {
			diagnostics.MissingPermissions = append(diagnostics.MissingPermissions, permission)
		}
	}
	if len(diagnostics.MissingPermissions) > 0 {
		diagnostics.State = "missing_permissions"
		diagnostics.Message = "The Facebook authorization is missing permissions required to discover WhatsApp accounts."
	}
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
