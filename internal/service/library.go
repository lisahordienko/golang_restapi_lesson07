package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"libratrack/internal/model"
	"libratrack/internal/repository"
)

var (
	ErrNotFound         = errors.New("resource not found")
	ErrInvalidReference = errors.New("invalid resource reference")
	ErrConflict         = errors.New("resource conflict")
)

type BookInput struct {
	Title      string
	ISBN       string
	CategoryID uint
	AuthorIDs  []uint
}

type Library struct {
	books      repository.BookRepository
	authors    repository.AuthorRepository
	categories repository.CategoryRepository
	copies     repository.CopyRepository
}

func NewLibrary(books repository.BookRepository, authors repository.AuthorRepository, categories repository.CategoryRepository, copies repository.CopyRepository) *Library {
	return &Library{books: books, authors: authors, categories: categories, copies: copies}
}

func (s *Library) Categories(ctx context.Context) ([]model.Category, error) {
	return s.categories.List(ctx)
}

func (s *Library) Category(ctx context.Context, id uint) (model.Category, error) {
	value, err := s.categories.FindByID(ctx, id)
	return value, translateError(err)
}

func (s *Library) CreateCategory(ctx context.Context, value *model.Category) error {
	return translateError(s.categories.Create(ctx, value))
}

func (s *Library) Authors(ctx context.Context) ([]model.Author, error) {
	return s.authors.List(ctx)
}

func (s *Library) Author(ctx context.Context, id uint) (model.Author, error) {
	value, err := s.authors.FindByID(ctx, id)
	return value, translateError(err)
}

func (s *Library) CreateAuthor(ctx context.Context, value *model.Author) error {
	return translateError(s.authors.Create(ctx, value))
}

func (s *Library) Books(ctx context.Context, categoryID, limit, offset uint) ([]model.Book, error) {
	return s.books.List(ctx, categoryID, limit, offset)
}

func (s *Library) Book(ctx context.Context, id uint) (*model.Book, error) {
	value, err := s.books.FindByID(ctx, id)
	return value, translateError(err)
}

func (s *Library) CreateBook(ctx context.Context, input BookInput) (*model.Book, error) {
	category, err := s.categoryReference(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}
	authors, err := s.authorReferences(ctx, input.AuthorIDs)
	if err != nil {
		return nil, err
	}
	book := &model.Book{Title: strings.TrimSpace(input.Title), ISBN: strings.TrimSpace(input.ISBN), CategoryID: category.ID, Category: category, Authors: authors}
	if err := translateError(s.books.Create(ctx, book)); err != nil {
		return nil, err
	}
	return s.Book(ctx, book.ID)
}

func (s *Library) UpdateBook(ctx context.Context, id uint, input BookInput) (*model.Book, error) {
	book, err := s.Book(ctx, id)
	if err != nil {
		return nil, err
	}
	category, err := s.categoryReference(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}
	authors, err := s.authorReferences(ctx, input.AuthorIDs)
	if err != nil {
		return nil, err
	}
	book.Title = strings.TrimSpace(input.Title)
	book.ISBN = strings.TrimSpace(input.ISBN)
	book.CategoryID = category.ID
	book.Category = category
	book.Authors = authors
	if err := translateError(s.books.Update(ctx, book)); err != nil {
		return nil, err
	}
	return s.Book(ctx, id)
}

func (s *Library) DeleteBook(ctx context.Context, id uint) error {
	return translateError(s.books.Delete(ctx, id))
}

func (s *Library) Copies(ctx context.Context, bookID uint) ([]model.Copy, error) {
	if _, err := s.Book(ctx, bookID); err != nil {
		return nil, err
	}
	return s.copies.ListByBookID(ctx, bookID)
}

func (s *Library) CreateCopy(ctx context.Context, value *model.Copy) error {
	if _, err := s.Book(ctx, value.BookID); err != nil {
		return err
	}
	return translateError(s.copies.Create(ctx, value))
}

func (s *Library) Availability(ctx context.Context, bookID uint) (int64, int64, error) {
	if _, err := s.Book(ctx, bookID); err != nil {
		return 0, 0, err
	}
	total, available, err := s.copies.CountByBookID(ctx, bookID)
	return total, available, translateError(err)
}

func (s *Library) categoryReference(ctx context.Context, id uint) (model.Category, error) {
	category, err := s.categories.FindByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Category{}, ErrInvalidReference
	}
	return category, translateError(err)
}

func (s *Library) authorReferences(ctx context.Context, ids []uint) ([]model.Author, error) {
	unique := uniqueIDs(ids)
	if len(unique) == 0 {
		return nil, ErrInvalidReference
	}
	authors, err := s.authors.FindByIDs(ctx, unique)
	if err != nil {
		return nil, translateError(err)
	}
	if len(authors) != len(unique) {
		return nil, ErrInvalidReference
	}
	return authors, nil
}

func uniqueIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

func translateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	message := strings.ToLower(fmt.Sprint(err))
	if strings.Contains(message, "duplicate key") || strings.Contains(message, "unique constraint") {
		return ErrConflict
	}
	return err
}
