package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"libratrack/internal/middleware"
	"libratrack/internal/model"
	"libratrack/internal/service"
)

const maxBodySize = 1 << 20

type LibraryAPI struct {
	library *service.Library
}

func NewLibraryAPI(library *service.Library) http.Handler {
	api := &LibraryAPI{library: library}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/swagger/index.html", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<!doctype html><title>LibraTrack API</title><h1>LibraTrack API</h1>")
	})
	mux.HandleFunc("/api/v1/categories", api.categories)
	mux.HandleFunc("/api/v1/categories/", api.category)
	mux.HandleFunc("/api/v1/authors", api.authors)
	mux.HandleFunc("/api/v1/authors/", api.author)
	mux.HandleFunc("/api/v1/books", api.booksCollection)
	mux.HandleFunc("/api/v1/books/", api.book)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "Resource not found")
	})
	return middleware.RequestLogger(slog.Default())(middleware.CORS(middleware.Recoverer(mux)))
}

func (a *LibraryAPI) categories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		values, err := a.library.Categories(r.Context())
		if err != nil {
			writeInternalError(w)
			return
		}
		writeJSON(w, http.StatusOK, values)
	case http.MethodPost:
		var input CategoryRequest
		if !decodeJSON(w, r, &input) || strings.TrimSpace(input.Name) == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name is required")
			return
		}
		value := model.Category{Name: strings.TrimSpace(input.Name)}
		if err := a.library.CreateCategory(r.Context(), &value); err != nil {
			writeServiceError(w, err, "Category already exists")
			return
		}
		writeJSON(w, http.StatusCreated, value)
	default:
		methodNotAllowed(w)
	}
}

func (a *LibraryAPI) category(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	id, ok := pathID(r.URL.Path, "/api/v1/categories/")
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Category not found")
		return
	}
	value, err := a.library.Category(r.Context(), id)
	if err != nil {
		writeServiceError(w, err, "Category not found")
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (a *LibraryAPI) authors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		values, err := a.library.Authors(r.Context())
		if err != nil {
			writeInternalError(w)
			return
		}
		writeJSON(w, http.StatusOK, values)
	case http.MethodPost:
		var input AuthorRequest
		if !decodeJSON(w, r, &input) || strings.TrimSpace(input.FullName) == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "full_name is required")
			return
		}
		value := model.Author{FullName: strings.TrimSpace(input.FullName), Bio: input.Bio}
		if err := a.library.CreateAuthor(r.Context(), &value); err != nil {
			writeServiceError(w, err, "Author could not be created")
			return
		}
		writeJSON(w, http.StatusCreated, value)
	default:
		methodNotAllowed(w)
	}
}

func (a *LibraryAPI) author(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	id, ok := pathID(r.URL.Path, "/api/v1/authors/")
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Author not found")
		return
	}
	value, err := a.library.Author(r.Context(), id)
	if err != nil {
		writeServiceError(w, err, "Author not found")
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (a *LibraryAPI) booksCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.listBooks(w, r)
	case http.MethodPost:
		a.createBook(w, r)
	default:
		methodNotAllowed(w)
	}
}

type bookInput = BookRequest

func (a *LibraryAPI) createBook(w http.ResponseWriter, r *http.Request) {
	var input bookInput
	if !decodeJSON(w, r, &input) || !validBookInput(input) {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "title, isbn, category_id, and author_ids are required")
		return
	}
	book, err := a.library.CreateBook(r.Context(), service.BookInput{Title: input.Title, ISBN: input.ISBN, CategoryID: input.CategoryID, AuthorIDs: input.AuthorIDs})
	if err != nil {
		writeServiceError(w, err, "Book already exists")
		return
	}
	writeJSON(w, http.StatusCreated, book)
}

func (a *LibraryAPI) listBooks(w http.ResponseWriter, r *http.Request) {
	page, limit, ok := pagination(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_pagination", "page and limit must be positive integers")
		return
	}
	var categoryID uint
	if raw := r.URL.Query().Get("category_id"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_pagination", "category_id must be a positive integer")
			return
		}
		categoryID = uint(parsed)
	}
	if page-1 > uint64(^uint(0)>>1)/limit {
		writeJSON(w, http.StatusOK, []model.Book{})
		return
	}
	offset := uint((page - 1) * limit)
	values, err := a.library.Books(r.Context(), categoryID, uint(limit), offset)
	if err != nil {
		writeInternalError(w)
		return
	}
	writeJSON(w, http.StatusOK, values)
}

func (a *LibraryAPI) book(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/books/")
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[1] == "copies" {
		a.copies(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "availability" {
		a.availability(w, r, parts[0])
		return
	}
	if len(parts) != 1 {
		writeError(w, http.StatusNotFound, "not_found", "Book not found")
		return
	}
	id, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "invalid_id", "ID must be an unsigned integer")
		return
	}
	switch r.Method {
	case http.MethodGet:
		value, err := a.library.Book(r.Context(), uint(id))
		if err != nil {
			writeServiceError(w, err, "Book not found")
			return
		}
		writeJSON(w, http.StatusOK, value)
	case http.MethodPut:
		a.updateBook(w, r, uint(id))
	case http.MethodDelete:
		if err := a.library.DeleteBook(r.Context(), uint(id)); err != nil {
			writeServiceError(w, err, "Book not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w)
	}
}

func (a *LibraryAPI) updateBook(w http.ResponseWriter, r *http.Request, id uint) {
	var input bookInput
	if !decodeJSON(w, r, &input) || !validBookInput(input) {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "title, isbn, category_id, and author_ids are required")
		return
	}
	updated, err := a.library.UpdateBook(r.Context(), id, service.BookInput{Title: input.Title, ISBN: input.ISBN, CategoryID: input.CategoryID, AuthorIDs: input.AuthorIDs})
	if err != nil {
		writeServiceError(w, err, "Book already exists")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *LibraryAPI) copies(w http.ResponseWriter, r *http.Request, rawID string) {
	bookID, err := strconv.ParseUint(rawID, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "ID must be an unsigned integer")
		return
	}
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		values, err := a.library.Copies(ctx, uint(bookID))
		if err != nil {
			writeServiceError(w, err, "Book not found")
			return
		}
		if values == nil {
			writeInternalError(w)
			return
		}
		writeJSON(w, http.StatusOK, values)
	case http.MethodPost:
		var input CopyRequest
		if !decodeJSON(w, r, &input) || strings.TrimSpace(input.InventoryNo) == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "inventory_no is required")
			return
		}
		if input.Status == "" {
			input.Status = "available"
		}
		if !validCopyStatus(input.Status) {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "Invalid copy status")
			return
		}
		value := model.Copy{BookID: uint(bookID), InventoryNo: strings.TrimSpace(input.InventoryNo), Status: input.Status}
		if err := a.library.CreateCopy(ctx, &value); err != nil {
			writeServiceError(w, err, "Inventory number already exists")
			return
		}
		writeJSON(w, http.StatusCreated, value)
	default:
		methodNotAllowed(w)
	}
}

func (a *LibraryAPI) availability(w http.ResponseWriter, r *http.Request, rawID string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	bookID, err := strconv.ParseUint(rawID, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "ID must be an unsigned integer")
		return
	}
	total, available, err := a.library.Availability(r.Context(), uint(bookID))
	if err != nil {
		writeServiceError(w, err, "Book not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"book_id": bookID, "total_copies": total, "available_copies": available})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		return false
	}
	var extra any
	return decoder.Decode(&extra) == io.EOF
}

func validBookInput(input bookInput) bool {
	return strings.TrimSpace(input.Title) != "" && strings.TrimSpace(input.ISBN) != "" && input.CategoryID > 0 && len(input.AuthorIDs) > 0
}

func pagination(r *http.Request) (uint64, uint64, bool) {
	page, limit := uint64(1), uint64(10)
	if raw := r.URL.Query().Get("page"); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || value == 0 {
			return 0, 0, false
		}
		page = value
	} else if _, ok := r.URL.Query()["page"]; ok {
		return 0, 0, false
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || value == 0 {
			return 0, 0, false
		}
		limit = value
	} else if _, ok := r.URL.Query()["limit"]; ok {
		return 0, 0, false
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit, true
}

func pathID(path, prefix string) (uint, bool) {
	raw := strings.TrimPrefix(path, prefix)
	value, err := strconv.ParseUint(raw, 10, 32)
	return uint(value), err == nil && raw != ""
}

func validCopyStatus(status string) bool {
	return status == "available" || status == "borrowed" || status == "lost"
}

func writeServiceError(w http.ResponseWriter, err error, conflictMessage string) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Resource not found")
	case errors.Is(err, service.ErrInvalidReference):
		writeError(w, http.StatusUnprocessableEntity, "invalid_reference", "One or more referenced resources do not exist")
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", conflictMessage)
	default:
		writeInternalError(w)
	}
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
}

func writeInternalError(w http.ResponseWriter) {
	writeError(w, http.StatusInternalServerError, "internal_server_error", "Internal server error")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	WriteError(w, status, code, message)
}
