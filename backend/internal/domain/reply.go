package domain

import (
	"context"
	"time"
)

type ReplySet struct {
	ID        string      `json:"id"`
	AccountID string      `json:"accountId"`
	Name      string      `json:"name"`
	Items     []ReplyItem `json:"items"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

type ReplyItem struct {
	ID         string `json:"id"`
	ReplySetID string `json:"replySetId"`
	Type       string `json:"type"` // "text", "image", "video", "audio"
	Content    string `json:"content"`
	OrderIndex int    `json:"orderIndex"`
	IsEnabled  bool   `json:"isEnabled"`
}

type ReplyRepository interface {
	CreateSet(ctx context.Context, set *ReplySet) error
	GetSetsByAccountID(ctx context.Context, accountID string) ([]ReplySet, error)
	GetSetByID(ctx context.Context, setID string) (*ReplySet, error)
	UpdateSet(ctx context.Context, set *ReplySet) error
	DeleteSet(ctx context.Context, setID string) error

	UpdateItems(ctx context.Context, setID string, items []ReplyItem) error
}
