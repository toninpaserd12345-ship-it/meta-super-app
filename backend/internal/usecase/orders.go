package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
)

type Orders struct {
	db        *gorm.DB
	leads     *Leads
	stream    *ChatStream
	inventory *Inventory
}

func NewOrders(db *gorm.DB, leads *Leads, stream *ChatStream, inventory *Inventory) *Orders {
	return &Orders{
		db:        db,
		leads:     leads,
		stream:    stream,
		inventory: inventory,
	}
}

type CreateOrderItemInput struct {
	ProductID string  `json:"productId"`
	Color     string  `json:"color"`
	Size      string  `json:"size"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}

type CreateOrderInput struct {
	WarehouseID       string                 `json:"warehouseId"`
	CustomerName      string                 `json:"customerName"`
	CustomerPhone     string                 `json:"customerPhone"`
	CustomerAddress   string                 `json:"customerAddress"`
	SenderID          string                 `json:"senderId"`
	PageID            string                 `json:"pageId"`
	LeadID            string                 `json:"leadId"`
	AdID              string                 `json:"adId"`
	CampaignID        string                 `json:"campaignId"`
	Currency          string                 `json:"currency"`
	Discount          float64                `json:"discount"`
	ShippingFee       float64                `json:"shippingFee"`
	PackagingCost     *float64               `json:"packagingCost,omitempty"`
	StoreShippingCost *float64               `json:"storeShippingCost,omitempty"`
	PaymentStatus     string                 `json:"paymentStatus"`
	PaymentMethod     string                 `json:"paymentMethod"`
	FulfillmentStatus string                 `json:"fulfillmentStatus"`
	DeliveryCourier   string                 `json:"deliveryCourier"`
	DeliveryTracking  string                 `json:"deliveryTracking"`
	Notes             string                 `json:"notes"`
	Items             []CreateOrderItemInput `json:"items"`
}

type UpdateOrderStatusInput struct {
	PaymentStatus     *string  `json:"paymentStatus"`
	PaymentMethod     *string  `json:"paymentMethod"`
	FulfillmentStatus *string  `json:"fulfillmentStatus"`
	DeliveryCourier   *string  `json:"deliveryCourier"`
	DeliveryTracking  *string  `json:"deliveryTracking"`
	Notes             *string  `json:"notes"`
	ReturnFee         *float64 `json:"returnFee"`
	CODTransferStatus *string  `json:"codTransferStatus"`
}

func (u *Orders) ListOrders(ctx context.Context, accountID string, status string) ([]database.OrderModel, error) {
	q := u.db.WithContext(ctx).
		Preload("Warehouse").
		Preload("Items").
		Where("account_id = ?", accountID)

	if status != "" && status != "all" {
		q = q.Where("fulfillment_status = ? OR payment_status = ?", status, status)
	}

	var list []database.OrderModel
	err := q.Order("created_at desc").Find(&list).Error
	return list, err
}

func (u *Orders) GetOrder(ctx context.Context, accountID, id string) (*database.OrderModel, error) {
	var order database.OrderModel
	err := u.db.WithContext(ctx).
		Preload("Warehouse").
		Preload("Items").
		Where("(id = ? OR order_number = ?) AND account_id = ?", id, id, accountID).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (u *Orders) CreateOrder(ctx context.Context, accountID string, in CreateOrderInput) (*database.OrderModel, error) {
	if len(in.Items) == 0 {
		return nil, errors.New("order must contain at least one product")
	}

	// 1. Resolve source warehouse
	warehouseID := in.WarehouseID
	if warehouseID == "" {
		defWH, err := u.inventory.EnsureDefaultWarehouse(ctx, accountID)
		if err != nil {
			return nil, fmt.Errorf("resolve default warehouse: %w", err)
		}
		warehouseID = defWH.ID
	}

	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency == "" {
		currency = "LAK"
	}

	payStatus := strings.TrimSpace(in.PaymentStatus)
	if payStatus == "" {
		payStatus = "paid"
	}

	fulStatus := strings.TrimSpace(in.FulfillmentStatus)
	if fulStatus == "" {
		fulStatus = "confirmed"
	}

	orderID := uuid.NewString()

	// 2. Generate clean, human-friendly Order Number (ORD-YYYYMMDD-XXXX)
	todayPrefix := fmt.Sprintf("ORD-%s", time.Now().Format("20060102"))
	var countToday int64
	_ = u.db.WithContext(ctx).Model(&database.OrderModel{}).
		Where("account_id = ? AND order_number LIKE ?", accountID, todayPrefix+"%").
		Count(&countToday)
	orderNumber := fmt.Sprintf("%s-%04d", todayPrefix, countToday+1)

	var subtotal float64
	var totalProductCost float64
	var calcPackagingCost float64
	var calcStoreShippingCost float64
	var orderItems []database.OrderItemModel

	// Transaction: create order, create items, deduct stock
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, itemIn := range in.Items {
			if itemIn.Quantity <= 0 {
				itemIn.Quantity = 1
			}

			var prod database.ProductModel
			if err := tx.Where("id = ? AND account_id = ?", itemIn.ProductID, accountID).First(&prod).Error; err != nil {
				return fmt.Errorf("product %s not found: %w", itemIn.ProductID, err)
			}

			unitPrice := itemIn.UnitPrice
			if unitPrice <= 0 && prod.SalePrice != nil {
				unitPrice = *prod.SalePrice
			}

			itemTotal := unitPrice * float64(itemIn.Quantity)
			subtotal += itemTotal

			unitCost := 0.0
			if prod.Cost != nil {
				unitCost = *prod.Cost
			}
			itemTotalCost := unitCost * float64(itemIn.Quantity)
			totalProductCost += itemTotalCost
			calcPackagingCost += prod.PackagingCost * float64(itemIn.Quantity)
			calcStoreShippingCost += prod.ShippingCostStore

			orderItems = append(orderItems, database.OrderItemModel{
				ID:           uuid.NewString(),
				OrderID:      orderID,
				ProductID:    prod.ID,
				ProductCode:  prod.Code,
				ProductName:  prod.Name,
				ProductImage: prod.ImageUrl,
				Color:        itemIn.Color,
				Size:         itemIn.Size,
				Quantity:     itemIn.Quantity,
				UnitPrice:    unitPrice,
				TotalPrice:   itemTotal,
				CostPrice:    unitCost,
				TotalCost:    itemTotalCost,
			})

			// 3. Deduct stock from specific warehouse
			var whStock database.WarehouseStockModel
			whStockErr := tx.Where("account_id = ? AND product_id = ? AND warehouse_id = ?", accountID, prod.ID, warehouseID).First(&whStock).Error
			if errors.Is(whStockErr, gorm.ErrRecordNotFound) {
				// Init stock if not present
				whStock = database.WarehouseStockModel{
					ID:             uuid.NewString(),
					AccountID:      accountID,
					ProductID:      prod.ID,
					WarehouseID:    warehouseID,
					Quantity:       -int64(itemIn.Quantity),
					AlertThreshold: 5,
					UpdatedAt:      time.Now(),
				}
				if createStockErr := tx.Create(&whStock).Error; createStockErr != nil {
					return createStockErr
				}
			} else if whStockErr == nil {
				newQty := whStock.Quantity - int64(itemIn.Quantity)
				if updateStockErr := tx.Model(&whStock).Updates(map[string]any{
					"quantity":   newQty,
					"updated_at": time.Now(),
				}).Error; updateStockErr != nil {
					return updateStockErr
				}
			}

			// 4. Deduct total product quantity
			if prod.Quantity != nil {
				newTot := *prod.Quantity - int64(itemIn.Quantity)
				if err := tx.Model(&prod).Update("quantity", newTot).Error; err != nil {
					return err
				}
			}
		}

		totalAmount := subtotal + in.ShippingFee - in.Discount
		if totalAmount < 0 {
			totalAmount = 0
		}

		var leadIDPtr *string
		if in.LeadID != "" {
			leadIDPtr = &in.LeadID
		}

		adID := strings.TrimSpace(in.AdID)
		campaignID := strings.TrimSpace(in.CampaignID)
		if (adID == "" || campaignID == "") && leadIDPtr != nil {
			var lead database.LeadModel
			if err := tx.Where("id = ?", *leadIDPtr).First(&lead).Error; err == nil {
				if lead.SourceType == "ad" && adID == "" {
					adID = lead.SourceID
				} else if lead.SourceType == "campaign" && campaignID == "" {
					campaignID = lead.SourceID
				}
			}
		}

		isCOD := strings.EqualFold(strings.TrimSpace(in.PaymentMethod), "cod")
		prepaidAmount := 0.0
		codAmount := 0.0
		if isCOD {
			codAmount = totalAmount
		} else {
			prepaidAmount = totalAmount
		}

		finalPkgCost := calcPackagingCost
		if in.PackagingCost != nil {
			finalPkgCost = *in.PackagingCost
		}
		finalStoreShipCost := calcStoreShippingCost
		if in.StoreShippingCost != nil {
			finalStoreShipCost = *in.StoreShippingCost
		}
		totalVariableCost := totalProductCost + finalPkgCost + finalStoreShipCost
		netProfitBeforeAds := totalAmount - totalVariableCost

		orderRow := database.OrderModel{
			ID:                 orderID,
			AccountID:          accountID,
			OrderNumber:        orderNumber,
			LeadID:             leadIDPtr,
			AdID:               adID,
			CampaignID:         campaignID,
			SenderID:           strings.TrimSpace(in.SenderID),
			PageID:             strings.TrimSpace(in.PageID),
			WarehouseID:        warehouseID,
			CustomerName:       strings.TrimSpace(in.CustomerName),
			CustomerPhone:      strings.TrimSpace(in.CustomerPhone),
			CustomerAddress:    strings.TrimSpace(in.CustomerAddress),
			Subtotal:           subtotal,
			Discount:           in.Discount,
			ShippingFee:        in.ShippingFee,
			TotalAmount:        totalAmount,
			ProductCost:        totalProductCost,
			PackagingCost:      finalPkgCost,
			StoreShippingCost:  finalStoreShipCost,
			TotalVariableCost:  totalVariableCost,
			NetProfitBeforeAds: netProfitBeforeAds,
			IsCOD:              isCOD,
			PrepaidAmount:      prepaidAmount,
			CODAmount:          codAmount,
			CODTransferStatus:  "pending",
			Currency:           currency,
			PaymentStatus:      payStatus,
			PaymentMethod:      strings.TrimSpace(in.PaymentMethod),
			FulfillmentStatus:  fulStatus,
			DeliveryCourier:    strings.TrimSpace(in.DeliveryCourier),
			DeliveryTracking:   strings.TrimSpace(in.DeliveryTracking),
			Notes:              strings.TrimSpace(in.Notes),
			FacebookLeadSent:   false,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if err := tx.Create(&orderRow).Error; err != nil {
			return err
		}

		for i := range orderItems {
			if err := tx.Create(&orderItems[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 5. Load complete order with preloads
	order, _ := u.GetOrder(ctx, accountID, orderID)

	// 6. Update or link Lead in CRM
	senderOrPhone := in.SenderID
	if senderOrPhone == "" {
		senderOrPhone = in.CustomerPhone
	}
	if senderOrPhone != "" {
		memo := fmt.Sprintf("[Order %s] ຍອດລວມ: %.2f %s · ສະຖານະ: %s", orderNumber, order.TotalAmount, order.Currency, order.FulfillmentStatus)
		statusVal := "purchased"
		_, _ = u.leads.UpdateLeadDetails(ctx, accountID, senderOrPhone, LeadDetailsInput{
			Name:    &in.CustomerName,
			Phone:   &in.CustomerPhone,
			Address: &in.CustomerAddress,
			Status:  &statusVal,
			Notes:   &memo,
			Tags:    nil, // tag update handled in leads
		})
	}

	// 7. Dispatch Meta CAPI (Purchase event)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		var targetLead database.LeadModel
		if in.LeadID != "" {
			_ = u.db.WithContext(bgCtx).Where("id = ?", in.LeadID).First(&targetLead).Error
		} else if in.SenderID != "" {
			_ = u.db.WithContext(bgCtx).Where("account_id = ? AND sender_id = ?", accountID, in.SenderID).First(&targetLead).Error
		}

		if targetLead.ID == "" {
			targetLead = database.LeadModel{
				ID:         order.ID,
				AccountID:  accountID,
				PageID:     order.PageID,
				SenderID:   order.SenderID,
				Name:       order.CustomerName,
				Phone:      order.CustomerPhone,
				Address:    order.CustomerAddress,
				SourceType: "order",
				Status:     "purchased",
			}
		}

		u.leads.sendCAPI(bgCtx, targetLead, "Purchase")
		_ = u.db.Model(&database.OrderModel{}).Where("id = ?", order.ID).Update("facebook_lead_sent", true)
	}()

	// 8. Broadcast SSE notification for instant frontend refresh
	if u.stream != nil {
		u.stream.Broadcast(accountID, ChatEvent{
			AccountID:  accountID,
			PageID:     order.PageID,
			SenderID:   order.SenderID,
			SenderName: order.CustomerName,
			Message:    fmt.Sprintf("⚡ ອໍເດີໃໝ່ %s: %.2f %s", order.OrderNumber, order.TotalAmount, order.Currency),
			Type:       "order_created",
			Timestamp:  fmt.Sprintf("%d", time.Now().UnixMilli()),
			Platform:   "system",
		})
	}

	return order, nil
}

func (u *Orders) UpdateOrderStatus(ctx context.Context, accountID, id string, in UpdateOrderStatusInput) (*database.OrderModel, error) {
	var order database.OrderModel
	if err := u.db.WithContext(ctx).Preload("Items").Where("id = ? AND account_id = ?", id, accountID).First(&order).Error; err != nil {
		return nil, err
	}

	oldFulStatus := order.FulfillmentStatus
	updates := map[string]any{"updated_at": time.Now()}

	if in.PaymentStatus != nil && strings.TrimSpace(*in.PaymentStatus) != "" {
		updates["payment_status"] = strings.TrimSpace(*in.PaymentStatus)
	}
	if in.PaymentMethod != nil {
		pm := strings.TrimSpace(*in.PaymentMethod)
		updates["payment_method"] = pm
		isCOD := strings.EqualFold(pm, "cod")
		updates["is_cod"] = isCOD
		if isCOD {
			updates["cod_amount"] = order.TotalAmount
			updates["prepaid_amount"] = 0.0
		} else {
			updates["prepaid_amount"] = order.TotalAmount
			updates["cod_amount"] = 0.0
		}
	}
	if in.FulfillmentStatus != nil && strings.TrimSpace(*in.FulfillmentStatus) != "" {
		updates["fulfillment_status"] = strings.TrimSpace(*in.FulfillmentStatus)
	}
	if in.DeliveryCourier != nil {
		updates["delivery_courier"] = strings.TrimSpace(*in.DeliveryCourier)
	}
	if in.DeliveryTracking != nil {
		updates["delivery_tracking"] = strings.TrimSpace(*in.DeliveryTracking)
	}
	if in.Notes != nil {
		updates["notes"] = strings.TrimSpace(*in.Notes)
	}
	if in.ReturnFee != nil {
		updates["return_fee"] = *in.ReturnFee
		totVar := order.ProductCost + order.PackagingCost + order.StoreShippingCost + *in.ReturnFee
		updates["total_variable_cost"] = totVar
		updates["net_profit_before_ads"] = order.TotalAmount - totVar
	}
	if in.CODTransferStatus != nil {
		updates["cod_transfer_status"] = strings.TrimSpace(*in.CODTransferStatus)
	}

	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// If status changed to cancelled and was NOT cancelled before -> Restore stock!
		if in.FulfillmentStatus != nil && *in.FulfillmentStatus == "cancelled" && oldFulStatus != "cancelled" {
			for _, item := range order.Items {
				// 1. Restore warehouse stock
				var whStock database.WarehouseStockModel
				if err := tx.Where("account_id = ? AND product_id = ? AND warehouse_id = ?", accountID, item.ProductID, order.WarehouseID).First(&whStock).Error; err == nil {
					_ = tx.Model(&whStock).Update("quantity", whStock.Quantity+int64(item.Quantity))
				}
				// 2. Restore total product stock
				var prod database.ProductModel
				if err := tx.Where("id = ? AND account_id = ?", item.ProductID, accountID).First(&prod).Error; err == nil && prod.Quantity != nil {
					_ = tx.Model(&prod).Update("quantity", *prod.Quantity+int64(item.Quantity))
				}
			}
		}

		return tx.Model(&order).Updates(updates).Error
	})

	if err != nil {
		return nil, err
	}

	return u.GetOrder(ctx, accountID, id)
}

func (u *Orders) DeleteOrder(ctx context.Context, accountID, id string) error {
	return u.db.WithContext(ctx).Where("id = ? AND account_id = ?", id, accountID).Delete(&database.OrderModel{}).Error
}
