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

type MetaWhatsAppDirectInput struct {
	AccessToken   string `json:"accessToken"`
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

type MetaPlacementMetric struct {
	PublisherPlatform string   `json:"publisherPlatform"`
	PlatformPosition  string   `json:"platformPosition"`
	Spend             *float64 `json:"spend,omitempty"`
	Impressions       *float64 `json:"impressions,omitempty"`
	Reach             *float64 `json:"reach,omitempty"`
	LinkClicks        *float64 `json:"linkClicks,omitempty"`
	MessagesCount     *float64 `json:"messagesCount,omitempty"`
	CostPerMessage    *float64 `json:"costPerMessage,omitempty"`
	PurchasesCount    *float64 `json:"purchasesCount,omitempty"`
	CostPerPurchase   *float64 `json:"costPerPurchase,omitempty"`
}

type MetaDemographicMetric struct {
	Key             string   `json:"key"`
	Label           string   `json:"label"`
	Category        string   `json:"category"`
	SubCategory     string   `json:"subCategory,omitempty"`
	Spend           *float64 `json:"spend,omitempty"`
	Impressions     *float64 `json:"impressions,omitempty"`
	Reach           *float64 `json:"reach,omitempty"`
	LinkClicks      *float64 `json:"linkClicks,omitempty"`
	MessagesCount   *float64 `json:"messagesCount,omitempty"`
	CostPerMessage  *float64 `json:"costPerMessage,omitempty"`
	PurchasesCount  *float64 `json:"purchasesCount,omitempty"`
	CostPerPurchase *float64 `json:"costPerPurchase,omitempty"`
}

type MetaVideoFunnelMetric struct {
	TotalImpressions *float64 `json:"totalImpressions,omitempty"`
	Views3s          *float64 `json:"views3s,omitempty"`
	Thruplay         *float64 `json:"thruplay,omitempty"`
	P25              *float64 `json:"p25,omitempty"`
	P50              *float64 `json:"p50,omitempty"`
	P75              *float64 `json:"p75,omitempty"`
	P100             *float64 `json:"p100,omitempty"`
	HookRate         *float64 `json:"hookRate,omitempty"`
	HoldRate         *float64 `json:"holdRate,omitempty"`
}

type MetaDemographicsReport struct {
	Currency    string                   `json:"currency"`
	Since       string                   `json:"since"`
	Until       string                   `json:"until"`
	Age         []MetaDemographicMetric  `json:"age"`
	Gender      []MetaDemographicMetric  `json:"gender"`
	AgeGender   []MetaDemographicMetric  `json:"ageGender"`
	Devices     []MetaDemographicMetric  `json:"devices"`
	Regions     []MetaDemographicMetric  `json:"regions"`
	Placements  []MetaPlacementMetric    `json:"placements"`
	Hourly      []MetaInsightHourlyPoint `json:"hourly,omitempty"`
	VideoFunnel *MetaVideoFunnelMetric   `json:"videoFunnel,omitempty"`
	Summary     struct {
		BestAge      string `json:"bestAge,omitempty"`
		BestGender   string `json:"bestGender,omitempty"`
		BestDevice   string `json:"bestDevice,omitempty"`
		BestRegion   string `json:"bestRegion,omitempty"`
		BestHours    string `json:"bestHours,omitempty"`
		WastedHours  string `json:"wastedHours,omitempty"`
		WastedTarget string `json:"wastedTarget,omitempty"`
	} `json:"summary"`
}

type MetaInsightItem struct {
	EntityType            string            `json:"entityType"`
	EntityID              string            `json:"entityId"`
	EntityName            string            `json:"entityName"`
	Status                string            `json:"status,omitempty"`
	EffectiveStatus       string            `json:"effectiveStatus,omitempty"`
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
	ConversionValue       *float64          `json:"conversionValue,omitempty"`
	ROAS                  *float64          `json:"roas,omitempty"`
	Results               MetaInsightResult `json:"results"`
	MessagesCount         *float64          `json:"messagesCount,omitempty"`
	CostPerMessage        *float64          `json:"costPerMessage,omitempty"`
	PurchasesCount        *float64          `json:"purchasesCount,omitempty"`
	CostPerPurchase       *float64          `json:"costPerPurchase,omitempty"`
	QualityRanking        string            `json:"qualityRanking,omitempty"`
	EngagementRateRanking string            `json:"engagementRateRanking,omitempty"`
	ConversionRateRanking string            `json:"conversionRateRanking,omitempty"`
	DailyBudget           *float64          `json:"dailyBudget,omitempty"`
	LifetimeBudget        *float64                   `json:"lifetimeBudget,omitempty"`
	CreatedTime           string                     `json:"createdTime,omitempty"`
	UpdatedTime           string                     `json:"updatedTime,omitempty"`
	ThumbnailURL          string                     `json:"thumbnailUrl,omitempty"`
	ImageURL              string                     `json:"imageUrl,omitempty"`
	CreativeTitle         string                     `json:"creativeTitle,omitempty"`
	CreativeBody          string                     `json:"creativeBody,omitempty"`
	VideoID                string                     `json:"videoId,omitempty"`
	EffectiveObjectStoryID string                     `json:"effectiveObjectStoryId,omitempty"`
	PermalinkURL          string                     `json:"permalinkUrl,omitempty"`
	VideoViews3s          *float64                   `json:"videoViews3s,omitempty"`
	VideoThruplay         *float64                   `json:"videoThruplay,omitempty"`
	VideoP25              *float64                   `json:"videoP25,omitempty"`
	VideoP50              *float64                   `json:"videoP50,omitempty"`
	VideoP75              *float64                   `json:"videoP75,omitempty"`
	VideoP100             *float64                   `json:"videoP100,omitempty"`
	HookRate              *float64                   `json:"hookRate,omitempty"`
	HoldRate              *float64                   `json:"holdRate,omitempty"`
	Placements            []MetaPlacementMetric      `json:"placements,omitempty"`
	Trend                 []MetaInsightPoint         `json:"trend,omitempty"`
	Hourly                []MetaInsightHourlyPoint   `json:"hourly,omitempty"`
}

type MetaInsightPoint struct {
	Date           string   `json:"date"`
	Spend          *float64 `json:"spend,omitempty"`
	Results        *float64 `json:"results,omitempty"`
	Cost           *float64 `json:"costPerResult,omitempty"`
	MessagesCount  *float64 `json:"messagesCount,omitempty"`
	CostPerMessage *float64 `json:"costPerMessage,omitempty"`
	LinkCTR        *float64 `json:"linkCtr,omitempty"`
	Frequency      *float64 `json:"frequency,omitempty"`
}

type MetaInsightHourlyPoint struct {
	Hour            string   `json:"hour"`
	Spend           *float64 `json:"spend,omitempty"`
	Results         *float64 `json:"results,omitempty"`
	Cost            *float64 `json:"costPerResult,omitempty"`
	MessagesCount   *float64 `json:"messagesCount,omitempty"`
	CostPerMessage  *float64 `json:"costPerMessage,omitempty"`
	PurchasesCount  *float64 `json:"purchasesCount,omitempty"`
	CostPerPurchase *float64 `json:"costPerPurchase,omitempty"`
	LinkCTR         *float64 `json:"linkCtr,omitempty"`
	Impressions     *float64 `json:"impressions,omitempty"`
}

type MetaInsightsReport struct {
	Currency    string                   `json:"currency"`
	Timezone    string                   `json:"timezone"`
	Level       string                   `json:"level"`
	DateStart   string                   `json:"dateStart"`
	DateStop    string                   `json:"dateStop"`
	GeneratedAt string                   `json:"generatedAt"`
	Summary     *MetaInsightItem         `json:"summary,omitempty"`
	Items       []MetaInsightItem        `json:"items"`
	Trend       []MetaInsightPoint       `json:"trend"`
	Hourly      []MetaInsightHourlyPoint `json:"hourly,omitempty"`
	Warnings    []string                 `json:"warnings"`
}

type MetaAudienceTargeting struct {
	AgeMin                  int      `json:"ageMin,omitempty"`
	AgeMax                  int      `json:"ageMax,omitempty"`
	Genders                 []string `json:"genders,omitempty"`
	Countries               []string `json:"countries,omitempty"`
	Regions                 []string `json:"regions,omitempty"`
	Cities                  []string `json:"cities,omitempty"`
	Interests               []string `json:"interests,omitempty"`
	CustomAudiences         []string `json:"customAudiences,omitempty"`
	ExcludedCustomAudiences []string `json:"excludedCustomAudiences,omitempty"`
	PublisherPlatforms      []string `json:"publisherPlatforms,omitempty"`
	DevicePlatforms         []string `json:"devicePlatforms,omitempty"`
}

type MetaAudienceAnalysis struct {
	AdSetID               string                `json:"adSetId"`
	AdSetName             string                `json:"adSetName"`
	OptimizationGoal      string                `json:"optimizationGoal,omitempty"`
	Targeting             MetaAudienceTargeting `json:"targeting"`
	EstimatedDailyUsers   *float64              `json:"estimatedDailyUsers,omitempty"`
	EstimatedMonthlyUsers *float64              `json:"estimatedMonthlyUsers,omitempty"`
	EstimateReady         bool                  `json:"estimateReady"`
	EstimateSource        string                `json:"estimateSource,omitempty"`
	Warnings              []string              `json:"warnings,omitempty"`
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
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Price       string   `json:"price"`
	SalePrice   *float64 `json:"salePrice,omitempty"`
	Cost        *float64 `json:"cost,omitempty"`
	Quantity    *int64   `json:"quantity,omitempty"`
	Currency    string   `json:"currency,omitempty"`
	Description string   `json:"description"`
	ImageUrl    string   `json:"imageUrl"`
	Barcode     string   `json:"barcode,omitempty"`
	Category    string   `json:"category,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Weight      *float64 `json:"weight,omitempty"`
	GalleryURLs string   `json:"galleryUrls,omitempty"`
	MinStock    *int64   `json:"minStock,omitempty"`
	Status      string   `json:"status,omitempty"`
	Colors            string   `json:"colors,omitempty"`
	Sizes             string   `json:"sizes,omitempty"`
	Variants          string   `json:"variants,omitempty"`
	PackagingCost     float64  `json:"packagingCost,omitempty"`
	ShippingCostStore float64  `json:"shippingCostStore,omitempty"`
	TargetProfit      float64  `json:"targetProfit,omitempty"`
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
	GetRule(ctx context.Context, id, accountID string) (*AutomationRule, error)
}

type LeadCapture struct{ AccountID, PageID, SenderID, SourceType, SourceID, ProductID, Name string }
type LeadSink interface {
	CaptureLead(context.Context, LeadCapture) error
}

type MetaConnector interface {
	SendMessage(ctx context.Context, accountID, pageID, recipientID, text string) error
	SendSenderAction(ctx context.Context, pageID, recipientID, action string) error
	SendMedia(ctx context.Context, accountID, pageID, recipientID, mediaType, mediaURL string) error
	AuthorizationURL(context.Context, string, string, []string) (string, error)
	CompleteAuthorization(context.Context, string, string) (*MetaOAuthResult, error)
	AttachAccount(context.Context, string, string) error
	ListPages(context.Context, string, string) ([]MetaPage, error)
	WhatsAppDiagnostics(context.Context, string) (*MetaWhatsAppDiagnostics, error)
	WhatsAppSignupConfig(context.Context) MetaWhatsAppSignupConfig
	CompleteWhatsAppSignup(context.Context, string, MetaWhatsAppSignupInput) ([]MetaPage, error)
	ConnectWhatsAppDirect(context.Context, string, MetaWhatsAppDirectInput) ([]MetaPage, error)
	PagePicture(context.Context, string, string) (*MetaPagePicture, error)
	ConnectPage(context.Context, string, string, string) (*MetaPage, error)
	DisconnectPage(context.Context, string, string, string) (*MetaPage, error)
	ListPosts(context.Context, string, string, string) ([]MetaPost, error)
	ListAdAccounts(context.Context, string, string) ([]MetaAdAccount, error)
	ListCampaigns(context.Context, string, string, string) ([]MetaCampaign, error)
	ListAds(context.Context, string, string, string) ([]MetaAd, error)
	AdsInsights(context.Context, string, string, string, string, string, string) (*MetaInsightsReport, error)
	Demographics(context.Context, string, string, string, string, string, string) (*MetaDemographicsReport, error)
	AudienceAnalysis(context.Context, string, string, string, string) (*MetaAudienceAnalysis, error)
	UpdateAdEntityStatus(context.Context, string, string, string, string) error
	UpdateAdEntityBudget(ctx context.Context, userID, accountID, entityID string, dailyBudget int64) error
	VerifyWebhook(string) bool
	ReceiveWebhook(context.Context, []byte, string) error
	SetAutomationProvider(AutomationProvider)
	SetLeadSink(LeadSink)
	CountConnectedPages(context.Context, string, ...string) (int, error)
}
