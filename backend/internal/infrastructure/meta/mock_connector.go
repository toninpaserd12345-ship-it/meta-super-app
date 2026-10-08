package meta

import (
	"context"
	"errors"
	"github.com/meta-super-app/backend/internal/domain"
	"sync"
)

func (m *MockConnector) AuthorizationURL(_ context.Context, _, _ string, _ []string) (string, error) {
	return "http://127.0.0.1:3001/meta-pages?meta=mock", nil
}
func (m *MockConnector) CompleteAuthorization(_ context.Context, _, _ string) (*domain.MetaOAuthResult, error) {
	return &domain.MetaOAuthResult{}, nil
}
func (m *MockConnector) PagePicture(_ context.Context, _, _ string) (*domain.MetaPagePicture, error) {
	return nil, errors.New("mock page picture not available")
}
func (m *MockConnector) VerifyWebhook(token string) bool                            { return token == "mock" }
func (m *MockConnector) ReceiveWebhook(_ context.Context, _ []byte, _ string) error { return nil }

type MockConnector struct {
	mu         sync.RWMutex
	connected  map[string]map[string]bool
	bindings   map[string]map[string]domain.ProductBinding
	products   map[string]map[string]domain.Product
	replyFlows map[string]map[string]domain.ReplyFlow
	automation domain.AutomationProvider
}

func (m *MockConnector) SetAutomationProvider(p domain.AutomationProvider) {
	m.automation = p
}

func NewMockConnector() *MockConnector {
	return &MockConnector{connected: make(map[string]map[string]bool), bindings: make(map[string]map[string]domain.ProductBinding), products: make(map[string]map[string]domain.Product), replyFlows: make(map[string]map[string]domain.ReplyFlow)}
}
func (m *MockConnector) ListReplyFlows(_ context.Context, _, accountID string) ([]domain.ReplyFlow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := []domain.ReplyFlow{}
	for _, item := range m.replyFlows[accountID] {
		result = append(result, item)
	}
	return result, nil
}
func (m *MockConnector) SaveReplyFlow(_ context.Context, _, accountID string, flow domain.ReplyFlow) (*domain.ReplyFlow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if flow.ID == "" {
		flow.ID = flow.Name
	}
	if m.replyFlows[accountID] == nil {
		m.replyFlows[accountID] = map[string]domain.ReplyFlow{}
	}
	m.replyFlows[accountID][flow.ID] = flow
	copy := flow
	return &copy, nil
}
func (m *MockConnector) ListPosts(_ context.Context, _, _, pageID string) ([]domain.MetaPost, error) {
	return []domain.MetaPost{{ID: pageID + "_post_1", Message: "New product available", CreatedTime: "2026-09-11T00:00:00+0000", PermalinkURL: "https://facebook.com/" + pageID}}, nil
}
func (m *MockConnector) ListAdAccounts(_ context.Context, _, _ string) ([]domain.MetaAdAccount, error) {
	return []domain.MetaAdAccount{{ID: "act_personal", Name: "Personal Demo", AccountStatus: 1, AccountType: "personal"}, {ID: "act_business", Name: "Business Demo", AccountStatus: 1, AccountType: "business", BusinessID: "business_1", BusinessName: "Demo Portfolio"}}, nil
}
func (m *MockConnector) ListCampaigns(_ context.Context, _, _, adAccountID string) ([]domain.MetaCampaign, error) {
	return []domain.MetaCampaign{{ID: adAccountID + "_campaign", Name: "Demo Messages Campaign", Status: "ACTIVE", EffectiveStatus: "ACTIVE", Objective: "OUTCOME_ENGAGEMENT", AdSetCount: 2, AdCount: 6}}, nil
}
func (m *MockConnector) ListAds(_ context.Context, _, _, campaignID string) ([]domain.MetaAd, error) {
	return []domain.MetaAd{{ID: campaignID + "_ad", Name: "Demo Click-to-Messenger Ad", Status: "ACTIVE", EffectiveStatus: "ACTIVE"}}, nil
}
func (m *MockConnector) ListProductBindings(_ context.Context, _, accountID string) ([]domain.ProductBinding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := []domain.ProductBinding{}
	for _, item := range m.bindings[accountID] {
		result = append(result, item)
	}
	return result, nil
}
func (m *MockConnector) SaveProductBinding(_ context.Context, _, accountID string, binding domain.ProductBinding) (*domain.ProductBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.bindings[accountID] == nil {
		m.bindings[accountID] = map[string]domain.ProductBinding{}
	}
	flow, flowFound := m.replyFlows[accountID][binding.FlowID]
	if !flowFound {
		return nil, errors.New("reply flow not found")
	}
	if binding.ProductID != "" {
		product, productFound := m.products[accountID][binding.ProductID]
		if !productFound {
			return nil, errors.New("product not found")
		}
		binding.ProductName, binding.Price, binding.Description = product.Name, product.Price, product.Description
	}
	binding.Messages = append([]domain.ReplyStep(nil), flow.Messages...)
	binding.Keywords = append([]string(nil), flow.Keywords...)
	m.bindings[accountID][binding.PageID+":"+binding.SourceType+":"+binding.SourceID] = binding
	copy := binding
	return &copy, nil
}
func (m *MockConnector) SaveProductBindings(ctx context.Context, userID, accountID string, bindings []domain.ProductBinding) ([]domain.ProductBinding, error) {
	result := make([]domain.ProductBinding, 0, len(bindings))
	for _, binding := range bindings {
		item, err := m.SaveProductBinding(ctx, userID, accountID, binding)
		if err != nil {
			return nil, err
		}
		result = append(result, *item)
	}
	return result, nil
}
func (m *MockConnector) ListPages(_ context.Context, _ string, accountID string) ([]domain.MetaPage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pages := seedPages()
	for i := range pages {
		pages[i].Connected = m.connected[accountID][pages[i].ID]
		if pages[i].Connected {
			pages[i].WebhookStatus = "subscribed"
		}
	}
	return pages, nil
}
func (m *MockConnector) ConnectPage(_ context.Context, _ string, accountID, pageID string) (*domain.MetaPage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, page := range seedPages() {
		if page.ID == pageID {
			if m.connected[accountID] == nil {
				m.connected[accountID] = make(map[string]bool)
			}
			m.connected[accountID][pageID] = true
			page.Connected = true
			page.WebhookStatus = "subscribed"
			return &page, nil
		}
	}
	return nil, errors.New("meta page not found")
}
func (m *MockConnector) DisconnectPage(_ context.Context, _ string, accountID, pageID string) (*domain.MetaPage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, page := range seedPages() {
		if page.ID == pageID {
			delete(m.connected[accountID], pageID)
			page.Connected = false
			page.WebhookStatus = "disconnected"
			return &page, nil
		}
	}
	return nil, errors.New("meta page not found")
}
func seedPages() []domain.MetaPage {
	return []domain.MetaPage{{ID: "page_1001", Name: "Vientiane Coffee", Category: "Cafe", TokenReady: true}, {ID: "page_1002", Name: "Lao Local Market", Category: "Shopping & retail", TokenReady: true}, {ID: "page_1003", Name: "Mekong Creative", Category: "Digital creator", TokenReady: true}}
}
