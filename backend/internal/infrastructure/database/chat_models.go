package database

import "time"

type ChatMessageModel struct {
	ID        string       `gorm:"type:uuid;primaryKey"`
	AccountID string       `gorm:"type:uuid;index;not null"`
	Account   AccountModel `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	PageID    string       `gorm:"size:50;index;not null"`
	SenderID   string       `gorm:"size:50;index;not null"` // Facebook User ID or WhatsApp Number
	SenderName string       `gorm:"size:255"`
	SenderPic  string       `gorm:"type:text"`
	Message    string       `gorm:"type:text;not null"`
	Type       string       `gorm:"size:20;not null"` // text, image, video, audio, etc.
	Platform  string       `gorm:"size:20;not null"` // facebook, whatsapp, system (for outgoing)
	Timestamp string       `gorm:"size:20;not null"`
	CreatedAt time.Time    `gorm:"index"`
}
