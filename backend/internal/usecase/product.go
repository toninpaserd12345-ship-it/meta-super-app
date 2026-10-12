package usecase

import (
	"context"
	"errors"
	"fmt"
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
	product.Currency = strings.ToUpper(strings.TrimSpace(product.Currency))
	product.Description = strings.TrimSpace(product.Description)
	if product.Currency == "" {
		product.Currency = "LAK"
	}
	if product.SalePrice != nil && product.Price == "" {
		product.Price = fmt.Sprintf("%g %s", *product.SalePrice, product.Currency)
	}

	if product.Name == "" || (product.Price == "" && product.SalePrice == nil) {
		return nil, errors.New("product name and price are required")
	}
	if (product.SalePrice != nil && *product.SalePrice < 0) || (product.Cost != nil && *product.Cost < 0) || (product.Quantity != nil && *product.Quantity < 0) {
		return nil, errors.New("product price, cost and quantity cannot be negative")
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
