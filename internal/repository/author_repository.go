package repository

import (
	"context"

	"gorm.io/gorm"

	"libratrack/internal/model"
)

type AuthorRepository interface {
	List(context.Context) ([]model.Author, error)
	FindByID(context.Context, uint) (model.Author, error)
	FindByIDs(context.Context, []uint) ([]model.Author, error)
	Create(context.Context, *model.Author) error
}

type gormAuthorRepository struct {
	db *gorm.DB
}

func NewGormAuthorRepository(db *gorm.DB) AuthorRepository {
	return &gormAuthorRepository{db: db}
}

func (r *gormAuthorRepository) List(ctx context.Context) ([]model.Author, error) {
	values := make([]model.Author, 0)
	err := r.db.WithContext(ctx).Order("id ASC").Find(&values).Error
	return values, err
}

func (r *gormAuthorRepository) FindByID(ctx context.Context, id uint) (model.Author, error) {
	var author model.Author
	err := r.db.WithContext(ctx).First(&author, id).Error
	return author, err
}

func (r *gormAuthorRepository) FindByIDs(ctx context.Context, ids []uint) ([]model.Author, error) {
	var authors []model.Author
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Order("id ASC").Find(&authors).Error
	return authors, err
}

func (r *gormAuthorRepository) Create(ctx context.Context, author *model.Author) error {
	return r.db.WithContext(ctx).Create(author).Error
}
