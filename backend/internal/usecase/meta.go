package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/meta-super-app/backend/internal/domain"
)

var ErrPageNotFound = errors.New("meta page not found")

type Meta struct {
	connector domain.MetaConnector
}

func NewMeta(connector domain.MetaConnector) *Meta {
	return &Meta{connector: connector}
}
func (u *Meta) AuthorizationURL(ctx context.Context, userID, accountID string, permissions []string) (string, error) {
	return u.connector.AuthorizationURL(ctx, userID, accountID, permissions)
}
func (u *Meta) CompleteAuthorization(ctx context.Context, state, code string) (*domain.MetaOAuthResult, error) {
	if state == "" || code == "" {
		return nil, errors.New("invalid oauth callback")
	}
	result, err := u.connector.CompleteAuthorization(ctx, state, code)
	return result, err
}
func (u *Meta) AttachAccount(ctx context.Context, accountID, accessToken string) error {
	if accountID == "" || accessToken == "" {
		return errors.New("accountID and accessToken are required")
	}
	return u.connector.AttachAccount(ctx, accountID, accessToken)
}
func (u *Meta) ListPages(ctx context.Context, userID, accountID string) ([]domain.MetaPage, error) {
	return u.connector.ListPages(ctx, userID, accountID)
}
func (u *Meta) WhatsAppDiagnostics(ctx context.Context, accountID string) (*domain.MetaWhatsAppDiagnostics, error) {
	return u.connector.WhatsAppDiagnostics(ctx, accountID)
}
func (u *Meta) WhatsAppSignupConfig(ctx context.Context) domain.MetaWhatsAppSignupConfig {
	return u.connector.WhatsAppSignupConfig(ctx)
}
func (u *Meta) CompleteWhatsAppSignup(ctx context.Context, accountID string, input domain.MetaWhatsAppSignupInput) ([]domain.MetaPage, error) {
	if accountID == "" || input.Code == "" || input.WABAID == "" {
		return nil, errors.New("WhatsApp signup code and business account are required")
	}
	items, err := u.connector.CompleteWhatsAppSignup(ctx, accountID, input)
	return items, err
}
func (u *Meta) PagePicture(ctx context.Context, accountID, pageID string) (*domain.MetaPagePicture, error) {
	if pageID == "" {
		return nil, ErrPageNotFound
	}
	return u.connector.PagePicture(ctx, accountID, pageID)
}
func (u *Meta) ConnectPage(ctx context.Context, userID, accountID, pageID string) (*domain.MetaPage, error) {
	if pageID == "" {
		return nil, ErrPageNotFound
	}
	page, err := u.connector.ConnectPage(ctx, userID, accountID, pageID)
	return page, err
}
func (u *Meta) DisconnectPage(ctx context.Context, userID, accountID, pageID string) (*domain.MetaPage, error) {
	if pageID == "" {
		return nil, ErrPageNotFound
	}
	page, err := u.connector.DisconnectPage(ctx, userID, accountID, pageID)
	return page, err
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
func (u *Meta) AdsInsights(ctx context.Context, userID, accountID, adAccountID, level, since, until string) (*domain.MetaInsightsReport, error) {
	if adAccountID == "" {
		return nil, errors.New("ad account is required")
	}
	level = strings.ToLower(strings.TrimSpace(level))
	if level != "campaign" && level != "adset" && level != "ad" {
		return nil, errors.New("insights level must be campaign, adset, or ad")
	}
	start, err := time.Parse("2006-01-02", since)
	if err != nil {
		return nil, errors.New("invalid insights start date")
	}
	stop, err := time.Parse("2006-01-02", until)
	if err != nil || stop.Before(start) || stop.Sub(start) > 366*24*time.Hour {
		return nil, errors.New("invalid insights date range")
	}
	return u.connector.AdsInsights(ctx, userID, accountID, adAccountID, level, since, until)
}
func (u *Meta) AudienceAnalysis(ctx context.Context, userID, accountID, adAccountID, adSetID string) (*domain.MetaAudienceAnalysis, error) {
	if strings.TrimSpace(adAccountID) == "" || strings.TrimSpace(adSetID) == "" {
		return nil, errors.New("ad account and ad set are required")
	}
	return u.connector.AudienceAnalysis(ctx, userID, accountID, adAccountID, adSetID)
}
func (u *Meta) UpdateAdEntityStatus(ctx context.Context, userID, accountID, entityType, entityID, status string) error {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	status = strings.ToUpper(strings.TrimSpace(status))
	if entityType != "campaign" && entityType != "adset" && entityType != "ad" {
		return errors.New("unsupported ad entity type")
	}
	if entityID == "" || (status != "ACTIVE" && status != "PAUSED") {
		return errors.New("invalid ad status update")
	}
	return u.connector.UpdateAdEntityStatus(ctx, userID, accountID, entityID, status)
}
func (u *Meta) UpdateAdEntityBudget(ctx context.Context, userID, accountID, entityType, entityID string, dailyBudget int64) error {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	if entityType != "campaign" && entityType != "adset" {
		return errors.New("budget update only supported for campaign and adset")
	}
	if entityID == "" || dailyBudget <= 0 {
		return errors.New("invalid ad budget update")
	}
	return u.connector.UpdateAdEntityBudget(ctx, userID, accountID, entityID, dailyBudget)
}
func (u *Meta) VerifyWebhook(token string) bool { return u.connector.VerifyWebhook(token) }
func (u *Meta) ReceiveWebhook(ctx context.Context, body []byte, signature string) error {
	return u.connector.ReceiveWebhook(ctx, body, signature)
}

func (u *Meta) SendMessage(ctx context.Context, accountID, pageID, recipientID, text string) error {
	return u.connector.SendMessage(ctx, accountID, pageID, recipientID, text)
}

func (u *Meta) SendSenderAction(ctx context.Context, pageID, recipientID, action string) error {
	return u.connector.SendSenderAction(ctx, pageID, recipientID, action)
}

func (u *Meta) SendMedia(ctx context.Context, accountID, pageID, recipientID, mediaType, mediaURL string) error {
	return u.connector.SendMedia(ctx, accountID, pageID, recipientID, mediaType, mediaURL)
}
