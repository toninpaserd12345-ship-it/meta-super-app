package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"github.com/meta-super-app/backend/internal/usecase"
	"gorm.io/gorm"
)

type GormChat struct {
	db *gorm.DB
}

func NewGormChat(db *gorm.DB) *GormChat {
	return &GormChat{db: db}
}

func (r *GormChat) SaveMessage(ctx context.Context, msg usecase.ChatEvent) error {
	id := uuid.NewString()
	model := database.ChatMessageModel{
		ID:         id,
		AccountID:  msg.AccountID,
		PageID:     msg.PageID,
		SenderID:   msg.SenderID,
		SenderName: msg.SenderName,
		SenderPic:  msg.SenderPic,
		Message:    msg.Message,
		Type:       msg.Type,
		Platform:   msg.Platform,
		Timestamp:  msg.Timestamp,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *GormChat) GetRecentMessages(ctx context.Context, accountID string, limit int) ([]usecase.ChatEvent, error) {
	var models []database.ChatMessageModel
	err := r.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Order("created_at DESC").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	// Reverse them so oldest comes first
	events := make([]usecase.ChatEvent, len(models))
	for i, m := range models {
		events[len(models)-1-i] = usecase.ChatEvent{
			AccountID:  m.AccountID,
			PageID:     m.PageID,
			SenderID:   m.SenderID,
			SenderName: m.SenderName,
			SenderPic:  m.SenderPic,
			Message:    m.Message,
			Type:       m.Type,
			Platform:   m.Platform,
			Timestamp:  m.Timestamp,
		}
	}
	return events, nil
}
