package usecase

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/meta-super-app/backend/internal/domain"
	"golang.org/x/sync/singleflight"
)

var ErrPageNotFound = errors.New("meta page not found")

type metaCacheEntry struct {
	value     any
	expiresAt time.Time
}

// Meta keeps short-lived, account-scoped copies of expensive Graph API lists.
// OAuth tokens are never used as cache keys or stored in cache values.
type Meta struct {
	connector domain.MetaConnector
	cacheMu   sync.RWMutex
	cache     map[string]metaCacheEntry
	requests  singleflight.Group
}

func NewMeta(connector domain.MetaConnector) *Meta {
	return &Meta{connector: connector, cache: make(map[string]metaCacheEntry)}
}

func cacheGet[T any](u *Meta, key string) (T, bool) {
	var zero T
	u.cacheMu.RLock()
	entry, ok := u.cache[key]
	u.cacheMu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			u.cacheMu.Lock()
			delete(u.cache, key)
			u.cacheMu.Unlock()
		}
		return zero, false
	}
	value, ok := entry.value.(T)
	return value, ok
}

func cachePut[T any](u *Meta, key string, value T, ttl time.Duration) {
	u.cacheMu.Lock()
	u.cache[key] = metaCacheEntry{value: value, expiresAt: time.Now().Add(ttl)}
	u.cacheMu.Unlock()
}

func cacheLoad[T any](u *Meta, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	if value, ok := cacheGet[T](u, key); ok {
		return value, nil
	}
	loaded, err, _ := u.requests.Do(key, func() (any, error) {
		if value, ok := cacheGet[T](u, key); ok {
			return value, nil
		}
		value, loadErr := loader()
		if loadErr == nil {
			cachePut(u, key, value, ttl)
		}
		return value, loadErr
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return loaded.(T), nil
}

func (u *Meta) invalidateCache(prefix string) {
	u.cacheMu.Lock()
	for key := range u.cache {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			delete(u.cache, key)
		}
	}
	u.cacheMu.Unlock()
}
func (u *Meta) AuthorizationURL(ctx context.Context, userID, accountID string, permissions []string) (string, error) {
	return u.connector.AuthorizationURL(ctx, userID, accountID, permissions)
}
func (u *Meta) CompleteAuthorization(ctx context.Context, state, code string) (*domain.MetaOAuthResult, error) {
	if state == "" || code == "" {
		return nil, errors.New("invalid oauth callback")
	}
	result, err := u.connector.CompleteAuthorization(ctx, state, code)
	if err == nil && result != nil {
		u.invalidateCache("account:" + result.AccountID + ":")
	}
	return result, err
}
func (u *Meta) ListPages(ctx context.Context, userID, accountID string) ([]domain.MetaPage, error) {
	key := "account:" + accountID + ":pages"
	items, err := cacheLoad(u, key, 30*time.Second, func() ([]domain.MetaPage, error) {
		items, loadErr := u.connector.ListPages(ctx, userID, accountID)
		return append([]domain.MetaPage(nil), items...), loadErr
	})
	items = append([]domain.MetaPage(nil), items...)
	return items, err
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
	if err == nil {
		u.invalidateCache("account:" + accountID + ":")
	}
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
	if err == nil {
		u.invalidateCache("account:" + accountID + ":")
	}
	return page, err
}
func (u *Meta) DisconnectPage(ctx context.Context, userID, accountID, pageID string) (*domain.MetaPage, error) {
	if pageID == "" {
		return nil, ErrPageNotFound
	}
	page, err := u.connector.DisconnectPage(ctx, userID, accountID, pageID)
	if err == nil {
		u.invalidateCache("account:" + accountID + ":")
	}
	return page, err
}
func (u *Meta) ListPosts(ctx context.Context, userID, accountID, pageID string) ([]domain.MetaPost, error) {
	if pageID == "" {
		return nil, ErrPageNotFound
	}
	key := "account:" + accountID + ":posts:" + pageID
	items, err := cacheLoad(u, key, 2*time.Minute, func() ([]domain.MetaPost, error) {
		items, loadErr := u.connector.ListPosts(ctx, userID, accountID, pageID)
		return append([]domain.MetaPost(nil), items...), loadErr
	})
	items = append([]domain.MetaPost(nil), items...)
	return items, err
}
func (u *Meta) ListAdAccounts(ctx context.Context, userID, accountID string) ([]domain.MetaAdAccount, error) {
	key := "account:" + accountID + ":ad-accounts"
	items, err := cacheLoad(u, key, 5*time.Minute, func() ([]domain.MetaAdAccount, error) {
		items, loadErr := u.connector.ListAdAccounts(ctx, userID, accountID)
		return append([]domain.MetaAdAccount(nil), items...), loadErr
	})
	items = append([]domain.MetaAdAccount(nil), items...)
	return items, err
}
func (u *Meta) ListCampaigns(ctx context.Context, userID, accountID, adAccountID string) ([]domain.MetaCampaign, error) {
	if adAccountID == "" {
		return nil, errors.New("ad account is required")
	}
	key := "account:" + accountID + ":campaigns:" + adAccountID
	items, err := cacheLoad(u, key, 2*time.Minute, func() ([]domain.MetaCampaign, error) {
		items, loadErr := u.connector.ListCampaigns(ctx, userID, accountID, adAccountID)
		return append([]domain.MetaCampaign(nil), items...), loadErr
	})
	items = append([]domain.MetaCampaign(nil), items...)
	return items, err
}
func (u *Meta) ListAds(ctx context.Context, userID, accountID, campaignID string) ([]domain.MetaAd, error) {
	if campaignID == "" {
		return nil, errors.New("campaign is required")
	}
	key := "account:" + accountID + ":ads:" + campaignID
	items, err := cacheLoad(u, key, 2*time.Minute, func() ([]domain.MetaAd, error) {
		items, loadErr := u.connector.ListAds(ctx, userID, accountID, campaignID)
		return append([]domain.MetaAd(nil), items...), loadErr
	})
	items = append([]domain.MetaAd(nil), items...)
	return items, err
}
func (u *Meta) VerifyWebhook(token string) bool { return u.connector.VerifyWebhook(token) }
func (u *Meta) ReceiveWebhook(ctx context.Context, body []byte, signature string) error {
	return u.connector.ReceiveWebhook(ctx, body, signature)
}

func (u *Meta) SendMessage(ctx context.Context, accountID, pageID, recipientID, text string) error {
	return u.connector.SendMessage(ctx, accountID, pageID, recipientID, text)
}
