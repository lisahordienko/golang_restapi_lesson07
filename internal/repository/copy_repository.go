package repository

import (
	"context"

	"gorm.io/gorm"

	"libratrack/internal/model"
)

type CopyRepository interface {
	ListByBookID(context.Context, uint) ([]model.Copy, error)
	Create(context.Context, *model.Copy) error
	CountByBookID(context.Context, uint) (int64, int64, error)
}

type gormCopyRepository struct {
	db *gorm.DB
}

func NewGormCopyRepository(db *gorm.DB) CopyRepository {
	return &gormCopyRepository{db: db}
}

func (r *gormCopyRepository) ListByBookID(ctx context.Context, bookID uint) ([]model.Copy, error) {
	values := make([]model.Copy, 0)
	err := r.db.WithContext(ctx).Where("book_id = ?", bookID).Order("id ASC").Find(&values).Error
	return values, err
}

func (r *gormCopyRepository) Create(ctx context.Context, value *model.Copy) error {
	return r.db.WithContext(ctx).Create(value).Error
}

func (r *gormCopyRepository) CountByBookID(ctx context.Context, bookID uint) (int64, int64, error) {
	var total, available int64
	query := r.db.WithContext(ctx).Model(&model.Copy{}).Where("book_id = ?", bookID)
	if err := query.Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if err := query.Where("status = ?", "available").Count(&available).Error; err != nil {
		return 0, 0, err
	}
	return total, available, nil
}
