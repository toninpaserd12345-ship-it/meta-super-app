package domain

import "context"

type ProductRepository interface {
	ListProducts(ctx context.Context, accountID string) ([]Product, error)
	SaveProduct(ctx context.Context, accountID string, product Product) (*Product, error)
	GetProduct(ctx context.Context, accountID, productID string) (*Product, error)
}
