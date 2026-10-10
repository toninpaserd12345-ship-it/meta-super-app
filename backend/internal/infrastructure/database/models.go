package database

import "time"

type UserModel struct {
	ID           string  `gorm:"type:uuid;primaryKey"`
	Name         string  `gorm:"size:120;not null"`
	Email        string  `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string  `gorm:"size:255;not null"`
	FacebookID   *string `gorm:"size:100;uniqueIndex"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AccountModel struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	Name      string `gorm:"size:160;not null"`
	Slug      string `gorm:"size:100;uniqueIndex;not null"`
	LogoURL   string `gorm:"size:500"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MembershipModel struct {
	ID        string                 `gorm:"type:uuid;primaryKey"`
	UserID    string                 `gorm:"type:uuid;not null;uniqueIndex:idx_user_account"`
	User      UserModel              `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	AccountID string                 `gorm:"type:uuid;not null;uniqueIndex:idx_user_account"`
	Account   AccountModel           `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	Role      string                 `gorm:"size:30;not null"`
	Claims    []MembershipClaimModel `gorm:"foreignKey:MembershipID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MembershipClaimModel struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	MembershipID string `gorm:"type:uuid;not null;uniqueIndex:idx_membership_claim"`
	Claim        string `gorm:"size:100;not null;uniqueIndex:idx_membership_claim"`
}

type PlanModel struct {
	ID        string `gorm:"size:50;primaryKey"` // e.g., "free", "pro"
	Name      string `gorm:"size:100;not null"`
	Price     int    // Price in local currency (e.g., LAK) or USD cents
	MaxUsers  int
	MaxPages  int
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReplySetModel struct {
	ID        string           `gorm:"type:uuid;primaryKey"`
	AccountID string           `gorm:"type:uuid;not null;index"`
	Account   AccountModel     `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	Code      string           `gorm:"size:60"`
	Name      string           `gorm:"size:255;not null"`
	Items     []ReplyItemModel `gorm:"foreignKey:ReplySetID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReplyItemModel struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	ReplySetID string `gorm:"type:uuid;not null;index"`
	Type       string `gorm:"size:20;not null"`   // "text", "image", "video", "audio"
	Content    string `gorm:"type:text;not null"` // Text message OR URL of the media
	OrderIndex int    `gorm:"not null"`           // Used for sorting front/back
	IsEnabled  bool   `gorm:"not null;default:true"`
}

type ProductModel struct {
	ID          string       `gorm:"type:uuid;primaryKey"`
	AccountID   string       `gorm:"type:uuid;index;not null"`
	Account     AccountModel `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	Code        string       `gorm:"size:50;index"`
	Name        string       `gorm:"size:120;not null"`
	Price       string       `gorm:"size:60;not null"`
	Description string       `gorm:"size:1000"`
	ImageUrl    string       `gorm:"size:1000"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AutomationRuleModel struct {
	ID               string        `gorm:"type:uuid;primaryKey"`
	AccountID        string        `gorm:"type:uuid;index;not null"`
	Account          AccountModel  `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	PageID           string        `gorm:"index"`
	TriggerType      string        `gorm:"size:50;not null"` // post, ad, keyword
	TriggerName      string        `gorm:"size:255"`
	TriggerValue     string        `gorm:"size:255;index"`
	ProductID        *string       `gorm:"type:uuid;index"`
	Product          *ProductModel `gorm:"foreignKey:ProductID;constraint:OnDelete:SET NULL"`
	ReplySetID       string        `gorm:"type:uuid;not null"`
	ReplySet         ReplySetModel `gorm:"foreignKey:ReplySetID;constraint:OnDelete:CASCADE"`
	FlowID           string        `gorm:"size:64;index"`
	FlowName         string        `gorm:"size:160"`
	FirstMessageOnly bool          `gorm:"default:true"`
	CooldownSeconds  int           `gorm:"default:0"`
	IsActive         bool          `gorm:"default:true"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type SubscriptionModel struct {
	ID               string       `gorm:"type:uuid;primaryKey"`
	AccountID        string       `gorm:"type:uuid;uniqueIndex;not null"`
	Account          AccountModel `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	PlanID           string       `gorm:"size:50;not null"`
	Plan             PlanModel    `gorm:"foreignKey:PlanID;constraint:OnDelete:RESTRICT"`
	Status           string       `gorm:"size:50;not null"` // e.g., "active", "expired", "canceled"
	CurrentPeriodEnd time.Time    `gorm:"not null"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TransactionModel struct {
	ID            string       `gorm:"type:uuid;primaryKey"`
	AccountID     string       `gorm:"type:uuid;not null;index"`
	Account       AccountModel `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	Amount        int          `gorm:"not null"`
	Currency      string       `gorm:"size:10;not null"`
	Status        string       `gorm:"size:50;not null"` // e.g., "pending", "completed", "failed"
	PaymentMethod string       `gorm:"size:50"`
	Reference     string       `gorm:"size:255"` // Transfer ref or Stripe Session ID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type MetaConnectionModel struct {
	AccountID           string       `gorm:"type:uuid;primaryKey"`
	Account             AccountModel `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	UserToken           string       `gorm:"type:text;not null"`
	TokenExpiresAt      int64
	DataAccessExpiresAt int64
	GrantedPermissions  string `gorm:"type:text"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type MetaPageTokenModel struct {
	PageID      string       `gorm:"size:255;primaryKey"`
	AccountID   string       `gorm:"type:uuid;not null;index"`
	Account     AccountModel `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	Name        string       `gorm:"size:255"`
	Category    string       `gorm:"size:255"`
	PictureURL  string       `gorm:"size:1000"`
	AccessToken string       `gorm:"type:text;not null"`
	IsConnected bool         `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MetaWhatsAppConnectionModel stores the credential issued by WhatsApp
// Embedded Signup for one phone number. It is deliberately scoped to a
// workspace so webhook events can be routed without relying on a shared
// Facebook user session.
type MetaWhatsAppConnectionModel struct {
	PhoneNumberID      string       `gorm:"size:255;primaryKey"`
	AccountID          string       `gorm:"type:uuid;not null;index"`
	Account            AccountModel `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	BusinessID         string       `gorm:"size:255;index"`
	WABAID             string       `gorm:"size:255;not null;index"`
	DisplayPhoneNumber string       `gorm:"size:80"`
	VerifiedName       string       `gorm:"size:255"`
	PictureURL         string       `gorm:"size:1000"`
	AccessToken        string       `gorm:"type:text;not null"`
	IsConnected        bool         `gorm:"default:true"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type LeadModel struct {
	ID string `json:"id"`; AccountID string `json:"accountId"`; PageID string `json:"pageId"`; SenderID string `json:"senderId"`; SourceType string `json:"sourceType"`; SourceID string `json:"sourceId"`; Name string `json:"name"`; Status string `json:"status"`; AssignedUserID string `json:"assignedUserId"`
	AlertRead bool `json:"alertRead"`; FollowUpEnabled bool `json:"followUpEnabled"`; NextFollowUpAt *time.Time `json:"nextFollowUpAt,omitempty"`; FollowUpStep int `json:"followUpStep"`; CreatedAt time.Time `json:"createdAt"`; UpdatedAt time.Time `json:"updatedAt"`
}

type LeadSettingsModel struct {
	AccountID       string `gorm:"type:uuid;primaryKey" json:"accountId"`
	AssignmentMode  string `gorm:"size:30;not null;default:round_robin" json:"assignmentMode"`
	AlertsEnabled   bool   `gorm:"not null;default:true" json:"alertsEnabled"`
	FollowUpEnabled bool   `gorm:"not null;default:false" json:"followUpEnabled"`
	CAPIEnabled     bool   `gorm:"not null;default:false" json:"capiEnabled"`
	CAPIDatasetID   string `gorm:"size:100" json:"capiDatasetId"`
	CAPIToken       string `gorm:"type:text" json:"-"`
	UpdatedAt       time.Time
}

type LeadFollowUpStepModel struct {
	ID string `gorm:"type:uuid;primaryKey" json:"id"`; AccountID string `gorm:"type:uuid;not null;index" json:"accountId"`; StepIndex int `json:"stepIndex"`; DelayMinutes int `json:"delayMinutes"`; Message string `gorm:"type:text;not null" json:"message"`; IsEnabled bool `gorm:"not null;default:true" json:"isEnabled"`
}

type CAPIEventModel struct {
	EventID              string `gorm:"size:100;primaryKey"`
	AccountID            string `gorm:"type:uuid;not null;index"`
	LeadID               string `gorm:"type:uuid;not null;index"`
	Status               string `gorm:"size:30;not null"`
	Error                string `gorm:"type:text"`
	CreatedAt, UpdatedAt time.Time
}
