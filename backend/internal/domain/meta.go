package domain

import (
	"context"
	"errors"
)

var ErrMetaReconnectRequired = errors.New("facebook authorization expired; reconnect Facebook")

type MetaPage struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Category            string   `json:"category"`
	PictureURL          string   `json:"pictureUrl,omitempty"`
	PhoneNumber         string   `json:"phoneNumber,omitempty"`
	TokenReady          bool     `json:"tokenReady"`
	Connected           bool     `json:"connected"`
	WebhookStatus       string   `json:"webhookStatus"`
	TokenExpiresAt      int64    `json:"tokenExpiresAt,omitempty"`
	DataAccessExpiresAt int64    `json:"dataAccessExpiresAt,omitempty"`
	GrantedPermissions  []string `json:"grantedPermissions,omitempty"`
}

type MetaPagePicture struct {
	ContentType string
	Data        []byte
}

type MetaWhatsAppDiagnostics struct {
	State                        string   `json:"state"`
	Message                      string   `json:"message"`
	RequiredPermissions          []string `json:"requiredPermissions"`
	GrantedPermissions           []string `json:"grantedPermissions,omitempty"`
	MissingPermissions           []string `json:"missingPermissions,omitempty"`
	Issues                       []string `json:"issues,omitempty"`
	BusinessCount                int      `json:"businessCount"`
	WhatsAppBusinessAccountCount int      `json:"whatsAppBusinessAccountCount"`
	PhoneNumberCount             int      `json:"phoneNumberCount"`
}

type MetaWhatsAppSignupConfig struct {
	AppID    string `json:"appId"`
	ConfigID string `json:"configId"`
	Version  string `json:"version"`
	Enabled  bool   `json:"enabled"`
}

type MetaWhatsAppSignupInput struct {
	Code          string `json:"code"`
	BusinessID    string `json:"businessId"`
	WABAID        string `json:"wabaId"`
	PhoneNumberID string `json:"phoneNumberId"`
}

type MetaOAuthResult struct {
	UserID      string
	AccountID   string
	AccessToken string
	FacebookID  string
	Name        string
	Email       string
}

type MetaPost struct {
	ID           string `json:"id"`
	Message      string `json:"message"`
	CreatedTime  string `json:"createdTime"`
	PermalinkURL string `json:"permalinkUrl"`
	PictureURL   string `json:"pictureUrl,omitempty"`
}

type MetaAdAccount struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AccountStatus int    `json:"accountStatus"`
	AccountType   string `json:"accountType"`
	BusinessID    string `json:"businessId,omitempty"`
	BusinessName  string `json:"businessName,omitempty"`
}
type MetaCampaign struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	EffectiveStatus string `json:"effectiveStatus"`
	Objective       string `json:"objective"`
	AdSetCount      int    `json:"adSetCount"`
	AdCount         int    `json:"adCount"`
}
type MetaAd struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	EffectiveStatus string `json:"effectiveStatus"`
	ThumbnailURL    string `json:"thumbnailUrl,omitempty"`
	ImageURL        string `json:"imageUrl,omitempty"`
}

type ProductBinding struct {
	PageID      string `json:"pageId"`
	SourceID    string `json:"sourceId"`
	SourceType  string `json:"sourceType"`
	ProductID   string `json:"productId"`
	FlowID      string `json:"flowId"`
	ProductName string `json:"productName"`
	Price       string `json:"price"`
	Description string `json:"description"`
	ImageUrl    string `json:"imageUrl"`

	Messages []ReplyStep `json:"messages,omitempty"`
	Keywords []string    `json:"keywords,omitempty"`
}

type Product struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Price       string `json:"price"`
	Description string `json:"description"`
	ImageUrl    string `json:"imageUrl"`
}

type ReplyFlow struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Keywords []string    `json:"keywords"`
	Messages []ReplyStep `json:"messages"`
}

type ReplyStep struct {
	Code    string `json:"code"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Enabled bool   `json:"enabled"`
}

type AutomationProvider interface {
	FindActiveRuleForTrigger(ctx context.Context, pageID, triggerType, triggerValue string) (*AutomationRule, error)
	GetKeywordRules(ctx context.Context, pageID string) ([]AutomationRule, error)
	GetReplySetItems(ctx context.Context, replySetID string, accountID string) ([]ReplyItem, error)
}

type MetaConnector interface {
	SendMessage(ctx context.Context, accountID, pageID, recipientID, text string) error
	AuthorizationURL(context.Context, string, string, []string) (string, error)
	CompleteAuthorization(context.Context, string, string) (*MetaOAuthResult, error)
	ListPages(context.Context, string, string) ([]MetaPage, error)
	WhatsAppDiagnostics(context.Context, string) (*MetaWhatsAppDiagnostics, error)
	WhatsAppSignupConfig(context.Context) MetaWhatsAppSignupConfig
	CompleteWhatsAppSignup(context.Context, string, MetaWhatsAppSignupInput) ([]MetaPage, error)
	PagePicture(context.Context, string, string) (*MetaPagePicture, error)
	ConnectPage(context.Context, string, string, string) (*MetaPage, error)
	DisconnectPage(context.Context, string, string, string) (*MetaPage, error)
	ListPosts(context.Context, string, string, string) ([]MetaPost, error)
	ListAdAccounts(context.Context, string, string) ([]MetaAdAccount, error)
	ListCampaigns(context.Context, string, string, string) ([]MetaCampaign, error)
	ListAds(context.Context, string, string, string) ([]MetaAd, error)
	VerifyWebhook(string) bool
	ReceiveWebhook(context.Context, []byte, string) error
	SetAutomationProvider(AutomationProvider)
}
