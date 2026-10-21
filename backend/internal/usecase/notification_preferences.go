package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidNotificationPreference = errors.New("invalid notification preference")

type NotificationPreference struct {
	Type      string `json:"type"`
	Frequency string `json:"frequency"`
	Enabled   bool   `json:"enabled"`
}

type NotificationPreferences struct{ db *gorm.DB }

func NewNotificationPreferences(db *gorm.DB) *NotificationPreferences {
	return &NotificationPreferences{db: db}
}

var notificationDefaults = []NotificationPreference{
	{Type: "new_lead", Frequency: "instant", Enabled: true},
	{Type: "order_updates", Frequency: "instant", Enabled: true},
	{Type: "ad_performance", Frequency: "daily", Enabled: true},
	{Type: "connection_health", Frequency: "instant", Enabled: true},
	{Type: "weekly_summary", Frequency: "weekly", Enabled: true},
}

var notificationFrequencies = map[string]map[string]bool{
	"new_lead":          {"instant": true, "hourly": true, "daily": true},
	"order_updates":     {"instant": true, "daily": true},
	"ad_performance":    {"instant": true, "daily": true, "weekly": true},
	"connection_health": {"instant": true, "daily": true},
	"weekly_summary":    {"weekly": true},
}

func (u *NotificationPreferences) List(ctx context.Context, accountID, userID string) ([]NotificationPreference, error) {
	var rows []database.NotificationPreferenceModel
	if err := u.db.WithContext(ctx).Where("account_id = ? AND user_id = ?", accountID, userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	stored := make(map[string]database.NotificationPreferenceModel, len(rows))
	for _, row := range rows {
		stored[row.Type] = row
	}
	result := make([]NotificationPreference, 0, len(notificationDefaults))
	for _, fallback := range notificationDefaults {
		if row, ok := stored[fallback.Type]; ok {
			result = append(result, NotificationPreference{Type: row.Type, Frequency: row.Frequency, Enabled: row.Enabled})
			continue
		}
		result = append(result, fallback)
	}
	return result, nil
}

func (u *NotificationPreferences) Save(ctx context.Context, accountID, userID string, preferences []NotificationPreference) ([]NotificationPreference, error) {
	for _, preference := range preferences {
		allowed, ok := notificationFrequencies[preference.Type]
		if !ok || !allowed[preference.Frequency] {
			return nil, ErrInvalidNotificationPreference
		}
		row := database.NotificationPreferenceModel{
			ID: uuid.NewString(), AccountID: accountID, UserID: userID,
			Type: preference.Type, Frequency: preference.Frequency, Enabled: preference.Enabled,
		}
		if err := u.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "account_id"}, {Name: "user_id"}, {Name: "type"}},
			DoUpdates: clause.AssignmentColumns([]string{"frequency", "enabled", "updated_at"}),
		}).Create(&row).Error; err != nil {
			return nil, err
		}
	}
	return u.List(ctx, accountID, userID)
}
