package usecase

import (
	"context"
	"errors"
	"github.com/meta-super-app/backend/internal/domain"
	"net/url"
	"strings"
)

var ErrPageNotFound = errors.New("meta page not found")

type Meta struct{ connector domain.MetaConnector }

func NewMeta(connector domain.MetaConnector) *Meta { return &Meta{connector: connector} }
func (u *Meta) AuthorizationURL(ctx context.Context, userID, accountID string, permissions []string) (string, error) {
	return u.connector.AuthorizationURL(ctx, userID, accountID, permissions)
}
func (u *Meta) CompleteAuthorization(ctx context.Context, state, code string) (*domain.MetaOAuthResult, error) {
	if state == "" || code == "" {
		return nil, errors.New("invalid oauth callback")
	}
	return u.connector.CompleteAuthorization(ctx, state, code)
}
func (u *Meta) ListPages(ctx context.Context, userID, accountID string) ([]domain.MetaPage, error) {
	return u.connector.ListPages(ctx, userID, accountID)
}
func (u *Meta) ConnectPage(ctx context.Context, userID, accountID, pageID string) (*domain.MetaPage, error) {
	if pageID == "" {
		return nil, ErrPageNotFound
	}
	return u.connector.ConnectPage(ctx, userID, accountID, pageID)
}
func (u *Meta) DisconnectPage(ctx context.Context, userID, accountID, pageID string) (*domain.MetaPage, error) {
	if pageID == "" {
		return nil, ErrPageNotFound
	}
	return u.connector.DisconnectPage(ctx, userID, accountID, pageID)
}
func (u *Meta) ListPosts(ctx context.Context, userID, accountID, pageID string) ([]domain.MetaPost, error) {
	if pageID == "" {
		return nil, ErrPageNotFound
	}
	return u.connector.ListPosts(ctx, userID, accountID, pageID)
}
func (u *Meta) ListAdAccounts(ctx context.Context, userID, accountID string) ([]domain.MetaAdAccount, error) {
	return u.connector.ListAdAccounts(ctx, userID, accountID)
}
func (u *Meta) ListCampaigns(ctx context.Context, userID, accountID, adAccountID string) ([]domain.MetaCampaign, error) {
	if adAccountID == "" {
		return nil, errors.New("ad account is required")
	}
	return u.connector.ListCampaigns(ctx, userID, accountID, adAccountID)
}
func (u *Meta) ListAds(ctx context.Context, userID, accountID, campaignID string) ([]domain.MetaAd, error) {
	if campaignID == "" {
		return nil, errors.New("campaign is required")
	}
	return u.connector.ListAds(ctx, userID, accountID, campaignID)
}
func (u *Meta) ListProductBindings(ctx context.Context, userID, accountID string) ([]domain.ProductBinding, error) {
	return u.connector.ListProductBindings(ctx, userID, accountID)
}
func (u *Meta) SaveProductBinding(ctx context.Context, userID, accountID string, binding domain.ProductBinding) (*domain.ProductBinding, error) {
	if binding.PageID == "" || binding.SourceID == "" || binding.ProductID == "" || binding.FlowID == "" {
		return nil, errors.New("page, source, product and reply flow are required")
	}
	if binding.SourceType != "post" && binding.SourceType != "ad" {
		return nil, errors.New("source type must be post or ad")
	}
	return u.connector.SaveProductBinding(ctx, userID, accountID, binding)
}
func (u *Meta) SaveProductBindings(ctx context.Context, userID, accountID string, bindings []domain.ProductBinding) ([]domain.ProductBinding, error) {
	if len(bindings) == 0 || len(bindings) > 1000 {
		return nil, errors.New("between 1 and 1000 automation targets are required")
	}
	for _, binding := range bindings {
		if binding.PageID == "" || binding.SourceID == "" || binding.ProductID == "" || binding.FlowID == "" {
			return nil, errors.New("every target requires page, source, product and reply flow")
		}
		if binding.SourceType != "post" && binding.SourceType != "ad" {
			return nil, errors.New("source type must be post or ad")
		}
	}
	return u.connector.SaveProductBindings(ctx, userID, accountID, bindings)
}
func (u *Meta) ListProducts(ctx context.Context, userID, accountID string) ([]domain.Product, error) {
	return u.connector.ListProducts(ctx, userID, accountID)
}
func (u *Meta) SaveProduct(ctx context.Context, userID, accountID string, product domain.Product) (*domain.Product, error) {
	product.Name, product.Price, product.Description = strings.TrimSpace(product.Name), strings.TrimSpace(product.Price), strings.TrimSpace(product.Description)
	if product.Name == "" || product.Price == "" {
		return nil, errors.New("product name and price are required")
	}
	if len(product.Name) > 120 || len(product.Price) > 60 || len(product.Description) > 1000 {
		return nil, errors.New("product fields exceed the allowed length")
	}
	return u.connector.SaveProduct(ctx, userID, accountID, product)
}
func (u *Meta) ListReplyFlows(ctx context.Context, userID, accountID string) ([]domain.ReplyFlow, error) {
	return u.connector.ListReplyFlows(ctx, userID, accountID)
}
func (u *Meta) SaveReplyFlow(ctx context.Context, userID, accountID string, flow domain.ReplyFlow) (*domain.ReplyFlow, error) {
	flow.Name = strings.TrimSpace(flow.Name)
	if flow.Name == "" || len(flow.Messages) == 0 {
		return nil, errors.New("flow name and at least one message are required")
	}
	if len(flow.Messages) > 20 {
		return nil, errors.New("a reply flow can contain at most 20 messages")
	}
	codes, enabled := map[string]bool{}, 0
	for index := range flow.Messages {
		message := &flow.Messages[index]
		message.Code, message.Content = strings.TrimSpace(message.Code), strings.TrimSpace(message.Content)
		if message.Code == "" || message.Content == "" {
			return nil, errors.New("message code and content are required")
		}
		if message.Type != "text" && message.Type != "image" && message.Type != "video" && message.Type != "audio" {
			return nil, errors.New("message type must be text, image, video or audio")
		}
		if codes[message.Code] {
			return nil, errors.New("message codes must be unique")
		}
		codes[message.Code] = true
		if message.Enabled {
			enabled++
		}
		if message.Type == "text" && len(message.Content) > 2000 {
			return nil, errors.New("text messages cannot exceed 2000 characters")
		}
		if message.Type != "text" {
			parsed, err := url.Parse(message.Content)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
				return nil, errors.New("media messages require a public HTTPS URL")
			}
		}
	}
	if enabled == 0 {
		return nil, errors.New("at least one message must be enabled")
	}
	return u.connector.SaveReplyFlow(ctx, userID, accountID, flow)
}
func (u *Meta) VerifyWebhook(token string) bool { return u.connector.VerifyWebhook(token) }
func (u *Meta) ReceiveWebhook(ctx context.Context, body []byte, signature string) error {
	return u.connector.ReceiveWebhook(ctx, body, signature)
}
