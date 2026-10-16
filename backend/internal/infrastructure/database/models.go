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
	SalePrice   *float64     `gorm:"type:decimal(18,2)"`
	Cost        *float64     `gorm:"type:decimal(18,2)"`
	Quantity    *int64
	Currency    string `gorm:"size:8"`
	Description string `gorm:"size:1000" json:"description"`
	ImageUrl    string `gorm:"size:1000" json:"imageUrl"`
	Barcode     string `gorm:"size:100;index" json:"barcode"`
	Category    string `gorm:"size:100;index" json:"category"`
	Unit        string `gorm:"size:30" json:"unit"`
	Weight      *float64 `gorm:"type:decimal(10,2)" json:"weight"`
	GalleryURLs string `gorm:"type:text" json:"galleryUrls"`
	MinStock    *int64 `json:"minStock"`
	Status      string `gorm:"size:30;default:'active'" json:"status"`
	Colors      string `gorm:"size:255" json:"colors"`
	Sizes       string `gorm:"size:255" json:"sizes"`
	Variants          string    `gorm:"type:text" json:"variants"`
	PackagingCost     float64   `gorm:"type:decimal(18,2);default:0" json:"packagingCost"`
	ShippingCostStore float64   `gorm:"type:decimal(18,2);default:0" json:"shippingCostStore"`
	TargetProfit      float64   `gorm:"type:decimal(18,2);default:0" json:"targetProfit"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type AutomationRuleModel struct {
	ID               string        `gorm:"type:uuid;primaryKey"`
	AccountID        string        `gorm:"type:uuid;index;not null"`
	Account          AccountModel  `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
	PageID           string        `gorm:"index"`
	TriggerType      string        `gorm:"size:50;not null"` // post, ad, keyword
	TriggerName      string        `gorm:"type:text"`
	TriggerValue     string        `gorm:"size:255;index"`
	ProductID        *string       `gorm:"type:uuid;index"`
	Product          *ProductModel `gorm:"foreignKey:ProductID;constraint:OnDelete:SET NULL"`
	ReplySetID       string        `gorm:"type:uuid;not null"`
	ReplySet         ReplySetModel `gorm:"foreignKey:ReplySetID;constraint:OnDelete:CASCADE"`
	FlowID           string        `gorm:"size:64;index"`
	FlowName         string        `gorm:"size:160"`
	CommentReplyText string        `gorm:"type:text"`
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
	ID              string     `gorm:"type:uuid;primaryKey" json:"id"`
	AccountID       string     `gorm:"type:uuid;index;not null" json:"accountId"`
	PageID          string     `gorm:"size:50;index;not null" json:"pageId"`
	SenderID        string     `gorm:"size:50;index;not null" json:"senderId"`
	SourceType      string     `gorm:"size:20" json:"sourceType"`
	SourceID        string     `gorm:"size:100" json:"sourceId"`
	ProductID       string     `gorm:"type:uuid;index" json:"productId,omitempty"`
	Name            string     `gorm:"size:255" json:"name"`
	Phone           string     `gorm:"size:50" json:"phone"`
	Address         string     `gorm:"type:text" json:"address"`
	Notes           string     `gorm:"type:text" json:"notes"`
	Tags            string     `gorm:"size:255" json:"tags"`
	Status          string     `gorm:"size:50;index" json:"status"`
	AssignedUserID  string     `gorm:"type:uuid;index" json:"assignedUserId"`
	AlertRead       bool       `gorm:"index" json:"alertRead"`
	FollowUpEnabled bool       `json:"followUpEnabled"`
	NextFollowUpAt  *time.Time `gorm:"index" json:"nextFollowUpAt,omitempty"`
	FollowUpStep    int        `json:"followUpStep"`
	CreatedAt       time.Time  `gorm:"index" json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
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
	ID           string `gorm:"type:uuid;primaryKey" json:"id"`
	AccountID    string `gorm:"type:uuid;not null;index" json:"accountId"`
	StepIndex    int    `json:"stepIndex"`
	DelayMinutes int    `json:"delayMinutes"`
	Message      string `gorm:"type:text;not null" json:"message"`
	IsEnabled    bool   `gorm:"not null;default:true" json:"isEnabled"`
}

type CAPIEventModel struct {
	EventID              string `gorm:"size:100;primaryKey"`
	AccountID            string `gorm:"type:uuid;not null;index"`
	LeadID               string `gorm:"type:uuid;not null;index"`
	Status               string `gorm:"size:30;not null"`
	Error                string `gorm:"type:text"`
	CreatedAt, UpdatedAt time.Time
}

type WarehouseModel struct {
	ID        string       `gorm:"type:uuid;primaryKey" json:"id"`
	AccountID string       `gorm:"type:uuid;index;not null" json:"accountId"`
	Account   AccountModel `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE" json:"-"`
	Code      string       `gorm:"size:50;index" json:"code"`
	Name      string       `gorm:"size:160;not null" json:"name"`
	Type      string       `gorm:"size:30;not null;default:'warehouse'" json:"type"` // "warehouse" or "shop"
	Address   string       `gorm:"type:text" json:"address"`
	Phone     string       `gorm:"size:50" json:"phone"`
	IsDefault bool         `gorm:"default:false" json:"isDefault"`
	IsActive  bool         `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

type WarehouseStockModel struct {
	ID             string          `gorm:"type:uuid;primaryKey" json:"id"`
	AccountID      string          `gorm:"type:uuid;index;not null" json:"accountId"`
	ProductID      string          `gorm:"type:uuid;index;not null" json:"productId"`
	Product        *ProductModel   `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"product,omitempty"`
	WarehouseID    string          `gorm:"type:uuid;index;not null" json:"warehouseId"`
	Warehouse      *WarehouseModel `gorm:"foreignKey:WarehouseID;constraint:OnDelete:CASCADE" json:"warehouse,omitempty"`
	Quantity       int64           `gorm:"not null;default:0" json:"quantity"`
	AlertThreshold int64           `gorm:"default:5" json:"alertThreshold"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type OrderModel struct {
	ID                 string           `gorm:"type:uuid;primaryKey" json:"id"`
	AccountID          string           `gorm:"type:uuid;index;not null" json:"accountId"`
	Account            AccountModel     `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE" json:"-"`
	OrderNumber        string           `gorm:"size:60;uniqueIndex;not null" json:"orderNumber"`
	LeadID             *string          `gorm:"type:uuid;index" json:"leadId,omitempty"`
	AdID               string           `gorm:"size:100;index" json:"adId,omitempty"`
	CampaignID         string           `gorm:"size:100;index" json:"campaignId,omitempty"`
	SenderID           string           `gorm:"size:60;index" json:"senderId"`
	PageID             string           `gorm:"size:60;index" json:"pageId"`
	WarehouseID        string           `gorm:"type:uuid;index" json:"warehouseId"`
	Warehouse          *WarehouseModel  `gorm:"foreignKey:WarehouseID;constraint:OnDelete:SET NULL" json:"warehouse,omitempty"`
	CustomerName       string           `gorm:"size:255" json:"customerName"`
	CustomerPhone      string           `gorm:"size:50" json:"customerPhone"`
	CustomerAddress    string           `gorm:"type:text" json:"customerAddress"`
	Subtotal           float64          `gorm:"type:decimal(18,2);default:0" json:"subtotal"`
	Discount           float64          `gorm:"type:decimal(18,2);default:0" json:"discount"`
	ShippingFee        float64          `gorm:"type:decimal(18,2);default:0" json:"shippingFee"`
	TotalAmount        float64          `gorm:"type:decimal(18,2);default:0" json:"totalAmount"`
	ProductCost        float64          `gorm:"type:decimal(18,2);default:0" json:"productCost"`
	PackagingCost      float64          `gorm:"type:decimal(18,2);default:0" json:"packagingCost"`
	StoreShippingCost  float64          `gorm:"type:decimal(18,2);default:0" json:"storeShippingCost"`
	ReturnFee          float64          `gorm:"type:decimal(18,2);default:0" json:"returnFee"`
	TotalVariableCost  float64          `gorm:"type:decimal(18,2);default:0" json:"totalVariableCost"`
	NetProfitBeforeAds float64          `gorm:"type:decimal(18,2);default:0" json:"netProfitBeforeAds"`
	IsCOD              bool             `gorm:"default:false;index" json:"isCod"`
	PrepaidAmount      float64          `gorm:"type:decimal(18,2);default:0" json:"prepaidAmount"`
	CODAmount          float64          `gorm:"type:decimal(18,2);default:0" json:"codAmount"`
	CODTransferStatus  string           `gorm:"size:30;default:'pending'" json:"codTransferStatus"` // pending, transferred
	Currency           string           `gorm:"size:10;default:'LAK'" json:"currency"`
	PaymentStatus      string           `gorm:"size:30;default:'unpaid'" json:"paymentStatus"`       // unpaid, paid, refunded
	PaymentMethod      string           `gorm:"size:50" json:"paymentMethod"`                       // transfer, cod, cash
	FulfillmentStatus  string           `gorm:"size:30;default:'pending'" json:"fulfillmentStatus"` // pending, confirmed, shipping, delivered, cancelled
	DeliveryCourier    string           `gorm:"size:100" json:"deliveryCourier"`                    // Anousith, HAL, Houngaloun, Flash, Kerry, etc.
	DeliveryTracking   string           `gorm:"size:255" json:"deliveryTracking"`
	Notes              string           `gorm:"type:text" json:"notes"`
	FacebookLeadSent   bool             `gorm:"default:false" json:"facebookLeadSent"`
	Items              []OrderItemModel `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"items"`
	CreatedAt          time.Time        `gorm:"index" json:"createdAt"`
	UpdatedAt          time.Time        `json:"updatedAt"`
}

type OrderItemModel struct {
	ID           string        `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID      string        `gorm:"type:uuid;index;not null" json:"orderId"`
	ProductID    string        `gorm:"type:uuid;index;not null" json:"productId"`
	Product      *ProductModel `gorm:"foreignKey:ProductID;constraint:OnDelete:SET NULL" json:"product,omitempty"`
	ProductCode  string        `gorm:"size:50" json:"productCode"`
	ProductName  string        `gorm:"size:255;not null" json:"productName"`
	ProductImage string        `gorm:"size:1000" json:"productImage"`
	Color        string        `gorm:"size:60" json:"color"`
	Size         string        `gorm:"size:60" json:"size"`
	Quantity     int           `gorm:"not null;default:1" json:"quantity"`
	UnitPrice    float64       `gorm:"type:decimal(18,2);not null" json:"unitPrice"`
	TotalPrice   float64       `gorm:"type:decimal(18,2);not null" json:"totalPrice"`
	CostPrice    float64       `gorm:"type:decimal(18,2);default:0" json:"costPrice"`
	TotalCost    float64       `gorm:"type:decimal(18,2);default:0" json:"totalCost"`
}

type AdStrategySettingsModel struct {
	ID                   string       `gorm:"type:uuid;primaryKey" json:"id"`
	AccountID            string       `gorm:"type:uuid;index;not null" json:"accountId"`
	Account              AccountModel `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE" json:"-"`
	AdAccountID          string       `gorm:"size:100;index" json:"adAccountId"`
	EffectiveFrom        time.Time    `gorm:"index;not null" json:"effectiveFrom"`
	EffectiveTo          *time.Time   `gorm:"index" json:"effectiveTo,omitempty"`
	TargetProfitPerOrder float64      `gorm:"type:decimal(18,2);default:0" json:"targetProfitPerOrder"`
	AvgPackagingCost     float64      `gorm:"type:decimal(18,2);default:0" json:"avgPackagingCost"`
	AvgShippingCostStore float64      `gorm:"type:decimal(18,2);default:0" json:"avgShippingCostStore"`
	CODReturnRatePercent float64      `gorm:"type:decimal(6,2);default:8.0" json:"codReturnRatePercent"`
	ReturnFeePerOrder    float64      `gorm:"type:decimal(18,2);default:0" json:"returnFeePerOrder"`
	DailyBudgetCap       float64      `gorm:"type:decimal(18,2);default:0" json:"dailyBudgetCap"`
	MaxTestLossBudget    float64      `gorm:"type:decimal(18,2);default:0" json:"maxTestLossBudget"`
	Note                 string       `gorm:"size:500" json:"note"`
	CreatedAt            time.Time    `json:"createdAt"`
	UpdatedAt            time.Time    `json:"updatedAt"`
}
