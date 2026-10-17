package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"libratrack/internal/model"
)

type BookRepository interface {
	Create(context.Context, *model.Book) error
	FindByID(context.Context, uint) (*model.Book, error)
	Update(context.Context, *model.Book) error
	Delete(context.Context, uint) error
	List(context.Context, uint, uint, uint) ([]model.Book, error)
	Search(context.Context, string, uint) ([]model.Book, error)
}

type gormBookRepository struct {
	db *gorm.DB
}

func NewGormBookRepository(db *gorm.DB) BookRepository {
	return &gormBookRepository{db: db}
}

func (r *gormBookRepository) Create(ctx context.Context, book *model.Book) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Authors", "Category", "Copies").Create(book).Error; err != nil {
			return err
		}
		return tx.Model(book).Association("Authors").Replace(book.Authors)
	})
}

func (r *gormBookRepository) FindByID(ctx context.Context, id uint) (*model.Book, error) {
	var book model.Book
	err := r.db.WithContext(ctx).Preload("Authors").Preload("Category").First(&book, id).Error
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *gormBookRepository) Update(ctx context.Context, book *model.Book) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Book{}).Where("id = ?", book.ID).Updates(map[string]any{
			"title": book.Title, "isbn": book.ISBN, "category_id": book.CategoryID,
		}).Error; err != nil {
			return err
		}
		return tx.Model(book).Association("Authors").Replace(book.Authors)
	})
}

func (r *gormBookRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Book{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormBookRepository) List(ctx context.Context, categoryID, limit, offset uint) ([]model.Book, error) {
	books := make([]model.Book, 0)
	query := r.db.WithContext(ctx).Joins("Category").Preload("Authors").Preload("Category").Order("id ASC")
	if categoryID != 0 {
		query = query.Where("category_id = ?", categoryID)
	}
	err := query.Limit(int(limit)).Offset(int(offset)).Find(&books).Error
	return books, err
}

func (r *gormBookRepository) Search(ctx context.Context, query string, limit uint) ([]model.Book, error) {
	books := make([]model.Book, 0)
	pattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	err := r.db.WithContext(ctx).
		Joins("Category").
		Preload("Authors").
		Preload("Category").
		Preload("Copies").
		Where("LOWER(books.title) LIKE ? OR LOWER(books.isbn) LIKE ? OR books.id IN (SELECT book_id FROM book_authors JOIN authors ON authors.id = book_authors.author_id WHERE LOWER(authors.full_name) LIKE ?)", pattern, pattern, pattern).
		Order("books.id ASC").
		Limit(int(limit)).
		Find(&books).Error
	return books, err
}
