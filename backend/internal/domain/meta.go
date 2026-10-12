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

type MetaInsightResult struct {
	Type  string   `json:"type"`
	Value *float64 `json:"value,omitempty"`
	Cost  *float64 `json:"cost,omitempty"`
}

type MetaInsightItem struct {
	EntityType            string            `json:"entityType"`
	EntityID              string            `json:"entityId"`
	EntityName            string            `json:"entityName"`
	CampaignID            string            `json:"campaignId,omitempty"`
	CampaignName          string            `json:"campaignName,omitempty"`
	AdSetID               string            `json:"adSetId,omitempty"`
	AdSetName             string            `json:"adSetName,omitempty"`
	AdID                  string            `json:"adId,omitempty"`
	AdName                string            `json:"adName,omitempty"`
	Spend                 *float64          `json:"spend,omitempty"`
	Impressions           *float64          `json:"impressions,omitempty"`
	Reach                 *float64          `json:"reach,omitempty"`
	Frequency             *float64          `json:"frequency,omitempty"`
	LinkClicks            *float64          `json:"linkClicks,omitempty"`
	LinkCTR               *float64          `json:"linkCtr,omitempty"`
	LinkCPC               *float64          `json:"linkCpc,omitempty"`
	CPM                   *float64          `json:"cpm,omitempty"`
	Results               MetaInsightResult `json:"results"`
	QualityRanking        string            `json:"qualityRanking,omitempty"`
	EngagementRateRanking string            `json:"engagementRateRanking,omitempty"`
	ConversionRateRanking string            `json:"conversionRateRanking,omitempty"`
}

type MetaInsightPoint struct {
	Date      string   `json:"date"`
	Spend     *float64 `json:"spend,omitempty"`
	Results   *float64 `json:"results,omitempty"`
	Cost      *float64 `json:"costPerResult,omitempty"`
	LinkCTR   *float64 `json:"linkCtr,omitempty"`
	Frequency *float64 `json:"frequency,omitempty"`
}

type MetaInsightsReport struct {
	Currency    string             `json:"currency"`
	Timezone    string             `json:"timezone"`
	Level       string             `json:"level"`
	DateStart   string             `json:"dateStart"`
	DateStop    string             `json:"dateStop"`
	GeneratedAt string             `json:"generatedAt"`
	Summary     *MetaInsightItem   `json:"summary,omitempty"`
	Items       []MetaInsightItem  `json:"items"`
	Trend       []MetaInsightPoint `json:"trend"`
	Warnings    []string           `json:"warnings"`
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
	FindReplySetByTrigger(ctx context.Context, accountID, trigger string) (*ReplySet, error)
	GetReplySetItems(ctx context.Context, replySetID string, accountID string) ([]ReplyItem, error)
}

type LeadCapture struct{ AccountID, PageID, SenderID, SourceType, SourceID, Name string }
type LeadSink interface {
	CaptureLead(context.Context, LeadCapture) error
}

type MetaConnector interface {
	SendMessage(ctx context.Context, accountID, pageID, recipientID, text string) error
	SendSenderAction(ctx context.Context, pageID, recipientID, action string) error
	SendMedia(ctx context.Context, accountID, pageID, recipientID, mediaType, mediaURL string) error
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
	AdsInsights(context.Context, string, string, string, string, string, string) (*MetaInsightsReport, error)
	UpdateAdEntityStatus(context.Context, string, string, string, string) error
	VerifyWebhook(string) bool
	ReceiveWebhook(context.Context, []byte, string) error
	SetAutomationProvider(AutomationProvider)
	SetLeadSink(LeadSink)
}
