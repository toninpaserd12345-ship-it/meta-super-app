package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
	"github.com/meta-super-app/backend/internal/infrastructure/database"
	"gorm.io/gorm"
)

// ensure Gorm implements ReplyRepository (to catch interface mismatches at compile time)
// var _ domain.ReplyRepository = (*Gorm)(nil)

func (r *Gorm) CreateSet(ctx context.Context, set *domain.ReplySet) error {
	id := uuid.New().String()
	set.ID = id
	model := database.ReplySetModel{
		ID:        id,
		AccountID: set.AccountID,
		Name:      set.Name,
	}
	
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}
	
	set.CreatedAt = model.CreatedAt
	set.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *Gorm) GetSetsByAccountID(ctx context.Context, accountID string) ([]domain.ReplySet, error) {
	var models []database.ReplySetModel
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_index ASC")
	}).Find(&models).Error; err != nil {
		return nil, err
	}

	var sets []domain.ReplySet
	for _, m := range models {
		sets = append(sets, toDomainReplySet(m))
	}
	return sets, nil
}

func (r *Gorm) GetSetByID(ctx context.Context, setID string) (*domain.ReplySet, error) {
	var model database.ReplySetModel
	if err := r.db.WithContext(ctx).Where("id = ?", setID).Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_index ASC")
	}).First(&model).Error; err != nil {
		return nil, err
	}

	set := toDomainReplySet(model)
	return &set, nil
}

func (r *Gorm) UpdateSet(ctx context.Context, set *domain.ReplySet) error {
	return r.db.WithContext(ctx).Model(&database.ReplySetModel{}).Where("id = ?", set.ID).Update("name", set.Name).Error
}

func (r *Gorm) DeleteSet(ctx context.Context, setID string) error {
	// Cascade delete handles items
	return r.db.WithContext(ctx).Where("id = ?", setID).Delete(&database.ReplySetModel{}).Error
}

func (r *Gorm) UpdateItems(ctx context.Context, setID string, items []domain.ReplyItem) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 1. Delete all existing items for this set
	if err := tx.Where("reply_set_id = ?", setID).Delete(&database.ReplyItemModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. Insert new items
	if len(items) > 0 {
		var models []database.ReplyItemModel
		for _, i := range items {
			models = append(models, database.ReplyItemModel{
				ID:         uuid.New().String(),
				ReplySetID: setID,
				Type:       i.Type,
				Content:    i.Content,
				OrderIndex: i.OrderIndex,
			})
		}
		if err := tx.Create(&models).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func toDomainReplySet(m database.ReplySetModel) domain.ReplySet {
	set := domain.ReplySet{
		ID:        m.ID,
		AccountID: m.AccountID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	
	for _, i := range m.Items {
		set.Items = append(set.Items, domain.ReplyItem{
			ID:         i.ID,
			ReplySetID: i.ReplySetID,
			Type:       i.Type,
			Content:    i.Content,
			OrderIndex: i.OrderIndex,
		})
	}
	return set
}
