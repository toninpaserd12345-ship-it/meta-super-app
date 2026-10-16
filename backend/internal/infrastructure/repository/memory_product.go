package repository

import (
	"context"
	"errors"
	"github.com/google/uuid"

	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
)

func (r *MemoryRepository) ListProducts(ctx context.Context, accountID string) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Product, 0)
	for _, p := range r.products {
		if p.AccountID == accountID {
			result = append(result, domain.Product{
				ID:          p.ID,
				Code:        p.Code,
				Name:        p.Name,
				Price:       p.Price,
				SalePrice:   p.SalePrice,
				Cost:        p.Cost,
				Quantity:    p.Quantity,
				Currency:    p.Currency,
				Description: p.Description,
				ImageUrl:    p.ImageUrl,
				Barcode:     p.Barcode,
				Category:    p.Category,
				Unit:        p.Unit,
				Weight:      p.Weight,
				GalleryURLs: p.GalleryURLs,
				MinStock:    p.MinStock,
				Status:      p.Status,
				Colors:            p.Colors,
				Sizes:             p.Sizes,
				Variants:          p.Variants,
				PackagingCost:     p.PackagingCost,
				ShippingCostStore: p.ShippingCostStore,
				TargetProfit:      p.TargetProfit,
			})
		}
	}
	return result, nil
}

func (r *MemoryRepository) SaveProduct(ctx context.Context, accountID string, product domain.Product) (*domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if product.ID == "" {
		product.ID = uuid.NewString()
	} else {
		// Check if exists
		existing, ok := r.products[product.ID]
		if !ok || existing.AccountID != accountID {
			return nil, errors.New("product not found")
		}
	}

	r.products[product.ID] = database.ProductModel{
		ID:                product.ID,
		AccountID:         accountID,
		Code:              product.Code,
		Name:              product.Name,
		Price:             product.Price,
		SalePrice:         product.SalePrice,
		Cost:              product.Cost,
		Quantity:          product.Quantity,
		Currency:          product.Currency,
		Description:       product.Description,
		ImageUrl:          product.ImageUrl,
		Barcode:           product.Barcode,
		Category:          product.Category,
		Unit:              product.Unit,
		Weight:            product.Weight,
		GalleryURLs:       product.GalleryURLs,
		MinStock:          product.MinStock,
		Status:            product.Status,
		Colors:            product.Colors,
		Sizes:             product.Sizes,
		Variants:          product.Variants,
		PackagingCost:     product.PackagingCost,
		ShippingCostStore: product.ShippingCostStore,
		TargetProfit:      product.TargetProfit,
	}
	return &product, nil
}

func (r *MemoryRepository) GetProduct(ctx context.Context, accountID, productID string) (*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.products[productID]
	if !ok || p.AccountID != accountID {
		return nil, nil // not found
	}
	return &domain.Product{
		ID:                p.ID,
		Code:              p.Code,
		Name:              p.Name,
		Price:             p.Price,
		SalePrice:         p.SalePrice,
		Cost:              p.Cost,
		Quantity:          p.Quantity,
		Currency:          p.Currency,
		Description:       p.Description,
		ImageUrl:          p.ImageUrl,
		Barcode:           p.Barcode,
		Category:          p.Category,
		Unit:              p.Unit,
		Weight:            p.Weight,
		GalleryURLs:       p.GalleryURLs,
		MinStock:          p.MinStock,
		Status:            p.Status,
		Colors:            p.Colors,
		Sizes:             p.Sizes,
		Variants:          p.Variants,
		PackagingCost:     p.PackagingCost,
		ShippingCostStore: p.ShippingCostStore,
		TargetProfit:      p.TargetProfit,
	}, nil
}

func (r *MemoryRepository) DeleteProduct(ctx context.Context, accountID, productID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	product, ok := r.products[productID]
	if !ok || product.AccountID != accountID {
		return errors.New("product not found")
	}
	delete(r.products, productID)
	return nil
}
