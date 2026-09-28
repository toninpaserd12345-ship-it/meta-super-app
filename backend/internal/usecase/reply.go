package usecase

import (
	"context"
	"fmt"

	"github.com/meta-super-app/backend/internal/domain"
)

type Reply struct {
	repo domain.ReplyRepository
}

func NewReply(repo domain.ReplyRepository) *Reply {
	return &Reply{repo: repo}
}

func (u *Reply) CreateSet(ctx context.Context, accountID, name string) (*domain.ReplySet, error) {
	if name == "" {
		return nil, fmt.Errorf("set name cannot be empty")
	}

	set := &domain.ReplySet{
		AccountID: accountID,
		Name:      name,
	}

	if err := u.repo.CreateSet(ctx, set); err != nil {
		return nil, err
	}

	return set, nil
}

func (u *Reply) GetSets(ctx context.Context, accountID string) ([]domain.ReplySet, error) {
	return u.repo.GetSetsByAccountID(ctx, accountID)
}

func (u *Reply) GetSet(ctx context.Context, setID string, accountID string) (*domain.ReplySet, error) {
	set, err := u.repo.GetSetByID(ctx, setID)
	if err != nil {
		return nil, err
	}
	
	// Ensure the set belongs to this account
	if set.AccountID != accountID {
		return nil, fmt.Errorf("unauthorized to access this set")
	}
	
	return set, nil
}

func (u *Reply) UpdateSet(ctx context.Context, setID string, accountID string, name string) error {
	set, err := u.GetSet(ctx, setID, accountID)
	if err != nil {
		return err
	}

	set.Name = name
	return u.repo.UpdateSet(ctx, set)
}

func (u *Reply) DeleteSet(ctx context.Context, setID string, accountID string) error {
	_, err := u.GetSet(ctx, setID, accountID) // check permission
	if err != nil {
		return err
	}

	return u.repo.DeleteSet(ctx, setID)
}

func (u *Reply) UpdateItems(ctx context.Context, setID string, accountID string, items []domain.ReplyItem) error {
	_, err := u.GetSet(ctx, setID, accountID) // check permission
	if err != nil {
		return err
	}

	// Validate items
	for i, item := range items {
		if item.Type != "text" && item.Type != "image" && item.Type != "video" && item.Type != "audio" {
			return fmt.Errorf("invalid item type at index %d", i)
		}
		if item.Content == "" {
			return fmt.Errorf("empty content at index %d", i)
		}
	}

	return u.repo.UpdateItems(ctx, setID, items)
}
