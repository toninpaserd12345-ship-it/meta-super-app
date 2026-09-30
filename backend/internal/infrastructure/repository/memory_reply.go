package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
)

// Ensure MemoryRepository implements ReplyRepository
// var _ domain.ReplyRepository = (*MemoryRepository)(nil)

func (r *MemoryRepository) CreateSet(ctx context.Context, set *domain.ReplySet) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := uuid.New().String()
	set.ID = id

	now := time.Now()
	set.CreatedAt, set.UpdatedAt = now, now
	r.replySets[id] = cloneReplySet(*set)
	return nil
}

func (r *MemoryRepository) GetSetsByAccountID(ctx context.Context, accountID string) ([]domain.ReplySet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sets := make([]domain.ReplySet, 0)
	for _, set := range r.replySets {
		if set.AccountID == accountID {
			sets = append(sets, cloneReplySet(set))
		}
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].UpdatedAt.After(sets[j].UpdatedAt) })
	return sets, nil
}

func (r *MemoryRepository) GetSetByID(ctx context.Context, setID string) (*domain.ReplySet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	set, ok := r.replySets[setID]
	if !ok {
		return nil, errors.New("reply set not found")
	}
	copy := cloneReplySet(set)
	return &copy, nil
}

func (r *MemoryRepository) UpdateSet(ctx context.Context, set *domain.ReplySet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.replySets[set.ID]
	if !ok {
		return errors.New("reply set not found")
	}
	current.Name, current.UpdatedAt = set.Name, time.Now()
	r.replySets[set.ID] = current
	return nil
}

func (r *MemoryRepository) DeleteSet(ctx context.Context, setID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.replySets[setID]; !ok {
		return errors.New("reply set not found")
	}
	delete(r.replySets, setID)
	return nil
}

func (r *MemoryRepository) UpdateItems(ctx context.Context, setID string, items []domain.ReplyItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	set, ok := r.replySets[setID]
	if !ok {
		return errors.New("reply set not found")
	}
	set.Items = make([]domain.ReplyItem, len(items))
	for index, item := range items {
		item.ID = uuid.New().String()
		item.ReplySetID = setID
		item.OrderIndex = index
		set.Items[index] = item
	}
	set.UpdatedAt = time.Now()
	r.replySets[setID] = set
	return nil
}

func cloneReplySet(set domain.ReplySet) domain.ReplySet {
	set.Items = append([]domain.ReplyItem(nil), set.Items...)
	return set
}
