package repository

import (
	"context"
	"testing"

	"github.com/meta-super-app/backend/internal/domain"
)

func TestMemoryProductDeleteIsScopedToAccount(t *testing.T) {
	ctx := context.Background()
	repo := NewMemory("")
	product, err := repo.SaveProduct(ctx, "account-1", domain.Product{Name: "Product", Price: "100"})
	if err != nil {
		t.Fatalf("SaveProduct() error = %v", err)
	}
	if err = repo.DeleteProduct(ctx, "account-2", product.ID); err == nil {
		t.Fatal("DeleteProduct() allowed a different account")
	}
	if err = repo.DeleteProduct(ctx, "account-1", product.ID); err != nil {
		t.Fatalf("DeleteProduct() error = %v", err)
	}
	remaining, err := repo.ListProducts(ctx, "account-1")
	if err != nil {
		t.Fatalf("ListProducts() error = %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("ListProducts() count = %d, want 0", len(remaining))
	}
}
