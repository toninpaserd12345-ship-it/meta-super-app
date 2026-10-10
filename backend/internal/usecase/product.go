package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/meta-super-app/backend/internal/domain"
)

type Product struct {
	repo domain.ProductRepository
}

func NewProduct(repo domain.ProductRepository) *Product {
	return &Product{repo: repo}
}

func (u *Product) ListProducts(ctx context.Context, userID, accountID string) ([]domain.Product, error) {
	return u.repo.ListProducts(ctx, accountID)
}

func (u *Product) SaveProduct(ctx context.Context, userID, accountID string, product domain.Product) (*domain.Product, error) {
	product.Code = strings.TrimSpace(product.Code)
	product.Name = strings.TrimSpace(product.Name)
	product.Price = strings.TrimSpace(product.Price)
	product.Description = strings.TrimSpace(product.Description)

	if product.Name == "" || product.Price == "" {
		return nil, errors.New("product name and price are required")
	}
	if len(product.Code) > 50 || len(product.Name) > 120 || len(product.Price) > 60 || len(product.Description) > 1000 {
		return nil, errors.New("product fields exceed the allowed length")
	}

	return u.repo.SaveProduct(ctx, accountID, product)
}

func (u *Product) GetProduct(ctx context.Context, userID, accountID, productID string) (*domain.Product, error) {
	return u.repo.GetProduct(ctx, accountID, productID)
}

func (u *Product) DeleteProduct(ctx context.Context, userID, accountID, productID string) error {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return errors.New("product ID is required")
	}
	return u.repo.DeleteProduct(ctx, accountID, productID)
}
