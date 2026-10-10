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
				Name:        p.Name,
				Price:       p.Price,
				Description: p.Description,
				ImageUrl:    p.ImageUrl,
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
		ID:          product.ID,
		AccountID:   accountID,
		Name:        product.Name,
		Price:       product.Price,
		Description: product.Description,
		ImageUrl:    product.ImageUrl,
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
		ID:          p.ID,
		Name:        p.Name,
		Price:       p.Price,
		Description: p.Description,
		ImageUrl:    p.ImageUrl,
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
