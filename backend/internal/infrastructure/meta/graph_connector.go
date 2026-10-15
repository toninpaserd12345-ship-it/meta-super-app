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
	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/usecase"
)

type GraphConfig struct {
	AppID, AppSecret, RedirectURI, Version, WebhookFields, WebhookVerifyToken, StateFile, EncryptionKey string
	WhatsAppConfigID                                                                                    string
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
	deliverySteps map[string]time.Time
	inFlight      map[string]bool
	automation    domain.AutomationProvider
	productRepo   domain.ProductRepository
	leadSink      domain.LeadSink

	adHierarchyCache map[string]adHierarchyInfo
	insightsCache    map[string]insightsCacheEntry

	profileMu    sync.RWMutex
	userProfiles map[string]struct{ Name, Pic string }
}

type insightsCacheEntry struct {
	report    *domain.MetaInsightsReport
	expiresAt time.Time
}

func (g *GraphConnector) SetChatStream(c *usecase.ChatStream) {
	g.chatStream = c
}

func (g *GraphConnector) SetAutomationProvider(p domain.AutomationProvider) {
	g.automation = p
}
func (g *GraphConnector) SetLeadSink(s domain.LeadSink) { g.leadSink = s }

func NewGraphConnector(cfg GraphConfig, db *gorm.DB) *GraphConnector {
	g := &GraphConnector{
		db:               db,
		cfg:              cfg,
		client:           &http.Client{Timeout: 45 * time.Second},
		states:           make(map[string]oauthState),
		sessions:         make(map[string]*graphSession),
		recentEvents:     make(map[string]time.Time),
		bindings:         make(map[string]domain.ProductBinding),
		products:         make(map[string]map[string]domain.Product),
		replyFlows:       make(map[string]map[string]domain.ReplyFlow),
		firstReplies:     make(map[string]time.Time),
		conversations:    make(map[string]string),
		deliverySteps:    make(map[string]time.Time),
		inFlight:         make(map[string]bool),
		userProfiles:     make(map[string]struct{ Name, Pic string }),
		adHierarchyCache: make(map[string]adHierarchyInfo),
		insightsCache:    make(map[string]insightsCacheEntry),
	}
	if err := g.loadState(); err != nil {
		slog.Warn("unable to load Meta state", "error", err)
	}
	go g.cleanupRoutine()
	return g
}

func (g *GraphConnector) cleanupRoutine() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		g.mu.Lock()
		for id, expires := range g.recentEvents {
			if now.After(expires) {
				delete(g.recentEvents, id)
			}
		}
		// insightsCache prunes expired reports
		for id, item := range g.insightsCache {
			if now.After(item.expiresAt) {
				delete(g.insightsCache, id)
			}
		}
		// firstReplies tracks anti-spam 24h
		for k, t := range g.firstReplies {
			if time.Since(t) > 24*time.Hour {
				delete(g.firstReplies, k)
			}
		}
		// deliverySteps tracks per-step delivery for 24h
		for k, t := range g.deliverySteps {
			if time.Since(t) > 24*time.Hour {
				delete(g.deliverySteps, k)
			}
		}
		// conversations cache prunes after 24h
		if len(g.conversations) > 50000 {
			g.conversations = make(map[string]string)
		}
		g.mu.Unlock()

		g.profileMu.Lock()
		// userProfiles cache is also safe to clear periodically to prevent unlimited growth
		g.userProfiles = make(map[string]struct{ Name, Pic string })
		g.profileMu.Unlock()
	}
}

func (g *GraphConnector) fetchNameViaConversation(ctx context.Context, pageID, psid, token string) (string, string) {
	// 1. Get conversation ID
	reqUrl := "https://graph.facebook.com/" + g.cfg.Version + "/" + pageID + "/conversations?user_id=" + psid + "&access_token=" + token
	req, _ := http.NewRequestWithContext(ctx, "GET", reqUrl, nil)
	resp, err := g.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return "", ""
	}

	var convList struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&convList); err != nil || len(convList.Data) == 0 {
		if resp != nil {
			resp.Body.Close()
		}
		return "", ""
	}
	resp.Body.Close()

	convID := convList.Data[0].ID

	// 2. Fetch participants
	reqUrl = "https://graph.facebook.com/" + g.cfg.Version + "/" + convID + "?fields=participants{id,name,profile_pic}&access_token=" + token
	req, _ = http.NewRequestWithContext(ctx, "GET", reqUrl, nil)
	resp2, err := g.client.Do(req)
	if err != nil || resp2.StatusCode != 200 {
		if resp2 != nil {
			resp2.Body.Close()
		}
		return "", ""
	}

	var convDetails struct {
		Participants struct {
			Data []struct {
				Name       string `json:"name"`
				ID         string `json:"id"`
				ProfilePic string `json:"profile_pic"`
			} `json:"data"`
		} `json:"participants"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&convDetails); err != nil {
		if resp2 != nil {
			resp2.Body.Close()
		}
		return "", ""
	}
	resp2.Body.Close()

	for _, p := range convDetails.Participants.Data {
		if p.ID == psid {
			return p.Name, p.ProfilePic
		}
	}
	return "", ""
}

func (g *GraphConnector) GetProfile(ctx context.Context, pageID, psid, token string) (string, string) {
	if psid == "" || token == "" {
		return "", ""
	}
	g.profileMu.RLock()
	p, ok := g.userProfiles[psid]
	g.profileMu.RUnlock()
	if ok {
		return p.Name, p.Pic
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://graph.facebook.com/"+g.cfg.Version+"/"+psid+"?fields=first_name,last_name,profile_pic&access_token="+token, nil)
	resp, err := g.client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if pageID != "" {
			name, pic := g.fetchNameViaConversation(ctx, pageID, psid, token)
			if name != "" {
				g.profileMu.Lock()
				g.userProfiles[psid] = struct{ Name, Pic string }{name, pic}
				g.profileMu.Unlock()
				return name, pic
			}
		}
		return "", ""
	}
	var res struct {
		Name       string `json:"name"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		ProfilePic string `json:"profile_pic"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", ""
	}
	name := strings.TrimSpace(res.FirstName + " " + res.LastName)

	g.profileMu.Lock()
	g.userProfiles[psid] = struct{ Name, Pic string }{name, res.ProfilePic}
	g.profileMu.Unlock()
	return name, res.ProfilePic
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
				Recipient struct {
					ID string `json:"id"`
				} `json:"recipient"`
				Timestamp int64 `json:"timestamp"`
				Message   struct {
					MID         string           `json:"mid"`
					IsEcho      bool             `json:"is_echo"`
					Text        string           `json:"text"`
					Referral    *webhookReferral `json:"referral"`
					Attachments []struct {
						Type    string `json:"type"`
						Payload struct {
							URL string `json:"url"`
						} `json:"payload"`
					} `json:"attachments"`
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
	if expires, duplicate := g.recentEvents[key]; duplicate && now.Before(expires) {
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

			replyKey := seenKey
			if rule != nil && rule.FlowID != "" {
				replyKey += ":" + rule.FlowID
			}
			// Guard against duplicate processing of the exact same Facebook Comment ID
			commentKey := "comment:" + change.Value.CommentID
			g.mu.Lock()
			if _, alreadyProcessed := g.recentEvents[commentKey]; alreadyProcessed {
				g.mu.Unlock()
				continue
			}
			g.recentEvents[commentKey] = now.Add(24 * time.Hour)

			lastReplyTime, alreadyReplied := g.firstReplies[replyKey]
			commentCooldown := 10 * time.Second
			if rule != nil && rule.CooldownSeconds > 0 {
				commentCooldown = time.Duration(rule.CooldownSeconds) * time.Second
			}
			isSpamming := alreadyReplied && time.Since(lastReplyTime) < commentCooldown

			busy := g.inFlight[replyKey]
			if found && !isSpamming && !busy {
				g.inFlight[replyKey] = true
			}
			g.mu.Unlock()
			_, page, pageFound := g.findPage(entry.ID)
			if pageFound && g.leadSink != nil {
				accountID, _, _ := g.findPage(entry.ID)
				productID := ""
				if rule != nil {
					productID = rule.ProductID
				}
				_ = g.leadSink.CaptureLead(ctx, domain.LeadCapture{AccountID: accountID, PageID: entry.ID, SenderID: change.Value.From.ID, SourceType: "post", SourceID: change.Value.PostID, ProductID: productID})
			}
			if !found || isSpamming || busy || !pageFound {
				continue
			}

			var items []domain.ReplyItem
			if g.automation != nil {
				items, _ = g.automation.GetReplySetItems(ctx, rule.ReplySetID, rule.AccountID)
			}
			items = g.resolveReplyItemsContext(ctx, rule, items)

			texts := []string{}
			for _, step := range items {
				if step.IsEnabled && step.Type == "text" && strings.TrimSpace(step.Content) != "" {
					texts = append(texts, step.Content)
				}
			}
			text := strings.Join(texts, "\n\n")
			if strings.TrimSpace(text) == "" {
				text = "ສະບາຍດີ! ພວກເຮົາໄດ້ສົ່ງລາຍລະອຽດໃຫ້ທາງຂໍ້ຄວາມແລ້ວ"
			}

			// Reply to comment publicly
			if err := g.replyToComment(ctx, page, change.Value.CommentID, text); err != nil {
				slog.Error("automatic comment reply failed", "page_id", entry.ID, "post_id", change.Value.PostID, "comment_id", change.Value.CommentID, "error", err)
			}

			// Reply privately
			if err := g.sendPrivateReply(ctx, page, change.Value.CommentID, text); err != nil {
				slog.Error("automatic private product reply failed", "page_id", entry.ID, "post_id", change.Value.PostID, "comment_id", change.Value.CommentID, "error", err)
				g.releaseReply(replyKey)
				continue
			}
			g.mu.Lock()
			g.firstReplies[replyKey] = time.Now()
			delete(g.inFlight, replyKey)
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
			accountID, page, pageFound := g.findPage(entry.ID)
			if !pageFound {
				continue
			}
			if _, err := uuid.Parse(accountID); err != nil && g.db != nil {
				var realAccount database.AccountModel
				if err := g.db.First(&realAccount).Error; err == nil {
					accountID = realAccount.ID
				}
			}

			customerID := message.Sender.ID
			isSystem := false
			if message.Sender.ID == entry.ID || message.Message.IsEcho {
				customerID = message.Recipient.ID
				isSystem = true
			}

			// 1. Deduplicate incoming messages by Meta Message ID (MID)
			if message.Message.MID != "" {
				midKey := "mid:" + message.Message.MID
				g.mu.Lock()
				if _, seen := g.recentEvents[midKey]; seen {
					g.mu.Unlock()
					slog.Info("ignoring duplicate message delivery by mid", "mid", message.Message.MID, "page_id", entry.ID)
					continue
				}
				g.recentEvents[midKey] = now.Add(24 * time.Hour)
				g.mu.Unlock()
			}

			// 2. Deduplicate standalone referral events
			if ref != nil && strings.TrimSpace(message.Message.Text) == "" {
				refID := ref.getAdID()
				if refID == "" {
					refID = ref.Ref
				}
				if refID != "" {
					refKey := fmt.Sprintf("ref:%s:%s:%s", entry.ID, customerID, refID)
					g.mu.Lock()
					if _, seen := g.recentEvents[refKey]; seen {
						g.mu.Unlock()
						slog.Info("ignoring duplicate referral delivery", "refKey", refKey)
						continue
					}
					g.recentEvents[refKey] = now.Add(10 * time.Minute)
					g.mu.Unlock()
				}
			}

			if g.leadSink != nil && !isSystem {
				sourceType, sourceID := "message", ""
				if ref != nil {
					sourceType, sourceID = "post", ref.Ref
					if adID := ref.getAdID(); adID != "" {
						sourceType, sourceID = "ad", adID
					} else if ref.AdsContextData.PostID != "" {
						sourceType, sourceID = "post", ref.AdsContextData.PostID
					} else if ref.CampaignID != "" {
						sourceType, sourceID = "campaign", ref.CampaignID
					} else if ref.AdGroupID != "" {
						sourceType, sourceID = "adset", ref.AdGroupID
					}
				}
				_ = g.leadSink.CaptureLead(ctx, domain.LeadCapture{AccountID: accountID, PageID: entry.ID, SenderID: customerID, SourceType: sourceType, SourceID: sourceID})
			}
			messageText := strings.TrimSpace(message.Message.Text)
			var directSet *domain.ReplySet
			if g.automation != nil && messageText != "" {
				directSet, _ = g.automation.FindReplySetByTrigger(ctx, accountID, messageText)
			}

			if g.chatStream != nil {
				msgType := "text"
				msgContent := message.Message.Text
				if len(message.Message.Attachments) > 0 {
					msgType = message.Message.Attachments[0].Type
					msgContent = message.Message.Attachments[0].Payload.URL
				}
				if msgContent != "" || ref != nil {
					if pageFound {
						timestampStr := fmt.Sprintf("%d", message.Timestamp)
						if message.Timestamp == 0 {
							timestampStr = fmt.Sprintf("%d", time.Now().UnixMilli())
						}

						senderName, senderPic := g.GetProfile(ctx, entry.ID, customerID, page.AccessToken)
						platform := "facebook"
						if isSystem {
							platform = "system"
						}
						broadcastContent := msgContent
						if broadcastContent == "" && ref != nil {
							broadcastContent = "Started chat from Ad / Post"
						}
						g.chatStream.Broadcast(accountID, usecase.ChatEvent{
							MID:        message.Message.MID,
							AccountID:  accountID,
							PageID:     entry.ID,
							SenderID:   customerID,
							SenderName: senderName,
							SenderPic:  senderPic,
							Message:    broadcastContent,
							Type:       msgType,
							Timestamp:  timestampStr,
							Platform:   platform,
						})
					}
				}
			}

			// If this message was sent by the page or is an echo event, NEVER trigger auto-replies!
			if isSystem || message.Message.IsEcho {
				seenKey := entry.ID + ":" + customerID
				g.mu.Lock()
				g.firstReplies[seenKey] = now
				g.mu.Unlock()
				continue
			}

			seenKey := entry.ID + ":" + customerID

			// 3. Conversation in-flight lock & 15-second debounce against dual events (e.g. referral + text)
			g.mu.RLock()
			isBusy := g.inFlight[seenKey]
			lastCustomerReplyTime, customerRecentlyReplied := g.firstReplies[seenKey]
			g.mu.RUnlock()
			if isBusy {
				slog.Info("automation reply already in-flight for customer, skipping duplicate", "page_id", entry.ID, "customer_id", customerID)
				continue
			}
			if customerRecentlyReplied && time.Since(lastCustomerReplyTime) < 15*time.Second {
				slog.Info("customer received automated reply within last 15s, debouncing", "page_id", entry.ID, "customer_id", customerID)
				continue
			}

			sourceType, sourceID, resolvedKey := "", "", ""
			if ref != nil {
				adID := ref.getAdID()
				if adID != "" {
					sourceType, sourceID = "ad", adID
				} else if ref.AdsContextData.PostID != "" {
					sourceType, sourceID = "post", ref.AdsContextData.PostID
				} else if ref.CampaignID != "" {
					sourceType, sourceID = "campaign", ref.CampaignID
				} else if ref.AdGroupID != "" {
					sourceType, sourceID = "adset", ref.AdGroupID
				} else if ref.Ref != "" {
					sourceType, sourceID = "post", ref.Ref
				}
				if sourceID != "" {
					resolvedKey = bindingKey(entry.ID, sourceType, sourceID)
				}
			} else {
				g.mu.RLock()
				resolvedKey = g.conversations[seenKey]
				g.mu.RUnlock()
				if resolvedKey != "" {
					parts := strings.Split(resolvedKey, ":")
					if len(parts) >= 3 {
						sourceType = parts[1]
						sourceID = strings.Join(parts[2:], ":")
					}
				}
			}

			// If sourceID is still unknown and we have DB access, check recent lead record
			if sourceID == "" && g.db != nil {
				var lead database.LeadModel
				if err := g.db.Where("page_id = ? AND sender_id = ? AND source_id <> '' AND source_id IS NOT NULL", entry.ID, customerID).Order("created_at DESC").First(&lead).Error; err == nil {
					if lead.SourceID != "" {
						sourceType, sourceID = lead.SourceType, lead.SourceID
						resolvedKey = bindingKey(entry.ID, sourceType, sourceID)
						g.mu.Lock()
						g.conversations[seenKey] = resolvedKey
						g.mu.Unlock()
					}
				}
			}

			hasImage := false
			for _, att := range message.Message.Attachments {
				if att.Type == "image" {
					hasImage = true
					break
				}
			}

			// Check whether customer has already received an automated reply in this conversation
			hasRepliedCustomer := false
			g.mu.RLock()
			for k := range g.firstReplies {
				if strings.HasPrefix(k, seenKey) {
					hasRepliedCustomer = true
					break
				}
			}
			g.mu.RUnlock()

			// Check database for active conversation history if not in memory (e.g. after server restart)
			if !hasRepliedCustomer && g.db != nil {
				var lastMsg database.ChatMessageModel
				if err := g.db.Model(&database.ChatMessageModel{}).
					Where("page_id = ? AND sender_id = ?", entry.ID, customerID).
					Order("created_at DESC").Limit(1).Find(&lastMsg).Error; err == nil && lastMsg.ID != "" {
					hasRepliedCustomer = true
					g.mu.Lock()
					g.firstReplies[seenKey] = lastMsg.CreatedAt
					g.mu.Unlock()
				}
			}

			var rule *domain.AutomationRule
			if g.automation != nil {
				if directSet != nil && !hasRepliedCustomer {
					sourceType, sourceID = "reply_set_name", directSet.Name
					resolvedKey = bindingKey(entry.ID, sourceType, directSet.ID)
					rule = &domain.AutomationRule{
						AccountID:        accountID,
						PageID:           entry.ID,
						TriggerType:      sourceType,
						TriggerValue:     directSet.Name,
						ReplySetID:       directSet.ID,
						FlowID:           "reply-set-name:" + directSet.ID,
						FlowName:         directSet.Name,
						CooldownSeconds:  86400,
						FirstMessageOnly: true,
						IsActive:         true,
					}
				} else if ref != nil || (sourceID != "" && !hasRepliedCustomer) {
					adID := ""
					if ref != nil {
						adID = ref.getAdID()
					} else if sourceType == "ad" {
						adID = sourceID
					}

					// 1. Direct AD match if adID is present
					if adID != "" {
						rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, "ad", adID)
					}
					// 2. Direct campaign match if referral includes campaign_id
					if rule == nil && ref != nil && ref.CampaignID != "" {
						rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, "campaign", ref.CampaignID)
					}
					// 3. Direct adgroup/adset match if referral includes adgroup_id
					if rule == nil && ref != nil && ref.AdGroupID != "" {
						rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, "adset", ref.AdGroupID)
					}
					// 4. Match by sourceType & sourceID
					if rule == nil && sourceID != "" {
						rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, sourceType, sourceID)
					}
					// 5. Match by alternative type (ad vs post)
					if rule == nil && sourceID != "" {
						altType := "ad"
						if sourceType == "ad" {
							altType = "post"
						}
						rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, altType, sourceID)
					}
					// 6. Hierarchy resolution (ad -> campaign / adset / post)
					if rule == nil && (adID != "" || (ref != nil && ref.Source == "ADS")) {
						targetAdID := adID
						if targetAdID == "" && ref != nil {
							targetAdID = ref.Ref
						}
						if targetAdID != "" {
							campaignID, adSetID, postID := g.resolveAdHierarchy(ctx, entry.ID, targetAdID)
							if campaignID != "" {
								rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, "campaign", campaignID)
							}
							if rule == nil && adSetID != "" {
								rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, "adset", adSetID)
							}
							if rule == nil && postID != "" {
								rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, "post", postID)
							}
						}
					}
					// 7. Referral ads_context_data post_id match
					if rule == nil && ref != nil && ref.AdsContextData.PostID != "" {
						rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, "post", ref.AdsContextData.PostID)
					}
					// 8. Direct campaign / custom match from ref.Ref if rule is still not found
					if rule == nil && ref != nil && ref.Ref != "" {
						rule, _ = g.automation.FindActiveRuleForTrigger(ctx, entry.ID, "campaign", ref.Ref)
					}
				}

				// 8. Keyword match
				if rule == nil {
					msgText := strings.TrimSpace(message.Message.Text)
					if msgText != "" {
						keywordRules, _ := g.automation.GetKeywordRules(ctx, entry.ID)
						for _, kr := range keywordRules {
							if matchesKeywords(msgText, []string{kr.TriggerValue}) {
								r := kr
								rule = &r
								break
							}
						}
					}
				}

				// 9. Natural language product inquiry, price question, image, or first contact fallback
				if rule == nil && g.db != nil {
					var activeRules []database.AutomationRuleModel
					if err := g.db.WithContext(ctx).Where("page_id = ? AND is_active = ?", entry.ID, true).Find(&activeRules).Error; err == nil && len(activeRules) > 0 {
						msgLower := strings.ToLower(messageText)
						// Check if message text matches a product associated with any active rule
						if msgLower != "" && g.productRepo != nil {
							for _, ar := range activeRules {
								if ar.ProductID != nil && *ar.ProductID != "" {
									prod, pErr := g.productRepo.GetProduct(ctx, accountID, *ar.ProductID)
									if pErr == nil && prod != nil {
										cLower := strings.ToLower(strings.TrimSpace(prod.Code))
										nLower := strings.ToLower(strings.TrimSpace(prod.Name))
										if (cLower != "" && len(cLower) >= 2 && strings.Contains(msgLower, cLower)) ||
											(nLower != "" && len([]rune(nLower)) >= 3 && strings.Contains(msgLower, nLower)) {
											r := ruleModelToDomain(ar)
											rule = &r
											sourceType = ar.TriggerType
											sourceID = ar.TriggerValue
											resolvedKey = bindingKey(entry.ID, sourceType, sourceID)
											break
										}
									}
								}
							}
						}

						// If still nil, only fallback if customer has NO prior replies and this is an initial inquiry or fresh ad
						if rule == nil && !hasRepliedCustomer && (ref != nil || isAskingPrice(messageText)) {
							targetRule := activeRules[0]
							for _, ar := range activeRules {
								if ar.TriggerType == "campaign" || ar.TriggerType == "ad" || ar.TriggerType == "post" {
									targetRule = ar
									break
								}
							}
							r := ruleModelToDomain(targetRule)
							rule = &r
							sourceType = targetRule.TriggerType
							sourceID = targetRule.TriggerValue
							resolvedKey = bindingKey(entry.ID, sourceType, sourceID)
						}
					}
				}
			}

			if rule != nil && resolvedKey == "" {
				resolvedKey = bindingKey(entry.ID, rule.TriggerType, rule.TriggerValue)
			}

			found := rule != nil
			if found && g.leadSink != nil && !isSystem {
				_ = g.leadSink.CaptureLead(ctx, domain.LeadCapture{AccountID: accountID, PageID: entry.ID, SenderID: customerID, SourceType: sourceType, SourceID: sourceID, ProductID: rule.ProductID})
			}
			if !found && resolvedKey == "" {
				continue
			}

			replyKey := seenKey
			if rule != nil && rule.FlowID != "" {
				replyKey += ":" + rule.FlowID
			}

			g.mu.Lock()
			// Check in-flight lock under write lock: conversation seenKey or specific replyKey
			if g.inFlight[seenKey] || g.inFlight[replyKey] {
				g.mu.Unlock()
				continue
			}

			lastReplyTime, alreadyReplied := g.firstReplies[seenKey]
			if !alreadyReplied {
				lastReplyTime, alreadyReplied = g.firstReplies[replyKey]
			}

			isSpamming := false
			if alreadyReplied {
				if time.Since(lastReplyTime) < 3*time.Second {
					// 1. Debounce rapid dual webhooks (e.g. referral + text in same second)
					isSpamming = true
				} else if ref != nil {
					// 2. Fresh Ad referral (Click-to-Messenger):
					// Only suppress if the customer received this exact reply within the rule's cooldown or 30 seconds
					minCooldown := 30 * time.Second
					if rule != nil && rule.CooldownSeconds > 0 {
						minCooldown = time.Duration(rule.CooldownSeconds) * time.Second
					}
					isSpamming = time.Since(lastReplyTime) < minCooldown
				} else if rule != nil && rule.FirstMessageOnly {
					// 3. Regular chat messages: FirstMessageOnly must never repeat mid-conversation
					isSpamming = true
				} else if rule != nil && (rule.TriggerType == "ad" || rule.TriggerType == "campaign" || rule.TriggerType == "post") {
					// 4. Ad/Post/Campaign welcome flows require a fresh ad referral; NEVER re-fire in mid-chat without referral
					isSpamming = true
				} else if rule != nil && rule.CooldownSeconds > 0 {
					// 5. Custom cooldown (minimum 15 minutes in active chats)
					cooldown := time.Duration(rule.CooldownSeconds) * time.Second
					if cooldown < 15*time.Minute {
						cooldown = 15 * time.Minute
					}
					isSpamming = time.Since(lastReplyTime) < cooldown
				} else {
					// 6. Default safe cooldown for all flows: 24 hours
					isSpamming = time.Since(lastReplyTime) < 24*time.Hour
				}
			}

			if isSpamming || !found || !pageFound {
				g.mu.Unlock()
				continue
			}

			// Immediately set in-flight and firstReplies to BLOCK any duplicate incoming events
			nowReply := time.Now()
			g.inFlight[seenKey] = true
			g.inFlight[replyKey] = true
			g.firstReplies[seenKey] = nowReply
			g.firstReplies[replyKey] = nowReply
			if resolvedKey != "" {
				g.conversations[seenKey] = resolvedKey
			}
			g.mu.Unlock()

			var items []domain.ReplyItem
			if g.automation != nil {
				items, _ = g.automation.GetReplySetItems(ctx, rule.ReplySetID, rule.AccountID)
			}
			items = g.resolveReplyItemsContext(ctx, rule, items)

			go func(rule *domain.AutomationRule, items []domain.ReplyItem, seenKey, resolvedKey, replyKey, pageID, recipientID, sourceType, sourceID, messageText string, hasImage bool) {
				sendCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				defer func() {
					g.mu.Lock()
					delete(g.inFlight, seenKey)
					delete(g.inFlight, replyKey)
					nowDone := time.Now()
					g.firstReplies[seenKey] = nowDone
					g.firstReplies[replyKey] = nowDone
					g.mu.Unlock()
				}()

				// Mark conversation as seen
				_ = g.SendSenderAction(sendCtx, pageID, recipientID, "mark_seen")
				time.Sleep(400 * time.Millisecond)

				for index, step := range items {
					if !step.IsEnabled {
						continue
					}
					stepKey := seenKey + ":" + step.ID
					g.mu.RLock()
					stepDone := !g.deliverySteps[stepKey].IsZero()
					g.mu.RUnlock()
					if stepDone {
						continue
					}

					if step.Type == "image" {
						time.Sleep(800 * time.Millisecond)
					} else if step.Type == "text" {
						if index > 0 {
							time.Sleep(600 * time.Millisecond)
						}
						_ = g.SendSenderAction(sendCtx, pageID, recipientID, "typing_on")
						runeCount := len([]rune(step.Content))
						delayMs := 1000 + (runeCount * 12)
						if delayMs < 1200 {
							delayMs = 1200
						} else if delayMs > 2500 {
							delayMs = 2500
						}
						time.Sleep(time.Duration(delayMs) * time.Millisecond)
					} else {
						time.Sleep(1000 * time.Millisecond)
					}

					var err error
					if step.Type == "text" {
						err = g.SendMessage(sendCtx, rule.AccountID, pageID, recipientID, step.Content)
					} else {
						err = g.SendMedia(sendCtx, rule.AccountID, pageID, recipientID, step.Type, step.Content)
					}
					if err != nil {
						slog.Error("automatic flow message step failed", "page_id", pageID, "recipient_id", recipientID, "source_id", sourceID, "step", index+1, "type", step.Type, "error", err)
						continue
					}
					g.mu.Lock()
					g.deliverySteps[stepKey] = time.Now()
					g.mu.Unlock()
				}
				slog.Info("automatic product reply sent", "page_id", pageID, "recipient_id", recipientID, "source_type", sourceType, "source_id", sourceID)
			}(rule, items, seenKey, resolvedKey, replyKey, page.ID, customerID, sourceType, sourceID, messageText, hasImage)
		}
	}
	return nil
}

type adHierarchyInfo struct {
	campaignID string
	adSetID    string
	postID     string
	expiresAt  time.Time
}

func (g *GraphConnector) resolveAdHierarchy(ctx context.Context, pageID, adID string) (string, string, string) {
	if adID == "" {
		return "", "", ""
	}
	g.mu.RLock()
	if cached, ok := g.adHierarchyCache[adID]; ok && time.Now().Before(cached.expiresAt) {
		g.mu.RUnlock()
		return cached.campaignID, cached.adSetID, cached.postID
	}
	g.mu.RUnlock()

	accountID, page, ok := g.findPage(pageID)
	if !ok {
		return "", "", ""
	}
	token, err := g.userToken(ctx, accountID)
	if err != nil || token == "" {
		token = page.AccessToken
	}
	if token == "" {
		return "", "", ""
	}

	type adGraphResp struct {
		CampaignID string `json:"campaign_id"`
		AdSetID    string `json:"adset_id"`
		Campaign   struct {
			ID string `json:"id"`
		} `json:"campaign"`
		AdSet struct {
			ID string `json:"id"`
		} `json:"adset"`
		Creative struct {
			EffectiveObjectStoryID string `json:"effective_object_story_id"`
		} `json:"creative"`
	}

	var response adGraphResp
	query := url.Values{"fields": {"campaign_id,adset_id,campaign{id},adset{id},creative{effective_object_story_id}"}, "access_token": {token}}
	endpoint := "https://graph.facebook.com/" + g.cfg.Version + "/" + url.PathEscape(adID) + "?" + query.Encode()
	if err := g.getJSON(ctx, endpoint, &response); err != nil {
		// Fallback without creative in case creative query is rejected
		queryFallback := url.Values{"fields": {"campaign_id,adset_id,campaign{id},adset{id}"}, "access_token": {token}}
		endpointFallback := "https://graph.facebook.com/" + g.cfg.Version + "/" + url.PathEscape(adID) + "?" + queryFallback.Encode()
		if err2 := g.getJSON(ctx, endpointFallback, &response); err2 != nil {
			if page.AccessToken != "" && page.AccessToken != token {
				queryFallback.Set("access_token", page.AccessToken)
				_ = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(adID)+"?"+queryFallback.Encode(), &response)
			}
		}
	}
	campaignID := response.Campaign.ID
	if campaignID == "" {
		campaignID = response.CampaignID
	}
	adSetID := response.AdSet.ID
	if adSetID == "" {
		adSetID = response.AdSetID
	}
	postID := response.Creative.EffectiveObjectStoryID
	if campaignID != "" || adSetID != "" || postID != "" {
		g.mu.Lock()
		if g.adHierarchyCache == nil {
			g.adHierarchyCache = make(map[string]adHierarchyInfo)
		}
		g.adHierarchyCache[adID] = adHierarchyInfo{
			campaignID: campaignID,
			adSetID:    adSetID,
			postID:     postID,
			expiresAt:  time.Now().Add(24 * time.Hour),
		}
		g.mu.Unlock()
	}
	return campaignID, adSetID, postID
}

type webhookReferral struct {
	Ref            string `json:"ref"`
	Source         string `json:"source"`
	Type           string `json:"type"`
	AdID           string `json:"ad_id"`
	AdGroupID      string `json:"adgroup_id"`
	CampaignID     string `json:"campaign_id"`
	AdsContextData struct {
		AdID      string `json:"ad_id"`
		AdTitle   string `json:"ad_title"`
		PhotoURL  string `json:"photo_url"`
		VideoURL  string `json:"video_url"`
		PostID    string `json:"post_id"`
		ProductID string `json:"product_id"`
	} `json:"ads_context_data"`
}

func (r *webhookReferral) getAdID() string {
	if r == nil {
		return ""
	}
	if id := strings.TrimSpace(r.AdID); id != "" {
		return id
	}
	if id := strings.TrimSpace(r.AdsContextData.AdID); id != "" {
		return id
	}
	if r.Source == "ADS" {
		ref := strings.TrimSpace(r.Ref)
		if strings.HasPrefix(ref, "ad_id:") {
			return strings.TrimPrefix(ref, "ad_id:")
		}
		if strings.HasPrefix(ref, "ad_id=") {
			return strings.TrimPrefix(ref, "ad_id=")
		}
		if strings.HasPrefix(ref, "ad:") {
			return strings.TrimPrefix(ref, "ad:")
		}
		allDigits := len(ref) >= 8
		for _, ch := range ref {
			if ch < '0' || ch > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return ref
		}
	}
	return ""
}

func ruleModelToDomain(m database.AutomationRuleModel) domain.AutomationRule {
	prodID := ""
	if m.ProductID != nil {
		prodID = *m.ProductID
	}
	return domain.AutomationRule{
		ID:               m.ID,
		AccountID:        m.AccountID,
		PageID:           m.PageID,
		TriggerType:      m.TriggerType,
		TriggerName:      m.TriggerName,
		TriggerValue:     m.TriggerValue,
		ProductID:        prodID,
		ReplySetID:       m.ReplySetID,
		FlowID:           m.FlowID,
		FlowName:         m.FlowName,
		FirstMessageOnly: m.FirstMessageOnly,
		CooldownSeconds:  m.CooldownSeconds,
		IsActive:         m.IsActive,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
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
	return g.resolveReplyItemsContext(context.Background(), rule, items)
}

func (g *GraphConnector) resolveReplyItemsContext(ctx context.Context, rule *domain.AutomationRule, items []domain.ReplyItem) []domain.ReplyItem {
	if rule == nil || rule.ProductID == "" {
		return append([]domain.ReplyItem(nil), items...)
	}
	var product domain.Product
	found := false
	if g.productRepo != nil {
		if stored, err := g.productRepo.GetProduct(ctx, rule.AccountID, rule.ProductID); err == nil && stored != nil {
			product, found = *stored, true
		}
	}
	if !found {
		g.mu.RLock()
		product, found = g.products[rule.AccountID][rule.ProductID]
		g.mu.RUnlock()
	}
	resolved := make([]domain.ReplyItem, len(items))
	copy(resolved, items)
	if !found {
		return resolved
	}
	for index := range resolved {
		resolved[index].Content = strings.ReplaceAll(resolved[index].Content, "{{product.name}}", product.Name)
		resolved[index].Content = strings.ReplaceAll(resolved[index].Content, "{{product.price}}", product.Price)
		resolved[index].Content = strings.ReplaceAll(resolved[index].Content, "{{product.sale_price}}", formatProductNumber(product.SalePrice))
		resolved[index].Content = strings.ReplaceAll(resolved[index].Content, "{{product.cost}}", formatProductNumber(product.Cost))
		resolved[index].Content = strings.ReplaceAll(resolved[index].Content, "{{product.quantity}}", formatProductInteger(product.Quantity))
		resolved[index].Content = strings.ReplaceAll(resolved[index].Content, "{{product.currency}}", product.Currency)
		resolved[index].Content = strings.ReplaceAll(resolved[index].Content, "{{product.description}}", product.Description)
	}
	return resolved
}

func formatProductNumber(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func formatProductInteger(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
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

var priceKeywords = []string{
	// Lao price variations & misspellings
	"ລາຄາ", "ລາຂາ", "ລະຄາ", "ລາກາ", "ລາຄ່າ", "ລາຂ່າ", "ຂໍລາຄາ", "ຂໍລາຄ່າ", "ລາຄາເທົ່າໃດ", "ລາຄາທໍ່ໃດ", "ລາຄາແນວໃດ", "ລາຄາຈັກບາດ", "ລາຄາຈັກກີບ",
	// Lao how much variations & misspellings
	"ທໍ່ໃດ", "ເທົ່າໃດ", "ທໍໃດ", "ເທົາໃດ", "ທໍ່ໄດ", "ເທົ່າໄດ", "ເທົາໄດ", "ທໍໄດ", "ທໍ່ໄດ໋", "ເທົ່າໃດ໋",
	"ທໍ່ໃດນິ", "ທໍ່ໃດນີ້", "ເທົ່າໃດນິ", "ເທົ່າໃດນີ້", "ທໍໃດນິ", "ທໍໃດນີ້",
	"ຈັກບາດ", "ຈັກກີບ", "ຈັກພັນ", "ຈັກຫມື່ນ", "ຈັກໝື່ນ", "ຈັກແສນ", "ຈັກລ້ານ", "ຈັກບາດນິ", "ຈັກກີບນິ",
	"ຂາຍແນວໃດ", "ຂາຍຈັ່ງໃດ", "ຂາຍແນວໄດ", "ຂາຍຈັ່ງໄດ", "ໂຕລະທໍ່ໃດ", "ອັນລະທໍ່ໃດ", "ຊຸດລະທໍ່ໃດ",
	// Lao interest & contact
	"ສົນໃຈ", "ສນໃຈ", "ສົນໄຈ", "ສນໄຈ", "ສົນໃຈເດີ້", "ສົນໃຈເຈົ້າ", "ສົນໃຈຄັບ", "ຂໍລາຍລະອຽດ", "ມີເຄື່ອງບໍ່", "ຍັງມີບໍ່",

	// Thai price & how much variations & misspellings
	"ราคา", "ราค่า", "ระคา", "ลากา", "ขอราคา", "มีราคาไหม", "ราคาเท่าไหร่", "ราคาเท่าไร",
	"เท่าไหร่", "เท่าไร", "เท่ารัย", "เท่าไหร่คะ", "เท่าไหร่ครับ", "เท่าไรครับ", "เท่าไรคะ",
	"กี่บาท", "กี่กีบ", "ตัวละเท่าไหร่", "อันละเท่าไหร่", "ชุดละเท่าไหร่", "ชิ้นละเท่าไหร่",
	"ขายยังไง", "ขายเท่าไหร่", "มีของไหม", "ยังมีไหม",
	// Thai interest & contact
	"สนใจ", "สนใจครับ", "สนใจค่ะ", "ขอรายละเอียด", "สอบถาม", "สอบถามราคา",

	// Slang / abbreviations
	"pm", "ib", "inbox", "dm",

	// English
	"price", "how much", "howmuch", "cost",
}

func isAskingPrice(text string) bool {
	if text == "" {
		return false
	}
	cleaned := strings.ToLower(strings.TrimSpace(text))
	noSpaces := strings.ReplaceAll(cleaned, " ", "")
	for _, kw := range priceKeywords {
		kwLower := strings.ToLower(kw)
		if strings.Contains(cleaned, kwLower) || strings.Contains(noSpaces, strings.ReplaceAll(kwLower, " ", "")) {
			return true
		}
	}
	return false
}

func (g *GraphConnector) findPage(pageID string) (string, graphPage, bool) {
	g.mu.RLock()
	for accountID, session := range g.sessions {
		if _, err := uuid.Parse(accountID); err == nil {
			if page, ok := session.Pages[pageID]; ok && page.AccessToken != "" {
				g.mu.RUnlock()
				return accountID, page, true
			}
		}
	}
	for accountID, session := range g.sessions {
		if page, ok := session.Pages[pageID]; ok && page.AccessToken != "" {
			g.mu.RUnlock()
			return accountID, page, true
		}
	}
	g.mu.RUnlock()

	if g.db != nil {
		var whatsApp []database.MetaWhatsAppConnectionModel
		if err := g.db.Where("phone_number_id = ?", pageID).Limit(1).Find(&whatsApp).Error; err == nil && len(whatsApp) > 0 {
			decrypted := g.decryptToken(whatsApp[0].AccessToken)
			page := graphPage{MetaPage: domain.MetaPage{
				ID:            whatsApp[0].PhoneNumberID,
				Name:          firstNonEmpty(whatsApp[0].VerifiedName, whatsApp[0].DisplayPhoneNumber, "WhatsApp"),
				Category:      "WhatsApp",
				PictureURL:    whatsApp[0].PictureURL,
				PhoneNumber:   whatsApp[0].DisplayPhoneNumber,
				Connected:     whatsApp[0].IsConnected,
				TokenReady:    decrypted != "",
				WebhookStatus: map[bool]string{true: "subscribed", false: "disconnected"}[whatsApp[0].IsConnected],
			}, AccessToken: decrypted}
			g.cachePage(whatsApp[0].AccountID, page)
			return whatsApp[0].AccountID, page, true
		}
		var models []database.MetaPageTokenModel
		if err := g.db.Where("page_id = ?", pageID).Limit(1).Find(&models).Error; err == nil && len(models) > 0 {
			model := models[0]
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

		// Fallback: Check all active user Meta connections to discover and auto-sync newly routed or un-synced pages
		var connections []database.MetaConnectionModel
		if err := g.db.Order("updated_at DESC").Find(&connections).Error; err == nil && len(connections) > 0 {
			for _, conn := range connections {
				userToken := g.decryptToken(conn.UserToken)
				if userToken == "" {
					continue
				}
				fetchedPages, _, fetchErr := g.fetchPages(context.Background(), userToken)
				if fetchErr != nil {
					continue
				}
				for pID, fPage := range fetchedPages {
					if fPage.Category != "WhatsApp" && fPage.AccessToken != "" {
						encTok := g.encryptToken(fPage.AccessToken)
						if encTok != "" {
							_ = g.db.Save(&database.MetaPageTokenModel{
								PageID:      pID,
								AccountID:   conn.AccountID,
								Name:        fPage.Name,
								Category:    fPage.Category,
								PictureURL:  fPage.PictureURL,
								AccessToken: encTok,
								IsConnected: true,
							}).Error
						}
					}
				}
				if targetPage, found := fetchedPages[pageID]; found {
					g.mu.Lock()
					if g.sessions[conn.AccountID] == nil {
						g.sessions[conn.AccountID] = &graphSession{Pages: make(map[string]graphPage)}
					}
					g.sessions[conn.AccountID].Pages[pageID] = targetPage
					g.mu.Unlock()
					return conn.AccountID, targetPage, true
				}
			}
		}
	}
	return "", graphPage{}, false
}

func (g *GraphConnector) AuthorizationURL(_ context.Context, userID, accountID string, requested []string) (string, error) {
	allowed := map[string]bool{"pages_show_list": true, "pages_manage_metadata": true, "pages_read_engagement": true, "pages_messaging": true, "pages_manage_posts": true, "read_insights": true, "ads_read": true, "ads_management": true, "business_management": true, "whatsapp_business_management": true, "whatsapp_business_messaging": true}
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

		for id, page := range pages {
			if page.Category != "WhatsApp" && page.AccessToken != "" {
				encTok := g.encryptToken(page.AccessToken)
				if encTok != "" {
					_ = g.db.Save(&database.MetaPageTokenModel{
						PageID:      id,
						AccountID:   pending.AccountID,
						Name:        page.Name,
						Category:    page.Category,
						PictureURL:  page.PictureURL,
						AccessToken: encTok,
						IsConnected: true,
					}).Error
				}
				// Auto-subscribe page to webhook events on Meta immediately upon completing OAuth
				go func(pID, tok string) {
					subCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer cancel()
					if subErr := g.SubscribePageOnMeta(subCtx, pID, tok); subErr != nil {
						slog.Warn("auto-subscription on oauth completion failed", "page_id", pID, "error", subErr)
					} else {
						slog.Info("auto-subscribed page on Meta during oauth completion", "page_id", pID)
					}
				}(id, page.AccessToken)
			}
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
		// Auto-heal: If page was supposed to be connected in DB but Meta dropped/lacked subscription, re-subscribe immediately!
		if !connected && page.AccessToken != "" && page.Category != "WhatsApp" {
			var existing database.MetaPageTokenModel
			if g.db != nil && g.db.Where("page_id = ? AND is_connected = ?", id, true).First(&existing).Error == nil {
				if subErr := g.SubscribePageOnMeta(ctx, id, page.AccessToken); subErr == nil {
					connected = true
					slog.Info("auto-healed page webhook subscription on Meta in ListPages", "page_id", id)
				} else {
					slog.Warn("auto-heal page webhook subscription failed in ListPages", "page_id", id, "error", subErr)
				}
			}
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
		if g.db != nil && accountID != "" && page.Category != "WhatsApp" && page.AccessToken != "" {
			encTok := g.encryptToken(page.AccessToken)
			if encTok != "" {
				var existing database.MetaPageTokenModel
				if err := g.db.Where("page_id = ?", id).Limit(1).Find(&existing).Error; err != nil || existing.PageID == "" {
					_ = g.db.Save(&database.MetaPageTokenModel{
						PageID:      id,
						AccountID:   accountID,
						Name:        page.Name,
						Category:    page.Category,
						PictureURL:  page.PictureURL,
						AccessToken: encTok,
						IsConnected: page.Connected,
					}).Error
				} else {
					_ = g.db.Model(&existing).Updates(map[string]interface{}{
						"name":         page.Name,
						"category":     page.Category,
						"picture_url":  page.PictureURL,
						"access_token": encTok,
						"is_connected": page.Connected,
					}).Error
				}
			}
		}
	}
	if err := g.persistStateLocked(); err != nil {
		slog.Warn("unable to persist refreshed Meta Pages", "error", err)
	}
	g.mu.Unlock()
	result := make([]domain.MetaPage, 0, len(pages))
	for _, page := range pages {
		result = append(result, page.MetaPage)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].ID < result[j].ID
		}
		return result[i].Name < result[j].Name
	})
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

func (g *GraphConnector) SubscribePageOnMeta(ctx context.Context, pageID, accessToken string) error {
	if pageID == "" || accessToken == "" {
		return errors.New("pageID and accessToken are required")
	}
	form := url.Values{"access_token": {accessToken}}
	fields := strings.TrimSpace(g.cfg.WebhookFields)
	if fields == "" {
		fields = "feed,messages,messaging_postbacks,message_deliveries,message_reads,messaging_referrals"
	}
	if !strings.Contains(fields, "messaging_referrals") {
		fields += ",messaging_referrals"
	}
	form.Set("subscribed_fields", fields)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(pageID)+"/subscribed_apps", strings.NewReader(form.Encode()))
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
		Success bool        `json:"success"`
		Error   *graphError `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	if resp.StatusCode >= 400 || envelope.Error != nil {
		return graphResponseError(resp.StatusCode, envelope.Error)
	}
	return nil
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
		if err := g.SubscribePageOnMeta(ctx, pageID, page.AccessToken); err != nil {
			return nil, err
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
	query := url.Values{"fields": {"id,name,status,effective_status,objective,adsets.summary(1).limit(1),ads.summary(1).limit(1)"}, "limit": {"250"}, "access_token": {token}}
	endpoint := "https://graph.facebook.com/" + g.cfg.Version + "/" + url.PathEscape(adAccountID) + "/campaigns?" + query.Encode()
	items := make([]domain.MetaCampaign, 0)
	pageCount := 0
	for endpoint != "" && pageCount < 10 {
		pageCount++
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
			Paging struct {
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err = g.getJSON(ctx, endpoint, &response); err != nil {
			return nil, err
		}
		for _, item := range response.Data {
			if item.EffectiveStatus == "DELETED" || item.Status == "DELETED" {
				continue
			}
			items = append(items, domain.MetaCampaign{ID: item.ID, Name: item.Name, Status: item.Status, EffectiveStatus: item.EffectiveStatus, Objective: item.Objective, AdSetCount: item.AdSets.Summary.TotalCount, AdCount: item.Ads.Summary.TotalCount})
		}
		endpoint = response.Paging.Next
	}
	return items, nil
}

func (g *GraphConnector) ListAds(ctx context.Context, _ string, accountID, campaignID string) ([]domain.MetaAd, error) {
	token, err := g.userToken(ctx, accountID)
	if err != nil {
		return []domain.MetaAd{}, nil
	}
	query := url.Values{"fields": {"id,name,status,effective_status,creative{thumbnail_url,image_url}"}, "limit": {"200"}, "access_token": {token}}
	endpoint := "https://graph.facebook.com/" + g.cfg.Version + "/" + url.PathEscape(campaignID) + "/ads?" + query.Encode()
	items := make([]domain.MetaAd, 0)
	pageCount := 0
	for endpoint != "" && pageCount < 10 {
		pageCount++
		var response struct {
			Data []struct {
				ID              string `json:"id"`
				Name            string `json:"name"`
				Status          string `json:"status"`
				EffectiveStatus string `json:"effective_status"`
				Creative        struct {
					ThumbnailURL string `json:"thumbnail_url"`
					ImageURL     string `json:"image_url"`
				} `json:"creative"`
			} `json:"data"`
			Paging struct {
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err = g.getJSON(ctx, endpoint, &response); err != nil {
			return nil, err
		}
		for _, item := range response.Data {
			items = append(items, domain.MetaAd{ID: item.ID, Name: item.Name, Status: item.Status, EffectiveStatus: item.EffectiveStatus, ThumbnailURL: item.Creative.ThumbnailURL, ImageURL: item.Creative.ImageURL})
		}
		endpoint = response.Paging.Next
	}
	return items, nil
}

type graphInsightAction struct {
	ActionType string `json:"action_type"`
	Value      string `json:"value"`
}

type graphInsightRow struct {
	CampaignID             string               `json:"campaign_id"`
	CampaignName           string               `json:"campaign_name"`
	AdSetID                string               `json:"adset_id"`
	AdSetName              string               `json:"adset_name"`
	AdID                   string               `json:"ad_id"`
	AdName                 string               `json:"ad_name"`
	DateStart              string               `json:"date_start"`
	DateStop               string               `json:"date_stop"`
	Spend                  string               `json:"spend"`
	Impressions            string               `json:"impressions"`
	Reach                  string               `json:"reach"`
	Frequency              string               `json:"frequency"`
	InlineLinkClicks       string               `json:"inline_link_clicks"`
	InlineLinkClickCTR     string               `json:"inline_link_click_ctr"`
	CostPerInlineLinkClick string               `json:"cost_per_inline_link_click"`
	CPM                    string               `json:"cpm"`
	QualityRanking         string               `json:"quality_ranking"`
	EngagementRanking      string               `json:"engagement_rate_ranking"`
	ConversionRanking      string               `json:"conversion_rate_ranking"`
	Actions                []graphInsightAction `json:"actions"`
	CostPerActionType      []graphInsightAction `json:"cost_per_action_type"`
	ActionValues           []graphInsightAction `json:"action_values"`
	PurchaseROAS           []graphInsightAction `json:"purchase_roas"`
	WebsitePurchaseROAS    []graphInsightAction `json:"website_purchase_roas"`
}

func numberPointer(value string) *float64 {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &number
}

func selectedInsightResult(actions, costs []graphInsightAction) domain.MetaInsightResult {
	values := make(map[string]string, len(actions))
	costValues := make(map[string]string, len(costs))
	for _, item := range actions {
		values[item.ActionType] = item.Value
	}
	for _, item := range costs {
		costValues[item.ActionType] = item.Value
	}
	groups := []struct {
		canonical string
		matches   func(string) bool
	}{
		{"purchase", func(value string) bool {
			return value == "purchase" || strings.HasSuffix(value, "_purchase") || strings.Contains(value, ".purchase")
		}},
		{"lead", func(value string) bool {
			return value == "lead" || strings.HasSuffix(value, ".lead") || strings.Contains(value, "lead_grouped")
		}},
		{"messaging_conversation_started_7d", func(value string) bool { return strings.Contains(value, "messaging_conversation_started") }},
		{"onsite_conversion.messaging_first_reply", func(value string) bool { return strings.Contains(value, "messaging_first_reply") }},
		{"landing_page_view", func(value string) bool { return value == "landing_page_view" }},
		{"link_click", func(value string) bool { return value == "link_click" }},
	}
	for _, group := range groups {
		for actionType, value := range values {
			if group.matches(actionType) {
				return domain.MetaInsightResult{Type: group.canonical, Value: numberPointer(value), Cost: numberPointer(costValues[actionType])}
			}
		}
	}
	return domain.MetaInsightResult{}
}

func selectedConversionValue(resultType string, values []graphInsightAction) *float64 {
	for _, item := range values {
		matches := resultType == "purchase" && (item.ActionType == "purchase" || strings.HasSuffix(item.ActionType, "_purchase") || strings.Contains(item.ActionType, ".purchase"))
		if matches {
			return numberPointer(item.Value)
		}
	}
	return nil
}

func firstROAS(groups ...[]graphInsightAction) *float64 {
	for _, group := range groups {
		for _, item := range group {
			if value := numberPointer(item.Value); value != nil {
				return value
			}
		}
	}
	return nil
}

func insightItem(level string, row graphInsightRow) domain.MetaInsightItem {
	result := selectedInsightResult(row.Actions, row.CostPerActionType)
	item := domain.MetaInsightItem{
		EntityType: level, CampaignID: row.CampaignID, CampaignName: row.CampaignName,
		AdSetID: row.AdSetID, AdSetName: row.AdSetName, AdID: row.AdID, AdName: row.AdName,
		Spend: numberPointer(row.Spend), Impressions: numberPointer(row.Impressions), Reach: numberPointer(row.Reach),
		Frequency: numberPointer(row.Frequency), LinkClicks: numberPointer(row.InlineLinkClicks), LinkCTR: numberPointer(row.InlineLinkClickCTR),
		LinkCPC: numberPointer(row.CostPerInlineLinkClick), CPM: numberPointer(row.CPM), Results: result,
		ConversionValue: selectedConversionValue(result.Type, row.ActionValues), ROAS: firstROAS(row.PurchaseROAS, row.WebsitePurchaseROAS),
		QualityRanking: row.QualityRanking, EngagementRateRanking: row.EngagementRanking, ConversionRateRanking: row.ConversionRanking,
	}
	if item.ROAS == nil && item.ConversionValue != nil && item.Spend != nil && *item.Spend > 0 {
		value := *item.ConversionValue / *item.Spend
		item.ROAS = &value
	}
	switch level {
	case "account":
		item.EntityID, item.EntityName = row.CampaignID, "Account"
	case "campaign":
		item.EntityID, item.EntityName = row.CampaignID, row.CampaignName
	case "adset":
		item.EntityID, item.EntityName = row.AdSetID, row.AdSetName
	case "ad":
		item.EntityID, item.EntityName = row.AdID, row.AdName
	}
	return item
}

func (g *GraphConnector) fetchInsightRows(ctx context.Context, token, adAccountID, level, since, until, increment string) ([]graphInsightRow, error) {
	fields := "date_start,date_stop,spend,impressions,reach,frequency,inline_link_clicks,inline_link_click_ctr,cost_per_inline_link_click,cpm,actions,cost_per_action_type,action_values,purchase_roas,website_purchase_roas"
	switch level {
	case "campaign":
		fields = "campaign_id,campaign_name," + fields
	case "adset":
		fields = "campaign_id,campaign_name,adset_id,adset_name," + fields
	case "ad":
		fields = "campaign_id,campaign_name,adset_id,adset_name,ad_id,ad_name," + fields + ",quality_ranking,engagement_rate_ranking,conversion_rate_ranking"
	}
	query := url.Values{
		"fields": {fields}, "level": {level}, "time_range": {fmt.Sprintf(`{"since":"%s","until":"%s"}`, since, until)},
		"limit": {"500"}, "access_token": {token},
	}
	if increment != "" {
		query.Set("time_increment", increment)
	}
	endpoint := "https://graph.facebook.com/" + g.cfg.Version + "/" + url.PathEscape(adAccountID) + "/insights?" + query.Encode()
	rows := make([]graphInsightRow, 0)
	pageCount := 0
	for endpoint != "" && pageCount < 10 {
		pageCount++
		var response struct {
			Data   []graphInsightRow `json:"data"`
			Paging struct {
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err := g.getJSON(ctx, endpoint, &response); err != nil {
			return nil, err
		}
		rows = append(rows, response.Data...)
		endpoint = response.Paging.Next
	}
	return rows, nil
}

type graphAdEntityDetails struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	EffectiveStatus string `json:"effective_status"`
	CampaignID      string `json:"campaign_id"`
	Campaign        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"campaign"`
	AdSetID string `json:"adset_id"`
	AdSet   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"adset"`
	CreatedTime string `json:"created_time"`
	UpdatedTime string `json:"updated_time"`
}

func (g *GraphConnector) fetchAdEntities(ctx context.Context, token, adAccountID, level string) (map[string]graphAdEntityDetails, error) {
	edges := map[string]string{"campaign": "campaigns", "adset": "adsets", "ad": "ads"}
	edge, ok := edges[level]
	if !ok {
		return map[string]graphAdEntityDetails{}, nil
	}
	fields := "id,name,status,effective_status,created_time,updated_time"
	switch level {
	case "adset":
		fields = "id,name,status,effective_status,campaign_id,campaign{id,name},created_time,updated_time"
	case "ad":
		fields = "id,name,status,effective_status,campaign_id,campaign{id,name},adset_id,adset{id,name},created_time,updated_time"
	}
	query := url.Values{"fields": {fields}, "limit": {"250"}, "access_token": {token}}
	endpoint := "https://graph.facebook.com/" + g.cfg.Version + "/" + url.PathEscape(adAccountID) + "/" + edge + "?" + query.Encode()
	entities := make(map[string]graphAdEntityDetails)
	pageCount := 0
	for endpoint != "" && pageCount < 10 {
		pageCount++
		var response struct {
			Data   []graphAdEntityDetails `json:"data"`
			Paging struct {
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err := g.getJSON(ctx, endpoint, &response); err != nil {
			return nil, err
		}
		for _, item := range response.Data {
			entities[item.ID] = item
		}
		endpoint = response.Paging.Next
	}
	return entities, nil
}

func (g *GraphConnector) AdsInsights(ctx context.Context, _ string, accountID, adAccountID, level, since, until string) (*domain.MetaInsightsReport, error) {
	token, err := g.userToken(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(adAccountID, "act_") {
		adAccountID = "act_" + adAccountID
	}
	cacheKey := fmt.Sprintf("%s:%s:%s:%s", adAccountID, level, since, until)
	g.mu.RLock()
	if cached, ok := g.insightsCache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		g.mu.RUnlock()
		return cached.report, nil
	}
	g.mu.RUnlock()

	var (
		account struct {
			Currency string `json:"currency"`
			Timezone string `json:"timezone_name"`
		}
		rows        []graphInsightRow
		entities    map[string]graphAdEntityDetails
		accountRows []graphInsightRow
		daily       []graphInsightRow

		errAccount, errRows, errEntities, errAccountRows, errDaily error
		wg                                                         sync.WaitGroup
	)

	wg.Add(5)

	go func() {
		defer wg.Done()
		accountQuery := url.Values{"fields": {"currency,timezone_name"}, "access_token": {token}}
		errAccount = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(adAccountID)+"?"+accountQuery.Encode(), &account)
	}()

	go func() {
		defer wg.Done()
		rows, errRows = g.fetchInsightRows(ctx, token, adAccountID, level, since, until, "")
	}()

	go func() {
		defer wg.Done()
		entities, errEntities = g.fetchAdEntities(ctx, token, adAccountID, level)
	}()

	go func() {
		defer wg.Done()
		accountRows, errAccountRows = g.fetchInsightRows(ctx, token, adAccountID, "account", since, until, "")
	}()

	go func() {
		defer wg.Done()
		daily, errDaily = g.fetchInsightRows(ctx, token, adAccountID, "account", since, until, "1")
	}()

	wg.Wait()

	if errAccount != nil {
		return nil, errAccount
	}
	if errRows != nil {
		return nil, errRows
	}
	if errEntities != nil {
		return nil, errEntities
	}
	if errAccountRows != nil {
		return nil, errAccountRows
	}
	if errDaily != nil {
		return nil, errDaily
	}

	seenEntities := make(map[string]bool)
	items := make([]domain.MetaInsightItem, 0, len(rows)+len(entities))
	for _, row := range rows {
		item := insightItem(level, row)
		if entity, ok := entities[item.EntityID]; ok {
			item.Status = entity.Status
			item.EffectiveStatus = entity.EffectiveStatus
			if item.EntityName == "" {
				item.EntityName = entity.Name
			}
			if item.CampaignName == "" && entity.Campaign.Name != "" {
				item.CampaignName = entity.Campaign.Name
			}
			if item.AdSetName == "" && entity.AdSet.Name != "" {
				item.AdSetName = entity.AdSet.Name
			}
			item.CreatedTime = entity.CreatedTime
			item.UpdatedTime = entity.UpdatedTime
		}
		seenEntities[item.EntityID] = true
		items = append(items, item)
	}

	// Include entities from Meta that had 0 impressions/spend in this date range
	// (e.g. newly created campaigns, adsets, ads, in-review, or paused entities)
	for entityID, entity := range entities {
		if seenEntities[entityID] {
			continue
		}
		if entity.EffectiveStatus == "DELETED" || entity.Status == "DELETED" || entity.EffectiveStatus == "ARCHIVED" || entity.Status == "ARCHIVED" {
			continue
		}
		campID := entity.CampaignID
		campName := entity.Campaign.Name
		adSetID := entity.AdSetID
		adSetName := entity.AdSet.Name
		if level == "campaign" {
			campID = entity.ID
			campName = entity.Name
		} else if level == "adset" {
			adSetID = entity.ID
			adSetName = entity.Name
		}
		item := domain.MetaInsightItem{
			EntityType:      level,
			EntityID:        entity.ID,
			EntityName:      entity.Name,
			Status:          entity.Status,
			EffectiveStatus: entity.EffectiveStatus,
			CampaignID:      campID,
			CampaignName:    campName,
			AdSetID:         adSetID,
			AdSetName:       adSetName,
			CreatedTime:     entity.CreatedTime,
			UpdatedTime:     entity.UpdatedTime,
		}
		if level == "ad" {
			item.AdID = entity.ID
			item.AdName = entity.Name
		}
		items = append(items, item)
	}

	trend := make([]domain.MetaInsightPoint, 0, len(daily))
	for _, row := range daily {
		result := selectedInsightResult(row.Actions, row.CostPerActionType)
		trend = append(trend, domain.MetaInsightPoint{Date: row.DateStart, Spend: numberPointer(row.Spend), Results: result.Value, Cost: result.Cost, LinkCTR: numberPointer(row.InlineLinkClickCTR), Frequency: numberPointer(row.Frequency)})
	}
	warnings := []string{"Recommendations are decision support, not automatic proof of profitability. Compare cost per result with your store's real break-even cost."}
	var summary *domain.MetaInsightItem
	if len(accountRows) > 0 {
		value := insightItem("account", accountRows[0])
		summary = &value
	} else {
		zero := 0.0
		summary = &domain.MetaInsightItem{
			EntityType:  "account",
			EntityName:  "Account",
			Spend:       &zero,
			Impressions: &zero,
			Reach:       &zero,
		}
	}
	report := &domain.MetaInsightsReport{Currency: account.Currency, Timezone: account.Timezone, Level: level, DateStart: since, DateStop: until, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Summary: summary, Items: items, Trend: trend, Warnings: warnings}

	g.mu.Lock()
	g.insightsCache[cacheKey] = insightsCacheEntry{
		report:    report,
		expiresAt: time.Now().Add(60 * time.Second),
	}
	g.mu.Unlock()

	return report, nil
}

type graphTargetingNamedItem struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type graphAdSetTargeting struct {
	AgeMin                  int                       `json:"age_min"`
	AgeMax                  int                       `json:"age_max"`
	Genders                 []int                     `json:"genders"`
	Interests               []graphTargetingNamedItem `json:"interests"`
	CustomAudiences         []graphTargetingNamedItem `json:"custom_audiences"`
	ExcludedCustomAudiences []graphTargetingNamedItem `json:"excluded_custom_audiences"`
	PublisherPlatforms      []string                  `json:"publisher_platforms"`
	DevicePlatforms         []string                  `json:"device_platforms"`
	GeoLocations            struct {
		Countries []string                  `json:"countries"`
		Regions   []graphTargetingNamedItem `json:"regions"`
		Cities    []graphTargetingNamedItem `json:"cities"`
	} `json:"geo_locations"`
}

func targetingNames(items []graphTargetingNamedItem) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item.Name)
		if value == "" {
			value = strings.TrimSpace(item.Key)
		}
		if value == "" {
			value = strings.TrimSpace(item.ID)
		}
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func audienceNumber(raw json.RawMessage) *float64 {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return &number
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return numberPointer(text)
	}
	return nil
}

func (g *GraphConnector) AudienceAnalysis(ctx context.Context, _ string, accountID, adAccountID, adSetID string) (*domain.MetaAudienceAnalysis, error) {
	token, err := g.userToken(ctx, accountID)
	if err != nil {
		return nil, err
	}
	normalizedAccountID := strings.TrimPrefix(adAccountID, "act_")
	var adSet struct {
		ID               string          `json:"id"`
		Name             string          `json:"name"`
		AccountID        string          `json:"account_id"`
		OptimizationGoal string          `json:"optimization_goal"`
		Targeting        json.RawMessage `json:"targeting"`
	}
	query := url.Values{"fields": {"id,name,account_id,optimization_goal,targeting"}, "access_token": {token}}
	if err = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(adSetID)+"?"+query.Encode(), &adSet); err != nil {
		return nil, err
	}
	if adSet.AccountID != "" && strings.TrimPrefix(adSet.AccountID, "act_") != normalizedAccountID {
		return nil, errors.New("ad set does not belong to selected ad account")
	}
	var rawTargeting graphAdSetTargeting
	if err = json.Unmarshal(adSet.Targeting, &rawTargeting); err != nil {
		return nil, fmt.Errorf("decode Meta Ad Set targeting: %w", err)
	}
	genders := make([]string, 0, len(rawTargeting.Genders))
	for _, gender := range rawTargeting.Genders {
		switch gender {
		case 1:
			genders = append(genders, "male")
		case 2:
			genders = append(genders, "female")
		default:
			genders = append(genders, strconv.Itoa(gender))
		}
	}
	analysis := &domain.MetaAudienceAnalysis{
		AdSetID: adSet.ID, AdSetName: adSet.Name, OptimizationGoal: adSet.OptimizationGoal,
		Targeting: domain.MetaAudienceTargeting{
			AgeMin: rawTargeting.AgeMin, AgeMax: rawTargeting.AgeMax, Genders: genders,
			Countries: rawTargeting.GeoLocations.Countries, Regions: targetingNames(rawTargeting.GeoLocations.Regions), Cities: targetingNames(rawTargeting.GeoLocations.Cities),
			Interests: targetingNames(rawTargeting.Interests), CustomAudiences: targetingNames(rawTargeting.CustomAudiences), ExcludedCustomAudiences: targetingNames(rawTargeting.ExcludedCustomAudiences),
			PublisherPlatforms: rawTargeting.PublisherPlatforms, DevicePlatforms: rawTargeting.DevicePlatforms,
		},
		Warnings: []string{"Audience size is an estimate and can change as Meta availability and targeting change."},
	}
	accountNodeID := "act_" + normalizedAccountID
	estimateQuery := url.Values{
		"targeting_spec":    {string(adSet.Targeting)},
		"optimization_goal": {adSet.OptimizationGoal},
		"access_token":      {token},
	}
	var estimate struct {
		Data []struct {
			EstimateDAU   json.RawMessage `json:"estimate_dau"`
			EstimateMAU   json.RawMessage `json:"estimate_mau"`
			EstimateReady bool            `json:"estimate_ready"`
		} `json:"data"`
	}
	if estimateErr := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(accountNodeID)+"/delivery_estimate?"+estimateQuery.Encode(), &estimate); estimateErr != nil {
		analysis.Warnings = append(analysis.Warnings, "Meta did not return an audience-size estimate for this Ad Set.")
		return analysis, nil
	}
	if len(estimate.Data) > 0 {
		analysis.EstimatedDailyUsers = audienceNumber(estimate.Data[0].EstimateDAU)
		analysis.EstimatedMonthlyUsers = audienceNumber(estimate.Data[0].EstimateMAU)
		analysis.EstimateReady = estimate.Data[0].EstimateReady
		analysis.EstimateSource = "meta_delivery_estimate"
	}
	return analysis, nil
}

func (g *GraphConnector) UpdateAdEntityStatus(ctx context.Context, _ string, accountID, entityID, status string) error {
	token, err := g.userToken(ctx, accountID)
	if err != nil {
		return err
	}
	values := url.Values{"status": {status}, "access_token": {token}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(entityID), strings.NewReader(values.Encode()))
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
		Success bool        `json:"success"`
		Error   *graphError `json:"error"`
	}
	if err = json.Unmarshal(body, &envelope); err != nil {
		return err
	}
	if resp.StatusCode >= 400 || envelope.Error != nil {
		return graphResponseError(resp.StatusCode, envelope.Error)
	}
	if !envelope.Success {
		return errors.New("Meta did not confirm the status change")
	}
	return nil
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
	if pageID == "" && accountID != "" {
		g.mu.RLock()
		if session := g.sessions[accountID]; session != nil {
			for pID, p := range session.Pages {
				if p.Connected && p.AccessToken != "" {
					pageID = pID
					break
				}
			}
		}
		g.mu.RUnlock()
		if pageID == "" && g.db != nil {
			var tokenModel database.MetaPageTokenModel
			if err := g.db.Where("account_id = ? AND is_connected = ?", accountID, true).First(&tokenModel).Error; err == nil && tokenModel.PageID != "" {
				pageID = tokenModel.PageID
			}
		}
	}

	_, page, found := g.findPage(pageID)
	if !found || page.AccessToken == "" {
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

func (g *GraphConnector) SendSenderAction(ctx context.Context, pageID, recipientID, action string) error {
	if pageID == "" {
		return nil
	}
	_, page, found := g.findPage(pageID)
	if !found || page.AccessToken == "" || page.Category == "WhatsApp" {
		return nil
	}
	payload, err := json.Marshal(map[string]any{
		"recipient":     map[string]string{"id": recipientID},
		"sender_action": action,
	})
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
	return nil
}

func (g *GraphConnector) SendMedia(ctx context.Context, accountID, pageID, recipientID, mediaType, mediaURL string) error {
	if pageID == "" && accountID != "" {
		g.mu.RLock()
		if session := g.sessions[accountID]; session != nil {
			for pID, p := range session.Pages {
				if p.Connected && p.AccessToken != "" {
					pageID = pID
					break
				}
			}
		}
		g.mu.RUnlock()
		if pageID == "" && g.db != nil {
			var tokenModel database.MetaPageTokenModel
			if err := g.db.Where("account_id = ? AND is_connected = ?", accountID, true).First(&tokenModel).Error; err == nil && tokenModel.PageID != "" {
				pageID = tokenModel.PageID
			}
		}
	}

	_, page, found := g.findPage(pageID)
	if !found || page.AccessToken == "" {
		return errors.New("meta page not found or not connected")
	}
	if page.Category == "WhatsApp" {
		return g.sendWhatsAppMedia(ctx, page.ID, page.AccessToken, recipientID, mediaType, mediaURL)
	}

	mType := strings.ToLower(strings.TrimSpace(mediaType))
	switch mType {
	case "voice", "audio", "void":
		mType = "audio"
	case "video", "vedio":
		mType = "video"
	case "image":
		mType = "image"
	default:
		mType = "file"
	}

	attachmentPayload := map[string]any{
		"url": mediaURL,
	}
	if mType == "image" {
		attachmentPayload["is_reusable"] = true
	}

	payloadMap := map[string]any{
		"recipient":      map[string]string{"id": recipientID},
		"messaging_type": "RESPONSE",
		"message": map[string]any{
			"attachment": map[string]any{
				"type":    mType,
				"payload": attachmentPayload,
			},
		},
	}

	payload, err := json.Marshal(payloadMap)
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
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

func (g *GraphConnector) SetProductRepo(r domain.ProductRepository) {
	g.productRepo = r
}
