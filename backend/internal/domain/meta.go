package domain

import "context"

type MetaPage struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Category            string   `json:"category"`
	PictureURL          string   `json:"pictureUrl,omitempty"`
	TokenReady          bool     `json:"tokenReady"`
	Connected           bool     `json:"connected"`
	WebhookStatus       string   `json:"webhookStatus"`
	TokenExpiresAt      int64    `json:"tokenExpiresAt,omitempty"`
	DataAccessExpiresAt int64    `json:"dataAccessExpiresAt,omitempty"`
	GrantedPermissions  []string `json:"grantedPermissions,omitempty"`
}

type MetaOAuthResult struct {
	UserID    string
	AccountID string
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
}

type ProductBinding struct {
	PageID      string      `json:"pageId"`
	SourceID    string      `json:"sourceId"`
	SourceType  string      `json:"sourceType"`
	ProductID   string      `json:"productId"`
	FlowID      string      `json:"flowId"`
	ProductName string      `json:"productName"`
	Price       string      `json:"price"`
	Description string      `json:"description"`
	ImageUrl    string `json:"imageUrl"`

	Messages    []ReplyStep `json:"messages,omitempty"`
	Keywords    []string    `json:"keywords,omitempty"`
}

type Product struct {
	ID          string `json:"id"`
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
	AuthorizationURL(context.Context, string, string, []string) (string, error)
	CompleteAuthorization(context.Context, string, string) (*MetaOAuthResult, error)
	ListPages(context.Context, string, string) ([]MetaPage, error)
	ConnectPage(context.Context, string, string, string) (*MetaPage, error)
	DisconnectPage(context.Context, string, string, string) (*MetaPage, error)
	ListPosts(context.Context, string, string, string) ([]MetaPost, error)
	ListAdAccounts(context.Context, string, string) ([]MetaAdAccount, error)
	ListCampaigns(context.Context, string, string, string) ([]MetaCampaign, error)
	ListAds(context.Context, string, string, string) ([]MetaAd, error)
	ListProductBindings(context.Context, string, string) ([]ProductBinding, error)
	SaveProductBinding(context.Context, string, string, ProductBinding) (*ProductBinding, error)
	SaveProductBindings(context.Context, string, string, []ProductBinding) ([]ProductBinding, error)
	ListProducts(context.Context, string, string) ([]Product, error)
	SaveProduct(context.Context, string, string, Product) (*Product, error)
	ListReplyFlows(context.Context, string, string) ([]ReplyFlow, error)
	SaveReplyFlow(context.Context, string, string, ReplyFlow) (*ReplyFlow, error)
	VerifyWebhook(string) bool
	ReceiveWebhook(context.Context, []byte, string) error
	SetAutomationProvider(AutomationProvider)
}
