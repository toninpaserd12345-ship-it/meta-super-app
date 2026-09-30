package repository

import (
	"context"
	"testing"

	"github.com/meta-super-app/backend/internal/domain"
)

func TestMemoryReplySetCRUDAndItemOrder(t *testing.T) {
	ctx := context.Background()
	repo := NewMemory("")
	set := &domain.ReplySet{AccountID: "account-1", Name: "Price reply"}

	if err := repo.CreateSet(ctx, set); err != nil {
		t.Fatalf("CreateSet() error = %v", err)
	}
	if set.ID == "" || set.CreatedAt.IsZero() || set.UpdatedAt.IsZero() {
		t.Fatalf("CreateSet() did not populate identity and timestamps: %#v", set)
	}

	other := &domain.ReplySet{AccountID: "account-2", Name: "Other account"}
	if err := repo.CreateSet(ctx, other); err != nil {
		t.Fatalf("CreateSet(other) error = %v", err)
	}
	sets, err := repo.GetSetsByAccountID(ctx, "account-1")
	if err != nil {
		t.Fatalf("GetSetsByAccountID() error = %v", err)
	}
	if len(sets) != 1 || sets[0].ID != set.ID {
		t.Fatalf("GetSetsByAccountID() = %#v, want only %q", sets, set.ID)
	}

	items := []domain.ReplyItem{
		{Type: "image", Content: "https://example.com/product.jpg", OrderIndex: 99},
		{Type: "text", Content: "Price: {{product.price}}", OrderIndex: -1},
		{Type: "audio", Content: "https://example.com/details.mp3", OrderIndex: 5},
	}
	if err = repo.UpdateItems(ctx, set.ID, items); err != nil {
		t.Fatalf("UpdateItems() error = %v", err)
	}

	stored, err := repo.GetSetByID(ctx, set.ID)
	if err != nil {
		t.Fatalf("GetSetByID() error = %v", err)
	}
	if len(stored.Items) != len(items) {
		t.Fatalf("stored item count = %d, want %d", len(stored.Items), len(items))
	}
	for index, item := range stored.Items {
		if item.OrderIndex != index {
			t.Errorf("item %d OrderIndex = %d, want %d", index, item.OrderIndex, index)
		}
		if item.ID == "" || item.ReplySetID != set.ID {
			t.Errorf("item %d identity = %#v", index, item)
		}
		if item.Type != items[index].Type || item.Content != items[index].Content {
			t.Errorf("item %d = %#v, want type/content from %#v", index, item, items[index])
		}
	}

	// Reads must return a copy so callers cannot mutate repository state.
	stored.Items[0].Content = "mutated"
	again, err := repo.GetSetByID(ctx, set.ID)
	if err != nil {
		t.Fatalf("GetSetByID(second read) error = %v", err)
	}
	if again.Items[0].Content == "mutated" {
		t.Fatal("GetSetByID() leaked mutable repository state")
	}

	set.Name = "Updated reply"
	if err = repo.UpdateSet(ctx, set); err != nil {
		t.Fatalf("UpdateSet() error = %v", err)
	}
	updated, err := repo.GetSetByID(ctx, set.ID)
	if err != nil {
		t.Fatalf("GetSetByID(updated) error = %v", err)
	}
	if updated.Name != "Updated reply" || len(updated.Items) != len(items) {
		t.Fatalf("UpdateSet() lost data: %#v", updated)
	}

	if err = repo.DeleteSet(ctx, set.ID); err != nil {
		t.Fatalf("DeleteSet() error = %v", err)
	}
	if _, err = repo.GetSetByID(ctx, set.ID); err == nil {
		t.Fatal("GetSetByID() after delete returned no error")
	}
}

func TestMemoryReplySetMissingMutationsReturnError(t *testing.T) {
	ctx := context.Background()
	repo := NewMemory("")

	if err := repo.UpdateSet(ctx, &domain.ReplySet{ID: "missing", Name: "No set"}); err == nil {
		t.Fatal("UpdateSet(missing) returned no error")
	}
	if err := repo.UpdateItems(ctx, "missing", []domain.ReplyItem{{Type: "text", Content: "hello"}}); err == nil {
		t.Fatal("UpdateItems(missing) returned no error")
	}
	if err := repo.DeleteSet(ctx, "missing"); err == nil {
		t.Fatal("DeleteSet(missing) returned no error")
	}
}
