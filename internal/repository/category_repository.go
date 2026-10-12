package repository

import (
	"context"

	"gorm.io/gorm"

	"libratrack/internal/model"
)

type CategoryRepository interface {
	List(context.Context) ([]model.Category, error)
	FindByID(context.Context, uint) (model.Category, error)
	Create(context.Context, *model.Category) error
}

type gormCategoryRepository struct {
	db *gorm.DB
}

func NewGormCategoryRepository(db *gorm.DB) CategoryRepository {
	return &gormCategoryRepository{db: db}
}

func (r *gormCategoryRepository) List(ctx context.Context) ([]model.Category, error) {
	values := make([]model.Category, 0)
	err := r.db.WithContext(ctx).Order("id ASC").Find(&values).Error
	return values, err
}

func (r *gormCategoryRepository) FindByID(ctx context.Context, id uint) (model.Category, error) {
	var category model.Category
	err := r.db.WithContext(ctx).First(&category, id).Error
	return category, err
}

func (r *gormCategoryRepository) Create(ctx context.Context, category *model.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}
