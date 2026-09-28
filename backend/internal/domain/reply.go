package domain

import (
	"context"
	"time"
)

type ReplySet struct {
	ID        string
	AccountID string
	Name      string
	Items     []ReplyItem
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReplyItem struct {
	ID         string
	ReplySetID string
	Type       string // "text", "image", "video", "audio"
	Content    string
	OrderIndex int
}

type ReplyRepository interface {
	CreateSet(ctx context.Context, set *ReplySet) error
	GetSetsByAccountID(ctx context.Context, accountID string) ([]ReplySet, error)
	GetSetByID(ctx context.Context, setID string) (*ReplySet, error)
	UpdateSet(ctx context.Context, set *ReplySet) error
	DeleteSet(ctx context.Context, setID string) error
	
	UpdateItems(ctx context.Context, setID string, items []ReplyItem) error
}
