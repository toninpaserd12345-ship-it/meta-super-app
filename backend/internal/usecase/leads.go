package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
)

type LeadInput struct{ AccountID, PageID, SenderID, SourceType, SourceID, ProductID, Name string }
type LeadDetailsInput struct {
	Name           *string `json:"name"`
	Phone          *string `json:"phone"`
	Address        *string `json:"address"`
	Notes          *string `json:"notes"`
	Tags           *string `json:"tags"`
	Status         *string `json:"status"`
	AssignedUserID *string `json:"assignedUserId"`
}
type LeadSettingsInput struct {
	AssignmentMode                              string `json:"assignmentMode"`
	AlertsEnabled, FollowUpEnabled, CAPIEnabled bool
	CAPIDatasetID                               string `json:"capiDatasetId"`
	CAPIToken                                   string `json:"capiToken"`
}
type FollowUpStepInput struct {
	DelayMinutes int    `json:"delayMinutes"`
	Message      string `json:"message"`
	IsEnabled    bool   `json:"isEnabled"`
}
type CAPIHealth struct {
	State         string     `json:"state"`
	Enabled       bool       `json:"enabled"`
	Configured    bool       `json:"configured"`
	Sent          int64      `json:"sent"`
	Failed        int64      `json:"failed"`
	Pending       int64      `json:"pending"`
	NotConfigured int64      `json:"notConfigured"`
	SuccessRate   *float64   `json:"successRate,omitempty"`
	LastEventAt   *time.Time `json:"lastEventAt,omitempty"`
	LastError     string     `json:"lastError,omitempty"`
}
type LeadMessenger interface {
	SendMessage(context.Context, string, string, string, string) error
}

type Leads struct {
	chatStream                     *ChatStream
	db                             *gorm.DB
	messenger                      LeadMessenger
	datasetID, token, graphVersion string
}

func NewLeads(db *gorm.DB, messenger LeadMessenger, chatStream *ChatStream, graphVersion string) *Leads {
	return &Leads{db: db, messenger: messenger, chatStream: chatStream, datasetID: os.Getenv("META_CAPI_DATASET_ID"), token: os.Getenv("META_CAPI_ACCESS_TOKEN"), graphVersion: graphVersion}
}
func (u *Leads) CaptureLead(ctx context.Context, inDomain domain.LeadCapture) error {
	return u.capture(ctx, LeadInput(inDomain))
}

func (u *Leads) capture(ctx context.Context, in LeadInput) error {
	if in.AccountID == "" || in.PageID == "" || in.SenderID == "" {
		return nil
	}
	if _, err := uuid.Parse(in.AccountID); err != nil {
		var realAccount database.AccountModel
		if err := u.db.WithContext(ctx).First(&realAccount).Error; err == nil {
			in.AccountID = realAccount.ID
		} else {
			return nil
		}
	}
	var existing database.LeadModel
	if err := u.db.WithContext(ctx).Where("account_id=? AND page_id=? AND sender_id=?", in.AccountID, in.PageID, in.SenderID).First(&existing).Error; err == nil {
		updates := map[string]interface{}{
			"updated_at": time.Now(),
		}
		if (existing.SourceID == "" || existing.SourceType == "message") && in.SourceID != "" && in.SourceType != "message" {
			updates["source_type"] = in.SourceType
			updates["source_id"] = in.SourceID
		}
		if existing.ProductID == "" && in.ProductID != "" {
			updates["product_id"] = in.ProductID
		}
		if in.Name != "" && existing.Name == "" {
			updates["name"] = in.Name
		}
		_ = u.db.WithContext(ctx).Model(&existing).Updates(updates).Error
		return nil
	}
	var settings database.LeadSettingsModel
	_ = u.db.WithContext(ctx).FirstOrCreate(&settings, database.LeadSettingsModel{AccountID: in.AccountID, AssignmentMode: "round_robin", AlertsEnabled: true}).Error
	assigned := ""
	if settings.AssignmentMode == "round_robin" {
		var members []database.MembershipModel
		_ = u.db.WithContext(ctx).Where("account_id=?", in.AccountID).Order("created_at asc").Find(&members).Error
		if len(members) > 0 {
			var count int64
			u.db.WithContext(ctx).Model(&database.LeadModel{}).Where("account_id=?", in.AccountID).Count(&count)
			assigned = members[int(count)%len(members)].UserID
		}
	}
	lead := database.LeadModel{ID: uuid.NewString(), AccountID: in.AccountID, PageID: in.PageID, SenderID: in.SenderID, SourceType: in.SourceType, SourceID: in.SourceID, ProductID: in.ProductID, Name: in.Name, Status: "new", AssignedUserID: assigned, AlertRead: !settings.AlertsEnabled, FollowUpEnabled: settings.FollowUpEnabled}
	if settings.FollowUpEnabled {
		var step database.LeadFollowUpStepModel
		if u.db.WithContext(ctx).Where("account_id=? AND is_enabled=?", in.AccountID, true).Order("step_index asc").First(&step).Error == nil {
			t := time.Now().Add(time.Duration(step.DelayMinutes) * time.Minute)
			lead.NextFollowUpAt = &t
		}
	}
	if err := u.db.WithContext(ctx).Create(&lead).Error; err != nil {
		return err
	}
	if settings.AlertsEnabled && u.chatStream != nil {
		u.chatStream.Broadcast(in.AccountID, ChatEvent{
			AccountID: in.AccountID,
			PageID:    in.PageID,
			SenderID:  in.SenderID,
			Message:   "New Lead!",
			Type:      "lead_alert",
			Timestamp: fmt.Sprintf("%d", time.Now().UnixMilli()),
			Platform:  "system",
		})
	}
	if settings.CAPIEnabled {
		go u.sendCAPI(context.Background(), lead, "Lead")
	}
	return nil
}

func (u *Leads) List(ctx context.Context, accountID string) ([]database.LeadModel, error) {
	var rows []database.LeadModel
	err := u.db.WithContext(ctx).Where("account_id=?", accountID).Order("created_at desc").Find(&rows).Error
	return rows, err
}

func (u *Leads) GetLeadBySender(ctx context.Context, accountID, senderID string) (*database.LeadModel, error) {
	var lead database.LeadModel
	err := u.db.WithContext(ctx).Where("account_id=? AND sender_id=?", accountID, senderID).Order("updated_at desc").First(&lead).Error
	if err != nil {
		return nil, err
	}
	return &lead, nil
}
func (u *Leads) Settings(ctx context.Context, accountID string) (database.LeadSettingsModel, []database.LeadFollowUpStepModel, error) {
	var s database.LeadSettingsModel
	err := u.db.WithContext(ctx).FirstOrCreate(&s, database.LeadSettingsModel{AccountID: accountID, AssignmentMode: "round_robin", AlertsEnabled: true}).Error
	var steps []database.LeadFollowUpStepModel
	if err == nil {
		err = u.db.WithContext(ctx).Where("account_id=?", accountID).Order("step_index asc").Find(&steps).Error
	}
	return s, steps, err
}
func (u *Leads) SaveSettings(ctx context.Context, accountID string, in LeadSettingsInput, steps []FollowUpStepInput) error {
	if in.AssignmentMode == "" {
		in.AssignmentMode = "round_robin"
	}
	var s database.LeadSettingsModel
	u.db.WithContext(ctx).FirstOrCreate(&s, database.LeadSettingsModel{AccountID: accountID, AssignmentMode: "round_robin", AlertsEnabled: true})

	s.AssignmentMode = in.AssignmentMode
	s.AlertsEnabled = in.AlertsEnabled
	s.FollowUpEnabled = in.FollowUpEnabled
	s.CAPIEnabled = in.CAPIEnabled
	s.CAPIDatasetID = in.CAPIDatasetID
	if in.CAPIToken != "" && in.CAPIToken != "********" {
		s.CAPIToken = in.CAPIToken
	}

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&s).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id=?", accountID).Delete(&database.LeadFollowUpStepModel{}).Error; err != nil {
			return err
		}
		for i, x := range steps {
			if x.Message == "" {
				continue
			}
			row := database.LeadFollowUpStepModel{ID: uuid.NewString(), AccountID: accountID, StepIndex: i, DelayMinutes: x.DelayMinutes, Message: x.Message, IsEnabled: x.IsEnabled}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (u *Leads) MarkRead(ctx context.Context, accountID, id string) error {
	return u.db.WithContext(ctx).Model(&database.LeadModel{}).Where("id=? AND account_id=?", id, accountID).Update("alert_read", true).Error
}

func (u *Leads) CAPIHealth(ctx context.Context, accountID string) (CAPIHealth, error) {
	var settings database.LeadSettingsModel
	err := u.db.WithContext(ctx).FirstOrCreate(&settings, database.LeadSettingsModel{AccountID: accountID, AssignmentMode: "round_robin", AlertsEnabled: true}).Error
	if err != nil {
		return CAPIHealth{}, err
	}
	health := CAPIHealth{Enabled: settings.CAPIEnabled, Configured: (settings.CAPIDatasetID != "" && settings.CAPIToken != "") || (u.datasetID != "" && u.token != "")}
	rows := []struct {
		Status string
		Count  int64
	}{}
	if err = u.db.WithContext(ctx).Model(&database.CAPIEventModel{}).Select("status, count(*) as count").Where("account_id = ? AND created_at >= ?", accountID, time.Now().Add(-7*24*time.Hour)).Group("status").Scan(&rows).Error; err != nil {
		return CAPIHealth{}, err
	}
	for _, row := range rows {
		switch row.Status {
		case "sent":
			health.Sent = row.Count
		case "failed":
			health.Failed = row.Count
		case "pending":
			health.Pending = row.Count
		case "not_configured":
			health.NotConfigured = row.Count
		}
	}
	var latest database.CAPIEventModel
	if latestErr := u.db.WithContext(ctx).Where("account_id = ?", accountID).Order("created_at desc").First(&latest).Error; latestErr == nil {
		health.LastEventAt = &latest.CreatedAt
		health.LastError = latest.Error
	}
	completed := health.Sent + health.Failed
	if completed > 0 {
		rate := float64(health.Sent) / float64(completed) * 100
		health.SuccessRate = &rate
	}
	switch {
	case !health.Enabled:
		health.State = "disabled"
	case !health.Configured:
		health.State = "setup_required"
	case health.Failed > 0 && (health.SuccessRate == nil || *health.SuccessRate < 90):
		health.State = "failing"
	case health.Failed > 0 || health.Pending > 0:
		health.State = "warning"
	case health.Sent == 0:
		health.State = "no_events"
	default:
		health.State = "healthy"
	}
	return health, nil
}

func (u *Leads) ProcessFollowUps(ctx context.Context) {
	var leads []database.LeadModel
	u.db.WithContext(ctx).Where("follow_up_enabled=? AND next_follow_up_at<=?", true, time.Now()).Find(&leads)
	for _, lead := range leads {
		var steps []database.LeadFollowUpStepModel
		u.db.WithContext(ctx).Where("account_id=? AND is_enabled=?", lead.AccountID, true).Order("step_index asc").Find(&steps)
		sort.Slice(steps, func(i, j int) bool { return steps[i].StepIndex < steps[j].StepIndex })
		if lead.FollowUpStep >= len(steps) {
			u.db.Model(&lead).Updates(map[string]any{"follow_up_enabled": false, "next_follow_up_at": nil})
			continue
		}
		step := steps[lead.FollowUpStep]
		if u.messenger.SendMessage(ctx, lead.AccountID, lead.PageID, lead.SenderID, step.Message) != nil {
			continue
		}
		next := lead.FollowUpStep + 1
		updates := map[string]any{"follow_up_step": next}
		if next < len(steps) {
			t := time.Now().Add(time.Duration(steps[next].DelayMinutes) * time.Minute)
			updates["next_follow_up_at"] = &t
		} else {
			updates["follow_up_enabled"] = false
			updates["next_follow_up_at"] = nil
		}
		u.db.Model(&lead).Updates(updates)
	}
}

func (u *Leads) sendCAPI(ctx context.Context, lead database.LeadModel, eventName string) {
	eventID := strings.ToLower(eventName) + "-" + lead.ID
	row := database.CAPIEventModel{EventID: eventID, AccountID: lead.AccountID, LeadID: lead.ID, Status: "pending"}
	if u.db.Create(&row).Error != nil {
		return
	}
	var settings database.LeadSettingsModel
	if u.db.Where("account_id = ?", lead.AccountID).First(&settings).Error != nil {
		u.db.Model(&row).Updates(map[string]any{"status": "not_configured", "error": "LeadSettings not found"})
		return
	}
	datasetID := settings.CAPIDatasetID
	token := settings.CAPIToken
	if datasetID == "" || token == "" {
		datasetID, token = u.datasetID, u.token // fallback to env
	}
	if datasetID == "" || token == "" {
		u.db.Model(&row).Updates(map[string]any{"status": "not_configured", "error": "CAPI Dataset ID or Token is missing"})
		return
	}
	sum := sha256.Sum256([]byte(lead.SenderID))
	customData := map[string]any{"source_type": lead.SourceType, "source_id": lead.SourceID}
	if lead.ProductID != "" {
		var product database.ProductModel
		if u.db.WithContext(ctx).Where("id = ? AND account_id = ?", lead.ProductID, lead.AccountID).First(&product).Error == nil {
			customData["content_ids"] = []string{product.ID}
			customData["content_name"] = product.Name
			if product.SalePrice != nil {
				customData["value"] = *product.SalePrice
				customData["currency"] = product.Currency
			}
		}
	}
	payload := map[string]any{"data": []any{map[string]any{"event_name": eventName, "event_time": time.Now().Unix(), "event_id": eventID, "action_source": "business_messaging", "user_data": map[string]any{"external_id": []string{hex.EncodeToString(sum[:])}}, "custom_data": customData}}}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/events?access_token=%s", u.graphVersion, datasetID, token)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		u.db.Model(&row).Updates(map[string]any{"status": "failed", "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	status := "sent"
	msg := ""
	if resp.StatusCode >= 300 {
		status = "failed"
		msg = resp.Status
	}
	u.db.Model(&row).Updates(map[string]any{"status": status, "error": msg})
}

func (u *Leads) UpdateLeadDetails(ctx context.Context, accountID, senderID string, in LeadDetailsInput) (*database.LeadModel, error) {
	var lead database.LeadModel
	err := u.db.WithContext(ctx).Where("account_id=? AND (sender_id=? OR id=?)", accountID, senderID, senderID).First(&lead).Error
	if err != nil {
		statusVal := "new"
		if in.Status != nil && *in.Status != "" {
			statusVal = *in.Status
		}
		lead = database.LeadModel{
			ID:         uuid.NewString(),
			AccountID:  accountID,
			SenderID:   senderID,
			Status:     statusVal,
			SourceType: "message",
			AlertRead:  true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if in.Name != nil {
			lead.Name = *in.Name
		}
		if in.Phone != nil {
			lead.Phone = *in.Phone
		}
		if in.Address != nil {
			lead.Address = *in.Address
		}
		if in.Notes != nil {
			lead.Notes = *in.Notes
		}
		if in.Tags != nil {
			lead.Tags = *in.Tags
		}
		if in.AssignedUserID != nil {
			lead.AssignedUserID = *in.AssignedUserID
		}
		if createErr := u.db.WithContext(ctx).Create(&lead).Error; createErr != nil {
			return nil, createErr
		}
	} else {
		updates := map[string]interface{}{
			"updated_at": time.Now(),
		}
		if in.Name != nil {
			updates["name"] = *in.Name
			lead.Name = *in.Name
		}
		if in.Phone != nil {
			updates["phone"] = *in.Phone
			lead.Phone = *in.Phone
		}
		if in.Address != nil {
			updates["address"] = *in.Address
			lead.Address = *in.Address
		}
		if in.Notes != nil {
			updates["notes"] = *in.Notes
			lead.Notes = *in.Notes
		}
		if in.Tags != nil {
			updates["tags"] = *in.Tags
			lead.Tags = *in.Tags
		}
		if in.AssignedUserID != nil {
			updates["assigned_user_id"] = *in.AssignedUserID
			lead.AssignedUserID = *in.AssignedUserID
		}
		if in.Status != nil && *in.Status != "" {
			updates["status"] = *in.Status
			lead.Status = *in.Status
		}
		if err := u.db.WithContext(ctx).Model(&lead).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	if lead.Status == "purchased" {
		go u.sendCAPI(context.Background(), lead, "Purchase")
	}

	if u.chatStream != nil {
		u.chatStream.Broadcast(accountID, ChatEvent{
			AccountID: accountID,
			PageID:    lead.PageID,
			SenderID:  lead.SenderID,
			Message:   fmt.Sprintf("lead_update:%s", lead.Status),
			Type:      "lead_alert",
			Timestamp: fmt.Sprintf("%d", time.Now().UnixMilli()),
			Platform:  "system",
		})
	}

	return &lead, nil
}

func (u *Leads) UpdateStatus(ctx context.Context, accountID, senderID, status string) error {
	_, err := u.UpdateLeadDetails(ctx, accountID, senderID, LeadDetailsInput{
		Status: &status,
	})
	return err
}
