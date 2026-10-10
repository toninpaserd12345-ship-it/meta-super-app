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
	"time"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
)

type LeadInput struct{ AccountID, PageID, SenderID, SourceType, SourceID, Name string }
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
	var existing database.LeadModel
	if err := u.db.WithContext(ctx).Where("account_id=? AND page_id=? AND sender_id=?", in.AccountID, in.PageID, in.SenderID).First(&existing).Error; err == nil {
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
	lead := database.LeadModel{ID: uuid.NewString(), AccountID: in.AccountID, PageID: in.PageID, SenderID: in.SenderID, SourceType: in.SourceType, SourceID: in.SourceID, Name: in.Name, Status: "new", AssignedUserID: assigned, AlertRead: !settings.AlertsEnabled, FollowUpEnabled: settings.FollowUpEnabled}
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
		go u.sendCAPI(context.Background(), lead)
	}
	return nil
}

func (u *Leads) List(ctx context.Context, accountID string) ([]database.LeadModel, error) {
	var rows []database.LeadModel
	err := u.db.WithContext(ctx).Where("account_id=?", accountID).Order("created_at desc").Find(&rows).Error
	return rows, err
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

func (u *Leads) sendCAPI(ctx context.Context, lead database.LeadModel) {
	eventID := "lead-" + lead.ID
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
	payload := map[string]any{"data": []any{map[string]any{"event_name": "Lead", "event_time": time.Now().Unix(), "event_id": eventID, "action_source": "business_messaging", "user_data": map[string]any{"external_id": []string{hex.EncodeToString(sum[:])}}, "custom_data": map[string]any{"source_type": lead.SourceType, "source_id": lead.SourceID}}}}
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

func (u *Leads) UpdateStatus(ctx context.Context, accountID, senderID, status string) error {
	var lead database.LeadModel
	if err := u.db.WithContext(ctx).Where("account_id=? AND sender_id=?", accountID, senderID).First(&lead).Error; err != nil {
		return err
	}
	if lead.Status == status {
		return nil
	}
	
	if err := u.db.WithContext(ctx).Model(&lead).Update("status", status).Error; err != nil {
		return err
	}
	
	if status == "purchased" {
		go u.sendCAPI(context.Background(), lead)
	}
	return nil
}
