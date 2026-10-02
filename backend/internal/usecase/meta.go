package usecase

import (
	"context"
	"errors"
	"github.com/meta-super-app/backend/internal/domain"
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
func (u *Meta) VerifyWebhook(token string) bool { return u.connector.VerifyWebhook(token) }
func (u *Meta) ReceiveWebhook(ctx context.Context, body []byte, signature string) error {
	return u.connector.ReceiveWebhook(ctx, body, signature)
}
