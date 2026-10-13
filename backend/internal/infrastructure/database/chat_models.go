package database

import "time"

type ChatMessageModel struct {
	SeqID      uint64    `gorm:"autoIncrement;index:idx_chat_seq;not null"` // Auto-incrementing BigInt for high-performance sequential B-Tree ordering & cursor pagination
	ID         string    `gorm:"type:uuid;primaryKey"`
	MID        string    `gorm:"size:120;index:idx_chat_mid"` // Meta Message ID for deduplication
	AccountID  string    `gorm:"type:uuid;index:idx_chat_acc_created,priority:1;not null"`
	PageID     string    `gorm:"size:50;index:idx_chat_thread,priority:1;not null"`
	SenderID   string    `gorm:"size:50;index:idx_chat_thread,priority:2;not null"` // Facebook User ID or WhatsApp Number
	SenderName string    `gorm:"size:255"`
	SenderPic  string    `gorm:"type:text"`
	Message    string    `gorm:"type:text;not null"`
	Type       string    `gorm:"size:20;not null"` // text, image, video, audio, etc.
	Platform   string    `gorm:"size:20;not null"` // facebook, whatsapp, system (for outgoing)
	Timestamp  string    `gorm:"size:20;not null"`
	CreatedAt  time.Time `gorm:"index:idx_chat_acc_created,priority:2;index:idx_chat_thread,priority:3"`
}
