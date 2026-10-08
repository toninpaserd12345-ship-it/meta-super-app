package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
)

func (g *GraphConnector) WhatsAppSignupConfig(_ context.Context) domain.MetaWhatsAppSignupConfig {
	return domain.MetaWhatsAppSignupConfig{
		AppID:    g.cfg.AppID,
		ConfigID: strings.TrimSpace(g.cfg.WhatsAppConfigID),
		Version:  g.cfg.Version,
		Enabled:  g.cfg.AppID != "" && strings.TrimSpace(g.cfg.WhatsAppConfigID) != "",
	}
}

// CompleteWhatsAppSignup exchanges the one-time Embedded Signup code,
// subscribes this app to the selected WABA, and stores each phone credential
// encrypted and scoped to the current workspace.
func (g *GraphConnector) CompleteWhatsAppSignup(ctx context.Context, accountID string, input domain.MetaWhatsAppSignupInput) ([]domain.MetaPage, error) {
	if strings.TrimSpace(g.cfg.WhatsAppConfigID) == "" {
		return nil, errors.New("WhatsApp Embedded Signup is not configured on the server")
	}
	if g.db == nil {
		return nil, errors.New("WhatsApp connections require database storage")
	}
	input.Code = strings.TrimSpace(input.Code)
	input.BusinessID = strings.TrimSpace(input.BusinessID)
	input.WABAID = strings.TrimSpace(input.WABAID)
	input.PhoneNumberID = strings.TrimSpace(input.PhoneNumberID)
	if input.Code == "" || input.WABAID == "" {
		return nil, errors.New("WhatsApp signup did not return a code and WABA ID")
	}

	query := url.Values{
		"client_id":     {g.cfg.AppID},
		"client_secret": {g.cfg.AppSecret},
		"code":          {input.Code},
	}
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/oauth/access_token?"+query.Encode(), &tokenResponse); err != nil {
		return nil, fmt.Errorf("exchange WhatsApp signup code: %w", err)
	}
	if tokenResponse.AccessToken == "" {
		return nil, errors.New("Meta returned an empty WhatsApp access token")
	}
	encryptedToken := g.encryptToken(tokenResponse.AccessToken)
	if encryptedToken == "" {
		return nil, errors.New("unable to encrypt WhatsApp access token")
	}

	if err := g.subscribeWhatsAppBusinessAccount(ctx, input.WABAID, tokenResponse.AccessToken); err != nil {
		return nil, fmt.Errorf("subscribe WhatsApp webhook: %w", err)
	}

	phones, err := g.whatsAppPhoneNumbers(ctx, input.WABAID, tokenResponse.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("load WhatsApp phone numbers: %w", err)
	}
	if len(phones) == 0 {
		return nil, errors.New("the selected WhatsApp account has no accessible phone number")
	}

	pages := make([]domain.MetaPage, 0, len(phones))
	for _, phone := range phones {
		if input.PhoneNumberID != "" && phone.ID != input.PhoneNumberID {
			continue
		}
		pictureURL := g.whatsAppProfilePicture(ctx, phone.ID, tokenResponse.AccessToken)
		model := database.MetaWhatsAppConnectionModel{
			PhoneNumberID:      phone.ID,
			AccountID:          accountID,
			BusinessID:         input.BusinessID,
			WABAID:             input.WABAID,
			DisplayPhoneNumber: phone.DisplayPhoneNumber,
			VerifiedName:       phone.VerifiedName,
			PictureURL:         pictureURL,
			AccessToken:        encryptedToken,
			IsConnected:        true,
		}
		if err := g.db.Save(&model).Error; err != nil {
			return nil, fmt.Errorf("save WhatsApp connection: %w", err)
		}
		page := graphPage{MetaPage: domain.MetaPage{
			ID:            phone.ID,
			Name:          firstNonEmpty(phone.VerifiedName, phone.DisplayPhoneNumber, "WhatsApp"),
			Category:      "WhatsApp",
			PictureURL:    pictureURL,
			PhoneNumber:   phone.DisplayPhoneNumber,
			TokenReady:    true,
			Connected:     true,
			WebhookStatus: "subscribed",
		}, AccessToken: tokenResponse.AccessToken}
		g.cachePage(accountID, page)
		pages = append(pages, page.MetaPage)
	}
	if len(pages) == 0 {
		return nil, errors.New("the selected WhatsApp phone number was not returned by Meta")
	}
	return pages, nil
}

func (g *GraphConnector) subscribeWhatsAppBusinessAccount(ctx context.Context, wabaID, accessToken string) error {
	form := url.Values{"access_token": {accessToken}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(wabaID)+"/subscribed_apps", strings.NewReader(form.Encode()))
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
	if resp.StatusCode >= http.StatusBadRequest || envelope.Error != nil || !envelope.Success {
		return graphResponseError(resp.StatusCode, envelope.Error)
	}
	return nil
}

type whatsAppPhone struct {
	ID                 string `json:"id"`
	DisplayPhoneNumber string `json:"display_phone_number"`
	VerifiedName       string `json:"verified_name"`
}

func (g *GraphConnector) whatsAppPhoneNumbers(ctx context.Context, wabaID, accessToken string) ([]whatsAppPhone, error) {
	query := url.Values{"fields": {"id,display_phone_number,verified_name"}, "access_token": {accessToken}}
	var response struct {
		Data []whatsAppPhone `json:"data"`
	}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(wabaID)+"/phone_numbers?"+query.Encode(), &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (g *GraphConnector) whatsAppProfilePicture(ctx context.Context, phoneID, accessToken string) string {
	query := url.Values{"fields": {"profile_picture_url"}, "access_token": {accessToken}}
	var response struct {
		Data []struct {
			PictureURL string `json:"profile_picture_url"`
		} `json:"data"`
	}
	if err := g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/"+url.PathEscape(phoneID)+"/whatsapp_business_profile?"+query.Encode(), &response); err == nil && len(response.Data) > 0 {
		return response.Data[0].PictureURL
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (g *GraphConnector) cachePage(accountID string, page graphPage) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.sessions[accountID] == nil {
		g.sessions[accountID] = &graphSession{Pages: make(map[string]graphPage)}
	}
	if g.sessions[accountID].Pages == nil {
		g.sessions[accountID].Pages = make(map[string]graphPage)
	}
	g.sessions[accountID].Pages[page.ID] = page
}

func (g *GraphConnector) persistedWhatsAppPages(accountID string) (map[string]graphPage, error) {
	result := make(map[string]graphPage)
	if g.db == nil {
		return result, nil
	}
	var models []database.MetaWhatsAppConnectionModel
	if err := g.db.Where("account_id = ?", accountID).Find(&models).Error; err != nil {
		return nil, err
	}
	for _, model := range models {
		token := g.decryptToken(model.AccessToken)
		result[model.PhoneNumberID] = graphPage{MetaPage: domain.MetaPage{
			ID:            model.PhoneNumberID,
			Name:          firstNonEmpty(model.VerifiedName, model.DisplayPhoneNumber, "WhatsApp"),
			Category:      "WhatsApp",
			PictureURL:    model.PictureURL,
			PhoneNumber:   model.DisplayPhoneNumber,
			TokenReady:    token != "",
			Connected:     model.IsConnected,
			WebhookStatus: map[bool]string{true: "subscribed", false: "disconnected"}[model.IsConnected],
		}, AccessToken: token}
	}
	return result, nil
}
