package usecase_test

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/meta-super-app/backend/internal/infrastructure/repository"
	"github.com/meta-super-app/backend/internal/usecase"
)

func TestReplySetCodeIsNormalizedAndUniquePerWorkspace(t *testing.T) {
	ctx := context.Background()
	service := usecase.NewReply(repository.NewMemory(""))

	created, err := service.CreateSet(ctx, "account-1", " price-durian_01 ", "Durian price")
	if err != nil {
		t.Fatalf("CreateSet() error = %v", err)
	}
	if created.Code != "PRICE-DURIAN_01" {
		t.Fatalf("CreateSet() code = %q, want normalized code", created.Code)
	}

	if _, err = service.CreateSet(ctx, "account-1", "price-durian_01", "Duplicate"); !errors.Is(err, usecase.ErrReplySetCodeExists) {
		t.Fatalf("CreateSet(duplicate) error = %v, want ErrReplySetCodeExists", err)
	}
	if _, err = service.CreateSet(ctx, "account-2", "price-durian_01", "Other workspace"); err != nil {
		t.Fatalf("CreateSet(same code in another workspace) error = %v", err)
	}
	if _, err = service.CreateSet(ctx, "account-1", "bad code!", "Invalid"); !errors.Is(err, usecase.ErrInvalidReplySetCode) {
		t.Fatalf("CreateSet(invalid code) error = %v, want ErrInvalidReplySetCode", err)
	}
	found, err := service.FindSetByTrigger(ctx, "account-1", "  durian PRICE ")
	if err != nil || found == nil || found.ID != created.ID {
		t.Fatalf("FindSetByTrigger() = %#v, %v; want %q", found, err, created.ID)
	}
	notFound, err := service.FindSetByTrigger(ctx, "account-1", "please send Durian price")
	if err != nil || notFound != nil {
		t.Fatalf("FindSetByTrigger(partial sentence) = %#v, %v; want nil exact-match result", notFound, err)
	}
	if _, err = service.CreateSet(ctx, "account-1", "ANOTHER-CODE", "durian price"); !errors.Is(err, usecase.ErrReplySetNameExists) {
		t.Fatalf("CreateSet(duplicate name) error = %v, want ErrReplySetNameExists", err)
	}
	otherStore, err := service.CreateSet(ctx, "account-2", "STORE-2-DURIAN", "Durian price")
	if err != nil {
		t.Fatalf("CreateSet(same name in another workspace) error = %v", err)
	}
	otherFound, err := service.FindSetByTrigger(ctx, "account-2", "durian price")
	if err != nil || otherFound == nil || otherFound.ID != otherStore.ID {
		t.Fatalf("FindSetByTrigger(other workspace) = %#v, %v; want %q", otherFound, err, otherStore.ID)
	}
	if otherFound.ID == created.ID {
		t.Fatal("FindSetByTrigger leaked a Reply Set from another workspace")
	}
}

func TestReplySetGeneratesCodeForLegacyClient(t *testing.T) {
	service := usecase.NewReply(repository.NewMemory(""))
	created, err := service.CreateSet(context.Background(), "account-1", "", "ລາຄາສິນຄ້າ")
	if err != nil {
		t.Fatalf("CreateSet() error = %v", err)
	}
	if created.Code == "" || !regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{1,59}$`).MatchString(created.Code) {
		t.Fatalf("generated code %q is invalid", created.Code)
	}
}
