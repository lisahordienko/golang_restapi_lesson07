package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"libratrack/internal/model"
	"libratrack/internal/service"
)

type LibraryHandler struct {
	library *service.Library
}

func NewLibraryHandler(library *service.Library) *LibraryHandler {
	return &LibraryHandler{library: library}
}

func (h *LibraryHandler) ListCategories(c *gin.Context) {
	values, err := h.library.Categories(c.Request.Context())
	if err != nil {
		abortInternal(c)
		return
	}
	c.JSON(http.StatusOK, values)
}

func (h *LibraryHandler) GetCategory(c *gin.Context) {
	id, ok := parseParamID(c, "id")
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_id", "ID must be an unsigned integer"))
		return
	}
	value, err := h.library.Category(c.Request.Context(), id)
	if err != nil {
		abortServiceError(c, err, "Category not found")
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *LibraryHandler) CreateCategory(c *gin.Context) {
	var input CategoryRequest
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Name) == "" {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "name is required"))
		return
	}
	value := model.Category{Name: strings.TrimSpace(input.Name)}
	if err := h.library.CreateCategory(c.Request.Context(), &value); err != nil {
		abortServiceError(c, err, "Category already exists")
		return
	}
	c.JSON(http.StatusCreated, value)
}

func (h *LibraryHandler) ListAuthors(c *gin.Context) {
	values, err := h.library.Authors(c.Request.Context())
	if err != nil {
		abortInternal(c)
		return
	}
	c.JSON(http.StatusOK, values)
}

func (h *LibraryHandler) GetAuthor(c *gin.Context) {
	id, ok := parseParamID(c, "id")
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_id", "ID must be an unsigned integer"))
		return
	}
	value, err := h.library.Author(c.Request.Context(), id)
	if err != nil {
		abortServiceError(c, err, "Author not found")
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *LibraryHandler) CreateAuthor(c *gin.Context) {
	var input AuthorRequest
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.FullName) == "" {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "full_name is required"))
		return
	}
	value := model.Author{FullName: strings.TrimSpace(input.FullName), Bio: input.Bio}
	if err := h.library.CreateAuthor(c.Request.Context(), &value); err != nil {
		abortServiceError(c, err, "Author could not be created")
		return
	}
	c.JSON(http.StatusCreated, value)
}

func (h *LibraryHandler) ListBooks(c *gin.Context) {
	page, limit, ok := parseGinPagination(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_pagination", "page and limit must be positive integers"))
		return
	}
	var categoryID uint
	if raw := c.Query("category_id"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_pagination", "category_id must be a positive integer"))
			return
		}
		categoryID = uint(parsed)
	}
	if page-1 > uint64(^uint(0)>>1)/limit {
		c.JSON(http.StatusOK, []model.Book{})
		return
	}
	offset := uint((page - 1) * limit)
	values, err := h.library.Books(c.Request.Context(), categoryID, uint(limit), offset)
	if err != nil {
		abortInternal(c)
		return
	}
	c.JSON(http.StatusOK, values)
}

func (h *LibraryHandler) GetBook(c *gin.Context) {
	id, ok := parseParamID(c, "id")
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_id", "ID must be an unsigned integer"))
		return
	}
	value, err := h.library.Book(c.Request.Context(), id)
	if err != nil {
		abortServiceError(c, err, "Book not found")
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *LibraryHandler) CreateBook(c *gin.Context) {
	var input BookRequest
	if err := c.ShouldBindJSON(&input); err != nil || !validBookInput(input) {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "title, isbn, category_id, and author_ids are required"))
		return
	}
	book, err := h.library.CreateBook(c.Request.Context(), service.BookInput{
		Title: input.Title, ISBN: input.ISBN, CategoryID: input.CategoryID, AuthorIDs: input.AuthorIDs,
	})
	if err != nil {
		abortServiceError(c, err, "Book already exists")
		return
	}
	c.JSON(http.StatusCreated, book)
}

func (h *LibraryHandler) UpdateBook(c *gin.Context) {
	id, ok := parseParamID(c, "id")
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_id", "ID must be an unsigned integer"))
		return
	}
	var input BookRequest
	if err := c.ShouldBindJSON(&input); err != nil || !validBookInput(input) {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "title, isbn, category_id, and author_ids are required"))
		return
	}
	updated, err := h.library.UpdateBook(c.Request.Context(), id, service.BookInput{
		Title: input.Title, ISBN: input.ISBN, CategoryID: input.CategoryID, AuthorIDs: input.AuthorIDs,
	})
	if err != nil {
		abortServiceError(c, err, "Book already exists")
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *LibraryHandler) DeleteBook(c *gin.Context) {
	id, ok := parseParamID(c, "id")
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_id", "ID must be an unsigned integer"))
		return
	}
	if err := h.library.DeleteBook(c.Request.Context(), id); err != nil {
		abortServiceError(c, err, "Book not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *LibraryHandler) ListCopies(c *gin.Context) {
	bookID, ok := parseParamID(c, "id")
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_id", "ID must be an unsigned integer"))
		return
	}
	values, err := h.library.Copies(c.Request.Context(), bookID)
	if err != nil {
		abortServiceError(c, err, "Book not found")
		return
	}
	if values == nil {
		abortInternal(c)
		return
	}
	c.JSON(http.StatusOK, values)
}

func (h *LibraryHandler) CreateCopy(c *gin.Context) {
	bookID, ok := parseParamID(c, "id")
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_id", "ID must be an unsigned integer"))
		return
	}
	var input CopyRequest
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.InventoryNo) == "" {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "inventory_no is required"))
		return
	}
	if input.Status == "" {
		input.Status = "available"
	}
	if !validCopyStatus(input.Status) {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "Invalid copy status"))
		return
	}
	value := model.Copy{BookID: bookID, InventoryNo: strings.TrimSpace(input.InventoryNo), Status: input.Status}
	if err := h.library.CreateCopy(c.Request.Context(), &value); err != nil {
		abortServiceError(c, err, "Inventory number already exists")
		return
	}
	c.JSON(http.StatusCreated, value)
}

func (h *LibraryHandler) Availability(c *gin.Context) {
	bookID, ok := parseParamID(c, "id")
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_id", "ID must be an unsigned integer"))
		return
	}
	total, available, err := h.library.Availability(c.Request.Context(), bookID)
	if err != nil {
		abortServiceError(c, err, "Book not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"book_id": bookID, "total_copies": total, "available_copies": available})
}

func parseParamID(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	if raw == "" || strings.ContainsAny(raw, "+-") {
		return 0, false
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	return uint(value), err == nil && value > 0
}

func parseGinPagination(c *gin.Context) (uint64, uint64, bool) {
	page, limit := uint64(1), uint64(10)
	if raw, present := c.GetQuery("page"); present {
		if raw == "" {
			return 0, 0, false
		}
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || value == 0 {
			return 0, 0, false
		}
		page = value
	}
	if raw, present := c.GetQuery("limit"); present {
		if raw == "" {
			return 0, 0, false
		}
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || value == 0 {
			return 0, 0, false
		}
		limit = value
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit, true
}

func abortServiceError(c *gin.Context, err error, conflictMessage string) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, apiError("not_found", "Resource not found"))
	case errors.Is(err, service.ErrInvalidReference):
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("invalid_reference", "One or more referenced resources do not exist"))
	case errors.Is(err, service.ErrConflict):
		c.AbortWithStatusJSON(http.StatusConflict, apiError("conflict", conflictMessage))
	default:
		abortInternal(c)
	}
}

func abortInternal(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, apiError("internal_server_error", "Internal server error"))
}
