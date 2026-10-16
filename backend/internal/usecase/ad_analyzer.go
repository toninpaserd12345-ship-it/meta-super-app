package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
)

type AdAnalyzer struct {
	db   *gorm.DB
	meta *Meta
}

func NewAdAnalyzer(db *gorm.DB, meta *Meta) *AdAnalyzer {
	return &AdAnalyzer{
		db:   db,
		meta: meta,
	}
}

// -------------------------------------------------------------------------
// Strategy Settings Types & Methods (Point-in-Time Versioning)
// -------------------------------------------------------------------------

type SaveStrategySettingsInput struct {
	AdAccountID          string     `json:"adAccountId"`
	EffectiveFrom        *time.Time `json:"effectiveFrom,omitempty"`
	TargetProfitPerOrder float64    `json:"targetProfitPerOrder"`
	AvgPackagingCost     float64    `json:"avgPackagingCost"`
	AvgShippingCostStore float64    `json:"avgShippingCostStore"`
	CODReturnRatePercent float64    `json:"codReturnRatePercent"`
	ReturnFeePerOrder    float64    `json:"returnFeePerOrder"`
	DailyBudgetCap       float64    `json:"dailyBudgetCap"`
	MaxTestLossBudget    float64    `json:"maxTestLossBudget"`
	Note                 string     `json:"note"`
}

func (u *AdAnalyzer) GetActiveStrategySettings(ctx context.Context, accountID, adAccountID string, targetTime time.Time) (*database.AdStrategySettingsModel, error) {
	if targetTime.IsZero() {
		targetTime = time.Now()
	}

	var setting database.AdStrategySettingsModel
	// Query matching AdAccountID or workspace default (ad_account_id = '') that was effective at targetTime
	err := u.db.WithContext(ctx).
		Where("account_id = ? AND (ad_account_id = ? OR ad_account_id = '')", accountID, adAccountID).
		Where("effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", targetTime, targetTime).
		Order("ad_account_id DESC, effective_from DESC").
		First(&setting).Error

	if err == nil {
		return &setting, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Smart defaults for Lao Chat Commerce if not yet configured
		return &database.AdStrategySettingsModel{
			ID:                   "default",
			AccountID:            accountID,
			AdAccountID:          adAccountID,
			EffectiveFrom:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			TargetProfitPerOrder: 30000,
			AvgPackagingCost:     5000,
			AvgShippingCostStore: 15000,
			CODReturnRatePercent: 8.0,
			ReturnFeePerOrder:    15000,
			DailyBudgetCap:       500000,
			MaxTestLossBudget:    200000,
			Note:                 "Default store settings",
		}, nil
	}

	return nil, err
}

func (u *AdAnalyzer) SaveStrategySettings(ctx context.Context, accountID, adAccountID string, in SaveStrategySettingsInput) (*database.AdStrategySettingsModel, error) {
	effectiveFrom := time.Now()
	if in.EffectiveFrom != nil && !in.EffectiveFrom.IsZero() {
		effectiveFrom = *in.EffectiveFrom
	}

	returnFee := in.ReturnFeePerOrder
	if returnFee <= 0 {
		returnFee = in.AvgShippingCostStore
	}
	codRate := in.CODReturnRatePercent
	if codRate <= 0 {
		codRate = 8.0
	}

	newID := uuid.NewString()
	var newModel database.AdStrategySettingsModel

	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Close previous active setting for this ad account
		_ = tx.Model(&database.AdStrategySettingsModel{}).
			Where("account_id = ? AND ad_account_id = ? AND effective_to IS NULL", accountID, adAccountID).
			Update("effective_to", effectiveFrom).Error

		newModel = database.AdStrategySettingsModel{
			ID:                   newID,
			AccountID:            accountID,
			AdAccountID:          adAccountID,
			EffectiveFrom:        effectiveFrom,
			EffectiveTo:          nil,
			TargetProfitPerOrder: in.TargetProfitPerOrder,
			AvgPackagingCost:     in.AvgPackagingCost,
			AvgShippingCostStore: in.AvgShippingCostStore,
			CODReturnRatePercent: codRate,
			ReturnFeePerOrder:    returnFee,
			DailyBudgetCap:       in.DailyBudgetCap,
			MaxTestLossBudget:    in.MaxTestLossBudget,
			Note:                 strings.TrimSpace(in.Note),
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}

		return tx.Create(&newModel).Error
	})

	if err != nil {
		return nil, err
	}
	return &newModel, nil
}

func (u *AdAnalyzer) ListStrategySettingsHistory(ctx context.Context, accountID, adAccountID string) ([]database.AdStrategySettingsModel, error) {
	var list []database.AdStrategySettingsModel
	err := u.db.WithContext(ctx).
		Where("account_id = ? AND (ad_account_id = ? OR ad_account_id = '')", accountID, adAccountID).
		Order("effective_from DESC").
		Find(&list).Error
	return list, err
}

// -------------------------------------------------------------------------
// Performance Analysis with Real CPA & COD breakdown
// -------------------------------------------------------------------------

type AdPerformanceItem struct {
	EntityType      string `json:"entityType"` // "campaign", "adset", "ad"
	EntityID        string `json:"entityId"`
	EntityName      string `json:"entityName"`
	CampaignID      string `json:"campaignId,omitempty"`
	CampaignName    string `json:"campaignName,omitempty"`
	AdSetID         string `json:"adSetId,omitempty"`
	AdSetName       string `json:"adSetName,omitempty"`
	Status          string `json:"status"`
	EffectiveStatus string `json:"effectiveStatus"`

	// Meta metrics
	Spend           float64  `json:"spend"`
	Impressions     float64  `json:"impressions"`
	Reach           float64  `json:"reach"`
	Frequency       float64  `json:"frequency"`
	LinkClicks      float64  `json:"linkClicks"`
	LinkCTR         float64  `json:"linkCtr"`
	LinkCPC         float64  `json:"linkCpc"`
	CPM             float64  `json:"cpm"`
	ReportedResults float64  `json:"reportedResults"`
	ReportedCost    *float64 `json:"reportedCost,omitempty"`

	// Real Order & Payment metrics
	TotalOrders           int     `json:"totalOrders"`
	PrepaidOrders         int     `json:"prepaidOrders"`         // transfer or cash, paid
	PrepaidRevenue        float64 `json:"prepaidRevenue"`        // Total cash collected upfront
	CODTotalOrders        int     `json:"codTotalOrders"`        // All COD bills
	CODDeliveredOrders    int     `json:"codDeliveredOrders"`    // COD delivered + paid
	CODDeliveredRevenue   float64 `json:"codDeliveredRevenue"`   // COD cash successfully remitted
	CODPendingOrders      int     `json:"codPendingOrders"`      // In transit (shipping/pending)
	CODPendingRevenue     float64 `json:"codPendingRevenue"`     // Potential cash on delivery
	CODReturnedOrders     int     `json:"codReturnedOrders"`     // Cancelled / refused COD
	CompletedOrders       int     `json:"completedOrders"`       // Prepaid + COD Delivered
	CompletedRevenue      float64 `json:"completedRevenue"`      // Total cash realized
	GrossProfitBeforeAds  float64 `json:"grossProfitBeforeAds"`  // Realized net margin before ads
	NetContribution       float64 `json:"netContribution"`       // Gross profit - Spend

	// Decision Engine metrics
	RealCPA              *float64 `json:"realCpa,omitempty"`              // Spend / CompletedOrders
	TargetCPA            float64  `json:"targetCpa"`                      // Target threshold
	BreakEvenCPA         float64  `json:"breakEvenCpa"`                   // Break-even threshold
	ClosingRate          *float64 `json:"closingRate,omitempty"`          // CompletedOrders / Chats
	DecisionCode         string   `json:"decisionCode"`                   // SCALE, HOLD, IMPROVE, PAUSE, PAUSE_TEST, INSUFFICIENT_DATA
	StatusBadgeColor     string   `json:"statusBadgeColor"`               // success, info, warning, error, secondary
	Reason               string   `json:"reason"`                         // Lao explanation
	RecommendedAction    string   `json:"recommendedAction"`              // Lao concrete recommendation
	ProposedBudgetChange *string  `json:"proposedBudgetChange,omitempty"` // e.g. "200,000 -> 240,000 (+20%)"

	// Product & Auto Reply Binding
	ProductID         string `json:"productId,omitempty"`
	ProductName       string `json:"productName,omitempty"`
	ProductImageUrl   string `json:"productImageUrl,omitempty"`
	ProductPrice      string `json:"productPrice,omitempty"`
	AutoReplyFlowID   string `json:"autoReplyFlowId,omitempty"`
	AutoReplyFlowName string `json:"autoReplyFlowName,omitempty"`
	IsLinkedToProduct bool   `json:"isLinkedToProduct"`
}

type PerformanceAnalysisSummary struct {
	TotalSpend            float64  `json:"totalSpend"`
	TotalImpressions      float64  `json:"totalImpressions"`
	TotalChats            int      `json:"totalChats"`
	TotalPrepaidOrders    int      `json:"totalPrepaidOrders"`
	TotalPrepaidRevenue   float64  `json:"totalPrepaidRevenue"`
	TotalCODOrders        int      `json:"totalCodOrders"`
	TotalCODDelivered     int      `json:"totalCodDelivered"`
	TotalCODDeliveredRev  float64  `json:"totalCodDeliveredRevenue"`
	TotalCODPending       int      `json:"totalCodPending"`
	TotalCODPendingRev    float64  `json:"totalCodPendingRevenue"`
	TotalCODReturned      int      `json:"totalCodReturned"`
	TotalCompletedOrders  int      `json:"totalCompletedOrders"`
	TotalCompletedRevenue float64  `json:"totalCompletedRevenue"`
	OverallRealCPA        *float64 `json:"overallRealCpa,omitempty"`
	TotalGrossProfit      float64  `json:"totalGrossProfit"`
	TotalNetProfit        float64  `json:"totalNetProfit"`
	PrepaidPercentage     float64  `json:"prepaidPercentage"` // e.g. 65%
}

type PerformanceActionPlan struct {
	ToPause         []string `json:"toPause"`
	ToScale         []string `json:"toScale"`
	ToHold          []string `json:"toHold"`
	NextExperiments []string `json:"nextExperiments"`
	NextReviewDate  string   `json:"nextReviewDate"`
	DailyBudgetCap  float64  `json:"dailyBudgetCap"`
}

type PerformanceAnalysisReport struct {
	Currency    string                     `json:"currency"`
	Timezone    string                     `json:"timezone"`
	Level       string                     `json:"level"`
	DateStart   string                     `json:"dateStart"`
	DateStop    string                     `json:"dateStop"`
	GeneratedAt string                     `json:"generatedAt"`
	Strategy    *database.AdStrategySettingsModel `json:"strategy"`
	Summary     PerformanceAnalysisSummary `json:"summary"`
	Items       []AdPerformanceItem        `json:"items"`
	ActionPlan  PerformanceActionPlan      `json:"actionPlan"`
}

func (u *AdAnalyzer) AnalyzePerformance(ctx context.Context, userID, accountID, adAccountID, level, since, until string) (*PerformanceAnalysisReport, error) {
	// 1. Resolve Timezone: default Asia/Vientiane (GMT+7)
	loc, err := time.LoadLocation("Asia/Vientiane")
	if err != nil {
		loc = time.FixedZone("Asia/Vientiane", 7*3600)
	}

	nowLoc := time.Now().In(loc)
	if until == "" {
		until = nowLoc.Format("2006-01-02")
	}
	if since == "" {
		since = nowLoc.AddDate(0, 0, -6).Format("2006-01-02")
	}

	startDate, err := time.ParseInLocation("2006-01-02", since, loc)
	if err != nil {
		startDate = nowLoc.AddDate(0, 0, -6)
		since = startDate.Format("2006-01-02")
	}
	endDate, err := time.ParseInLocation("2006-01-02", until, loc)
	if err != nil {
		endDate = nowLoc
		until = endDate.Format("2006-01-02")
	}
	endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, loc)

	// 2. Load Meta Insights
	insightsReport, err := u.meta.AdsInsights(ctx, userID, accountID, adAccountID, level, since, until)
	if err != nil {
		return nil, fmt.Errorf("load meta ads insights: %w", err)
	}

	// 3. Load Strategy Settings effective for this date range (Point-in-Time)
	strategy, _ := u.GetActiveStrategySettings(ctx, accountID, adAccountID, startDate)

	// 4. Query all Orders created in this period for this account
	var orders []database.OrderModel
	_ = u.db.WithContext(ctx).
		Preload("Items").
		Where("account_id = ? AND created_at >= ? AND created_at <= ?", accountID, startDate, endOfDay).
		Find(&orders).Error

	// Group orders by AdID and CampaignID
	ordersByAdID := make(map[string][]database.OrderModel)
	ordersByCampaignID := make(map[string][]database.OrderModel)
	for _, ord := range orders {
		if ord.AdID != "" {
			ordersByAdID[ord.AdID] = append(ordersByAdID[ord.AdID], ord)
		}
		if ord.CampaignID != "" {
			ordersByCampaignID[ord.CampaignID] = append(ordersByCampaignID[ord.CampaignID], ord)
		}
	}

	// 4b. Query Automation Rules to link Ads/Campaigns to Products via Auto Reply
	var autoRules []database.AutomationRuleModel
	_ = u.db.WithContext(ctx).
		Preload("Product").
		Where("account_id = ?", accountID).
		Find(&autoRules).Error

	rulesByTriggerValue := make(map[string]database.AutomationRuleModel)
	for _, rule := range autoRules {
		if rule.TriggerValue != "" {
			rulesByTriggerValue[rule.TriggerValue] = rule
		}
	}

	// 5. Build Performance Items
	items := make([]AdPerformanceItem, 0, len(insightsReport.Items))
	var sumSummary PerformanceAnalysisSummary
	sumSummary.TotalChats = 0

	var planPause []string
	var planScale []string
	var planHold []string
	var planExperiments []string

	for _, metaItem := range insightsReport.Items {
		spend := 0.0
		if metaItem.Spend != nil {
			spend = *metaItem.Spend
		}
		impressions := 0.0
		if metaItem.Impressions != nil {
			impressions = *metaItem.Impressions
		}
		reach := 0.0
		if metaItem.Reach != nil {
			reach = *metaItem.Reach
		}
		frequency := 0.0
		if metaItem.Frequency != nil {
			frequency = *metaItem.Frequency
		}
		linkClicks := 0.0
		if metaItem.LinkClicks != nil {
			linkClicks = *metaItem.LinkClicks
		}
		linkCtr := 0.0
		if metaItem.LinkCTR != nil {
			linkCtr = *metaItem.LinkCTR
		}
		linkCpc := 0.0
		if metaItem.LinkCPC != nil {
			linkCpc = *metaItem.LinkCPC
		}
		cpm := 0.0
		if metaItem.CPM != nil {
			cpm = *metaItem.CPM
		}
		repResults := 0.0
		if metaItem.Results.Value != nil {
			repResults = *metaItem.Results.Value
		}

		// Match orders according to level
		var matchedOrders []database.OrderModel
		if level == "ad" && metaItem.AdID != "" {
			matchedOrders = ordersByAdID[metaItem.AdID]
		} else if (level == "campaign" || level == "adset") && metaItem.CampaignID != "" {
			matchedOrders = ordersByCampaignID[metaItem.CampaignID]
		} else if metaItem.EntityID != "" {
			if ords, ok := ordersByAdID[metaItem.EntityID]; ok {
				matchedOrders = ords
			} else if ords, ok := ordersByCampaignID[metaItem.EntityID]; ok {
				matchedOrders = ords
			}
		}

		// Calculate order tallies
		prepaidCount := 0
		prepaidRev := 0.0
		codTotalCount := 0
		codDeliveredCount := 0
		codDeliveredRev := 0.0
		codPendingCount := 0
		codPendingRev := 0.0
		codReturnedCount := 0
		grossProfitBeforeAds := 0.0

		for _, o := range matchedOrders {
			isPaid := strings.EqualFold(o.PaymentStatus, "paid")
			isDelivered := strings.EqualFold(o.FulfillmentStatus, "delivered")
			isCancelled := strings.EqualFold(o.FulfillmentStatus, "cancelled")
			isShipping := strings.EqualFold(o.FulfillmentStatus, "shipping") || strings.EqualFold(o.FulfillmentStatus, "pending")

			if !o.IsCOD && isPaid {
				prepaidCount++
				prepaidRev += o.TotalAmount
				grossProfitBeforeAds += o.NetProfitBeforeAds
			} else if o.IsCOD {
				codTotalCount++
				if isDelivered && isPaid {
					codDeliveredCount++
					codDeliveredRev += o.TotalAmount
					grossProfitBeforeAds += o.NetProfitBeforeAds
				} else if isCancelled {
					codReturnedCount++
				} else if isShipping {
					codPendingCount++
					codPendingRev += o.TotalAmount
				}
			}
		}

		completedOrders := prepaidCount + codDeliveredCount
		completedRev := prepaidRev + codDeliveredRev
		netContribution := grossProfitBeforeAds - spend

		// Calculate Real CPA safely (prevent divide by zero)
		var realCPA *float64
		if completedOrders > 0 {
			val := spend / float64(completedOrders)
			realCPA = &val
		}

		// Closing rate calculation
		chatsCount := int(repResults)
		if chatsCount <= 0 {
			chatsCount = len(matchedOrders)
		}
		var closingRate *float64
		if chatsCount > 0 && completedOrders > 0 {
			cr := float64(completedOrders) / float64(chatsCount)
			closingRate = &cr
		}

		// Match linked Product & Auto Reply Flow
		var linkedProduct *database.ProductModel
		var linkedFlowID, linkedFlowName string
		if rule, ok := rulesByTriggerValue[metaItem.EntityID]; ok && rule.Product != nil {
			linkedProduct = rule.Product
			linkedFlowID = rule.FlowID
			linkedFlowName = rule.FlowName
		} else if metaItem.AdID != "" {
			if rule, ok := rulesByTriggerValue[metaItem.AdID]; ok && rule.Product != nil {
				linkedProduct = rule.Product
				linkedFlowID = rule.FlowID
				linkedFlowName = rule.FlowName
			}
		}
		if linkedProduct == nil && metaItem.CampaignID != "" {
			if rule, ok := rulesByTriggerValue[metaItem.CampaignID]; ok && rule.Product != nil {
				linkedProduct = rule.Product
				linkedFlowID = rule.FlowID
				linkedFlowName = rule.FlowName
			}
		}

		// Also check if any order items matched to this ad/campaign have a product
		if linkedProduct == nil && len(matchedOrders) > 0 {
			for _, ord := range matchedOrders {
				for _, itm := range ord.Items {
					if itm.ProductID != "" {
						var prod database.ProductModel
						if err := u.db.WithContext(ctx).Where("id = ?", itm.ProductID).First(&prod).Error; err == nil && prod.ID != "" {
							linkedProduct = &prod
							break
						}
					}
				}
				if linkedProduct != nil {
					break
				}
			}
		}

		// Break-even and Target CPA calculation with SKU-specific overrides
		avgSellingPrice := 150000.0
		if completedOrders > 0 {
			avgSellingPrice = completedRev / float64(completedOrders)
		}

		effectiveTargetProfit := strategy.TargetProfitPerOrder
		effectivePackagingCost := strategy.AvgPackagingCost
		effectiveShippingCostStore := strategy.AvgShippingCostStore

		if linkedProduct != nil {
			if linkedProduct.TargetProfit > 0 {
				effectiveTargetProfit = linkedProduct.TargetProfit
			}
			if linkedProduct.PackagingCost > 0 {
				effectivePackagingCost = linkedProduct.PackagingCost
			}
			if linkedProduct.ShippingCostStore > 0 {
				effectiveShippingCostStore = linkedProduct.ShippingCostStore
			}
		}

		codLossFactor := (strategy.CODReturnRatePercent / 100.0) * strategy.ReturnFeePerOrder
		totalVarPerOrder := effectivePackagingCost + effectiveShippingCostStore + codLossFactor
		breakEvenCPA := avgSellingPrice - totalVarPerOrder
		if breakEvenCPA < 20000 {
			breakEvenCPA = 85000 // default reasonable benchmark
		}
		targetCPA := breakEvenCPA - effectiveTargetProfit
		if targetCPA < 15000 {
			targetCPA = breakEvenCPA * 0.65
		}

		// Decision Logic
		status := strings.ToUpper(metaItem.EffectiveStatus)
		if status == "" {
			status = strings.ToUpper(metaItem.Status)
		}
		if status == "" {
			status = "ACTIVE"
		}

		decisionCode, badgeColor, reason, action, proposedBudget := ClassifyAdDecision(
			spend, impressions, targetCPA, breakEvenCPA, realCPA,
			completedOrders, codPendingCount, codPendingRev, strategy.MaxTestLossBudget,
		)

		if decisionCode == "SCALE" && realCPA != nil {
			planScale = append(planScale, fmt.Sprintf("%s: Real CPA %.0f ₭ (ເປົ້າ %.0f ₭) -> ສະເໜີ Scale +20%%", metaItem.EntityName, *realCPA, targetCPA))
		} else if decisionCode == "HOLD" && codPendingCount > 0 {
			planHold = append(planHold, fmt.Sprintf("%s: ລໍຖ້າຜົນອໍເດີ (COD %d ບິນ)", metaItem.EntityName, codPendingCount))
		} else if decisionCode == "PAUSE" || decisionCode == "PAUSE_TEST" {
			planPause = append(planPause, fmt.Sprintf("%s: %s", metaItem.EntityName, reason))
		}

		item := AdPerformanceItem{
			EntityType:            metaItem.EntityType,
			EntityID:              metaItem.EntityID,
			EntityName:            metaItem.EntityName,
			CampaignID:            metaItem.CampaignID,
			CampaignName:          metaItem.CampaignName,
			AdSetID:               metaItem.AdSetID,
			AdSetName:             metaItem.AdSetName,
			Status:                metaItem.Status,
			EffectiveStatus:       status,
			Spend:                 spend,
			Impressions:           impressions,
			Reach:                 reach,
			Frequency:             frequency,
			LinkClicks:            linkClicks,
			LinkCTR:               linkCtr,
			LinkCPC:               linkCpc,
			CPM:                   cpm,
			ReportedResults:       repResults,
			ReportedCost:          metaItem.Results.Cost,
			TotalOrders:           len(matchedOrders),
			PrepaidOrders:         prepaidCount,
			PrepaidRevenue:        prepaidRev,
			CODTotalOrders:        codTotalCount,
			CODDeliveredOrders:    codDeliveredCount,
			CODDeliveredRevenue:   codDeliveredRev,
			CODPendingOrders:      codPendingCount,
			CODPendingRevenue:     codPendingRev,
			CODReturnedOrders:     codReturnedCount,
			CompletedOrders:       completedOrders,
			CompletedRevenue:      completedRev,
			GrossProfitBeforeAds:  grossProfitBeforeAds,
			NetContribution:       netContribution,
			RealCPA:               realCPA,
			TargetCPA:             targetCPA,
			BreakEvenCPA:          breakEvenCPA,
			ClosingRate:           closingRate,
			DecisionCode:          decisionCode,
			StatusBadgeColor:      badgeColor,
			Reason:                reason,
			RecommendedAction:     action,
			ProposedBudgetChange:  proposedBudget,
			IsLinkedToProduct:     linkedProduct != nil,
		}

		if linkedProduct != nil {
			item.ProductID = linkedProduct.ID
			item.ProductName = linkedProduct.Name
			item.ProductImageUrl = linkedProduct.ImageUrl
			item.ProductPrice = linkedProduct.Price
			item.AutoReplyFlowID = linkedFlowID
			item.AutoReplyFlowName = linkedFlowName
		}

		items = append(items, item)

		// Accumulate totals
		sumSummary.TotalSpend += spend
		sumSummary.TotalImpressions += impressions
		sumSummary.TotalChats += chatsCount
		sumSummary.TotalPrepaidOrders += prepaidCount
		sumSummary.TotalPrepaidRevenue += prepaidRev
		sumSummary.TotalCODOrders += codTotalCount
		sumSummary.TotalCODDelivered += codDeliveredCount
		sumSummary.TotalCODDeliveredRev += codDeliveredRev
		sumSummary.TotalCODPending += codPendingCount
		sumSummary.TotalCODPendingRev += codPendingRev
		sumSummary.TotalCODReturned += codReturnedCount
		sumSummary.TotalCompletedOrders += completedOrders
		sumSummary.TotalCompletedRevenue += completedRev
		sumSummary.TotalGrossProfit += grossProfitBeforeAds
		sumSummary.TotalNetProfit += netContribution
	}

	if sumSummary.TotalCompletedOrders > 0 {
		cpa := sumSummary.TotalSpend / float64(sumSummary.TotalCompletedOrders)
		sumSummary.OverallRealCPA = &cpa
	}
	if sumSummary.TotalCompletedRevenue > 0 {
		sumSummary.PrepaidPercentage = math.Round((sumSummary.TotalPrepaidRevenue / sumSummary.TotalCompletedRevenue) * 100)
	}

	nextRevDate := nowLoc.AddDate(0, 0, 3).Format("02/01/2006")
	if len(planScale) > 0 {
		planExperiments = append(planExperiments, "ທົດສອບ Creative/Hook ໃໝ່ ສຳລັບ Campaign ທີ່ກຳລັງ Scale ງົບ 50,000 ₭/ມື້")
	}

	report := &PerformanceAnalysisReport{
		Currency:    insightsReport.Currency,
		Timezone:    "Asia/Vientiane",
		Level:       level,
		DateStart:   since,
		DateStop:    until,
		GeneratedAt: nowLoc.Format(time.RFC3339),
		Strategy:    strategy,
		Summary:     sumSummary,
		Items:       items,
		ActionPlan: PerformanceActionPlan{
			ToPause:         planPause,
			ToScale:         planScale,
			ToHold:          planHold,
			NextExperiments: planExperiments,
			NextReviewDate:  nextRevDate,
			DailyBudgetCap:  strategy.DailyBudgetCap,
		},
	}

	return report, nil
}

// ClassifyAdDecision evaluates an ad performance against the SOP rules and returns the decision code, badge color, reason, recommended action, and proposed budget change.
func ClassifyAdDecision(
	spend, impressions, targetCPA, breakEvenCPA float64,
	realCPA *float64,
	completedOrders, codPendingCount int,
	codPendingRev, maxTestLossBudget float64,
) (decisionCode, badgeColor, reason, action string, proposedBudget *string) {
	if spend <= 0 && impressions <= 0 {
		return "INSUFFICIENT_DATA", "secondary", "ໂຄສະນາຫາກໍ່ສ້າງໃໝ່ ຫຼື ຍັງບໍ່ທັນມີ Spend", "ລໍຖ້າໃຫ້ໂຄສະນາເລີ່ມນຳສົ່ງຢ່າງໜ້ອຍ 24-48 ຊົ່ວໂມງ", nil
	}

	if realCPA != nil && *realCPA <= targetCPA && completedOrders >= 10 {
		newBudget := spend * 1.2
		budgetText := fmt.Sprintf("+20%% ງົບປະມານ (ປະມານ %.0f ₭)", newBudget)
		return "SCALE", "success",
			fmt.Sprintf("CPA ດີກວ່າເປົ້າ (%.0f ₭ ≤ %.0f ₭), ອໍເດີສຳເລັດ %d ອໍເດີ (≥10)", *realCPA, targetCPA, completedOrders),
			"ສະເໜີເພີ່ມງົບ 15-20% (CBO/ABO) ແລະ ຕິດຕາມຜົນ 48-72 ຊົ່ວໂມງ",
			&budgetText
	}

	if (realCPA != nil && *realCPA <= targetCPA && completedOrders < 10) || codPendingCount > 0 {
		var r string
		if codPendingCount > 0 {
			r = fmt.Sprintf("ມີອໍເດີ COD ກຳລັງເດີນທາງ %d ອໍເດີ (ຍອດ %.0f ₭) ຍັງລໍຖ້າຜົນການສົ່ງສຳເລັດ", codPendingCount, codPendingRev)
		} else {
			r = fmt.Sprintf("CPA ດີ ແຕ່ຈຳນວນອໍເດີຍັງໜ້ອຍ (%d ອໍເດີ < 10 ອໍເດີ)", completedOrders)
		}
		return "HOLD", "info", r, "ຄົງງົບເກົ່າ, ຫ້າມປິດແອດ ແລະ ລະບຸວັນກວດຄືນຫຼັງ COD ສົ່ງສຳເລັດ", nil
	}

	if realCPA != nil && *realCPA > targetCPA && *realCPA < breakEvenCPA {
		return "IMPROVE", "warning",
			fmt.Sprintf("CPA ເກີນເປົ້າ ແຕ່ຍັງບໍ່ຫຼຸດທຶນ (%.0f ₭ < %.0f ₭ < %.0f ₭)", targetCPA, *realCPA, breakEvenCPA),
			"ບໍ່ Scale, ທົດລອງແກ້ໄຂ 1 ປັດໄຈ (Hook ວິດີໂອ, ປັບ Script ຕອບແຊັດ, ປັບໂປຣໂມຊັ່ນ)",
			nil
	}

	if realCPA != nil && *realCPA >= breakEvenCPA {
		return "PAUSE", "error",
			fmt.Sprintf("CPA ຫຼຸດທຶນ (%.0f ₭ ≥ Break-even %.0f ₭)", *realCPA, breakEvenCPA),
			"ສະເໜີພັກໂຄສະນາທັນທີ ເພື່ອຢຸດງົບຮົ່ວໄຫຼ",
			nil
	}

	if completedOrders == 0 && (spend >= 2*targetCPA || (maxTestLossBudget > 0 && spend >= maxTestLossBudget)) {
		return "PAUSE_TEST", "error",
			fmt.Sprintf("ຍັງບໍ່ມີອໍເດີເລີຍ ແລະ Spend (%.0f ₭) ຮອດເພດານງົບທົດລອງ", spend),
			"ສະເໜີພັກໂຄສະນາ, ກວດເຊັກປຸ່ມທັກແຊັດ ແລະ ຊິ້ນງານໂຄສະນາ",
			nil
	}

	return "HOLD", "info", "ຍັງຢູ່ໃນຊ່ວງທົດລອງເລີ່ມຕົ້ນ (Learning Phase)", "ຄົງງົບເກົ່າ ແລະ ຕິດຕາມຜົນຢ່າງໃກ້ຊິດ", nil
}

