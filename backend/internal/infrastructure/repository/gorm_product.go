package repository

import (
	"context"
	"errors"
	"github.com/google/uuid"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
)

func (r *GormRepository) ListProducts(ctx context.Context, accountID string) ([]domain.Product, error) {
	var models []database.ProductModel
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("created_at desc").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Product, len(models))
	for i, m := range models {
		result[i] = domain.Product{
			ID:          m.ID,
			Code:        m.Code,
			Name:        m.Name,
			Price:       m.Price,
			SalePrice:   m.SalePrice,
			Cost:        m.Cost,
			Quantity:    m.Quantity,
			Currency:    m.Currency,
			Description: m.Description,
			ImageUrl:    m.ImageUrl,
			Barcode:     m.Barcode,
			Category:    m.Category,
			Unit:        m.Unit,
			Weight:      m.Weight,
			GalleryURLs: m.GalleryURLs,
			MinStock:    m.MinStock,
			Status:      m.Status,
			Colors:      m.Colors,
			Sizes:       m.Sizes,
			Variants:    m.Variants,
		}
	}
	return result, nil
}

func (r *GormRepository) SaveProduct(ctx context.Context, accountID string, product domain.Product) (*domain.Product, error) {
	isCreate := false
	if product.ID == "" {
		isCreate = true
		product.ID = uuid.NewString()
	}

	model := database.ProductModel{
		ID:          product.ID,
		AccountID:   accountID,
		Code:        product.Code,
		Name:        product.Name,
		Price:       product.Price,
		SalePrice:   product.SalePrice,
		Cost:        product.Cost,
		Quantity:    product.Quantity,
		Currency:    product.Currency,
		Description: product.Description,
		ImageUrl:    product.ImageUrl,
		Barcode:     product.Barcode,
		Category:    product.Category,
		Unit:        product.Unit,
		Weight:      product.Weight,
		GalleryURLs: product.GalleryURLs,
		MinStock:    product.MinStock,
		Status:      product.Status,
		Colors:      product.Colors,
		Sizes:       product.Sizes,
		Variants:    product.Variants,
	}

	if isCreate {
		if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
			return nil, err
		}
	} else {
		// Update existing
		res := r.db.WithContext(ctx).Model(&database.ProductModel{}).Where("id = ? AND account_id = ?", product.ID, accountID).Select("*").Updates(model)
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, errors.New("product not found")
		}
	}

	return &product, nil
}

func (r *GormRepository) GetProduct(ctx context.Context, accountID, productID string) (*domain.Product, error) {
	var m database.ProductModel
	if err := r.db.WithContext(ctx).Where("id = ? AND account_id = ?", productID, accountID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &domain.Product{
		ID:          m.ID,
		Code:        m.Code,
		Name:        m.Name,
		Price:       m.Price,
		SalePrice:   m.SalePrice,
		Cost:        m.Cost,
		Quantity:    m.Quantity,
		Currency:    m.Currency,
		Description: m.Description,
		ImageUrl:    m.ImageUrl,
		Barcode:     m.Barcode,
		Category:    m.Category,
		Unit:        m.Unit,
		Weight:      m.Weight,
		GalleryURLs: m.GalleryURLs,
		MinStock:    m.MinStock,
		Status:      m.Status,
		Colors:      m.Colors,
		Sizes:       m.Sizes,
		Variants:    m.Variants,
	}, nil
}

func (r *GormRepository) DeleteProduct(ctx context.Context, accountID, productID string) error {
	result := r.db.WithContext(ctx).Where("id = ? AND account_id = ?", productID, accountID).Delete(&database.ProductModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("product not found")
	}
	return nil
}
