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

type Inventory struct {
	db *gorm.DB
}

func NewInventory(db *gorm.DB) *Inventory {
	return &Inventory{db: db}
}

type CreateWarehouseInput struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "warehouse" or "shop"
	Address   string `json:"address"`
	Phone     string `json:"phone"`
	IsDefault bool   `json:"isDefault"`
}

type UpdateWarehouseInput struct {
	Code      *string `json:"code"`
	Name      *string `json:"name"`
	Type      *string `json:"type"`
	Address   *string `json:"address"`
	Phone     *string `json:"phone"`
	IsDefault *bool   `json:"isDefault"`
	IsActive  *bool   `json:"isActive"`
}

type SetStockInput struct {
	WarehouseID    string `json:"warehouseId"`
	Quantity       int64  `json:"quantity"`
	AlertThreshold *int64 `json:"alertThreshold"`
}

func (u *Inventory) EnsureDefaultWarehouse(ctx context.Context, accountID string) (*database.WarehouseModel, error) {
	var wh database.WarehouseModel
	err := u.db.WithContext(ctx).Where("account_id = ? AND is_default = ?", accountID, true).First(&wh).Error
	if err == nil {
		return &wh, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// If no default, check if any warehouse exists
	err = u.db.WithContext(ctx).Where("account_id = ?", accountID).Order("created_at asc").First(&wh).Error
	if err == nil {
		wh.IsDefault = true
		_ = u.db.WithContext(ctx).Model(&wh).Update("is_default", true)
		return &wh, nil
	}

	// Create default warehouse
	wh = database.WarehouseModel{
		ID:        uuid.NewString(),
		AccountID: accountID,
		Code:      "WH-01",
		Name:      "ສາງໃຫຍ່ຫຼັກ (Main Warehouse)",
		Type:      "warehouse",
		IsDefault: true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if createErr := u.db.WithContext(ctx).Create(&wh).Error; createErr != nil {
		return nil, createErr
	}
	return &wh, nil
}

func (u *Inventory) ListWarehouses(ctx context.Context, accountID string) ([]database.WarehouseModel, error) {
	_, _ = u.EnsureDefaultWarehouse(ctx, accountID)
	var list []database.WarehouseModel
	err := u.db.WithContext(ctx).Where("account_id = ?", accountID).Order("is_default desc, type asc, created_at asc").Find(&list).Error
	return list, err
}

func (u *Inventory) CreateWarehouse(ctx context.Context, accountID string, in CreateWarehouseInput) (*database.WarehouseModel, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("warehouse name is required")
	}
	code := strings.TrimSpace(in.Code)
	if code == "" {
		code = fmt.Sprintf("WH-%s", strings.ToUpper(uuid.NewString()[:6]))
	}
	whType := strings.TrimSpace(in.Type)
	if whType != "shop" {
		whType = "warehouse"
	}

	wh := database.WarehouseModel{
		ID:        uuid.NewString(),
		AccountID: accountID,
		Code:      code,
		Name:      name,
		Type:      whType,
		Address:   strings.TrimSpace(in.Address),
		Phone:     strings.TrimSpace(in.Phone),
		IsDefault: in.IsDefault,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if in.IsDefault {
			if err := tx.Model(&database.WarehouseModel{}).Where("account_id = ?", accountID).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&wh).Error
	})
	if err != nil {
		return nil, err
	}
	return &wh, nil
}

func (u *Inventory) UpdateWarehouse(ctx context.Context, accountID, id string, in UpdateWarehouseInput) (*database.WarehouseModel, error) {
	var wh database.WarehouseModel
	if err := u.db.WithContext(ctx).Where("id = ? AND account_id = ?", id, accountID).First(&wh).Error; err != nil {
		return nil, err
	}

	updates := map[string]any{"updated_at": time.Now()}
	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		updates["name"] = strings.TrimSpace(*in.Name)
	}
	if in.Code != nil && strings.TrimSpace(*in.Code) != "" {
		updates["code"] = strings.TrimSpace(*in.Code)
	}
	if in.Type != nil && (*in.Type == "warehouse" || *in.Type == "shop") {
		updates["type"] = *in.Type
	}
	if in.Address != nil {
		updates["address"] = strings.TrimSpace(*in.Address)
	}
	if in.Phone != nil {
		updates["phone"] = strings.TrimSpace(*in.Phone)
	}
	if in.IsActive != nil {
		updates["is_active"] = *in.IsActive
	}

	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if in.IsDefault != nil && *in.IsDefault {
			if err := tx.Model(&database.WarehouseModel{}).Where("account_id = ?", accountID).Update("is_default", false).Error; err != nil {
				return err
			}
			updates["is_default"] = true
		}
		return tx.Model(&wh).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	_ = u.db.WithContext(ctx).Where("id = ?", id).First(&wh)
	return &wh, nil
}

func (u *Inventory) DeleteWarehouse(ctx context.Context, accountID, id string) error {
	var wh database.WarehouseModel
	if err := u.db.WithContext(ctx).Where("id = ? AND account_id = ?", id, accountID).First(&wh).Error; err != nil {
		return err
	}
	if wh.IsDefault {
		return errors.New("cannot delete default warehouse")
	}
	return u.db.WithContext(ctx).Where("id = ? AND account_id = ?", id, accountID).Delete(&database.WarehouseModel{}).Error
}

func (u *Inventory) GetProductStocks(ctx context.Context, accountID, productID string) ([]database.WarehouseStockModel, error) {
	var stocks []database.WarehouseStockModel
	err := u.db.WithContext(ctx).
		Preload("Warehouse").
		Where("account_id = ? AND product_id = ?", accountID, productID).
		Find(&stocks).Error
	return stocks, err
}

func (u *Inventory) SetProductStock(ctx context.Context, accountID, productID, warehouseID string, quantity int64, alertThreshold *int64) error {
	var stock database.WarehouseStockModel
	err := u.db.WithContext(ctx).Where("account_id = ? AND product_id = ? AND warehouse_id = ?", accountID, productID, warehouseID).First(&stock).Error

	threshold := int64(5)
	if alertThreshold != nil {
		threshold = *alertThreshold
	}

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			stock = database.WarehouseStockModel{
				ID:             uuid.NewString(),
				AccountID:      accountID,
				ProductID:      productID,
				WarehouseID:    warehouseID,
				Quantity:       quantity,
				AlertThreshold: threshold,
				UpdatedAt:      time.Now(),
			}
			if createErr := tx.Create(&stock).Error; createErr != nil {
				return createErr
			}
		} else if err != nil {
			return err
		} else {
			if updateErr := tx.Model(&stock).Updates(map[string]any{
				"quantity":        quantity,
				"alert_threshold": threshold,
				"updated_at":      time.Now(),
			}).Error; updateErr != nil {
				return updateErr
			}
		}

		// Recalculate total product quantity across all warehouses
		var totalQty int64
		if sumErr := tx.Model(&database.WarehouseStockModel{}).
			Where("account_id = ? AND product_id = ?", accountID, productID).
			Select("COALESCE(SUM(quantity), 0)").Scan(&totalQty).Error; sumErr == nil {
			_ = tx.Model(&database.ProductModel{}).Where("id = ? AND account_id = ?", productID, accountID).Update("quantity", totalQty)
		}
		return nil
	})
}
