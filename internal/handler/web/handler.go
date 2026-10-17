package web

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"libratrack/internal/auth"
	"libratrack/internal/middleware"
	"libratrack/internal/model"
	"libratrack/internal/service"
)

const sessionCookieName = "libratrack_session"

type Handler struct {
	library *service.Library
	users   interface {
		User(context.Context, uint) (model.User, error)
	}
	templates *template.Template
}

type catalogPage struct {
	Books     []catalogRow
	Count     int
	CanDelete bool
	Email     string
	Role      string
	SignedIn  bool
}

type catalogRow struct {
	model.Book
	Total     int
	Available int
	CanDelete bool
}

func New(library *service.Library, users interface {
	User(context.Context, uint) (model.User, error)
}) (*Handler, error) {
	parsed, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	return &Handler{library: library, users: users, templates: parsed}, nil
}

func (h *Handler) Register(router *gin.Engine) error {
	staticFS, err := assets()
	if err != nil {
		return err
	}
	router.StaticFS("/static", http.FS(staticFS))
	router.GET("/catalog", h.Catalog)
	router.GET("/web/books", h.Search)
	router.POST("/web/logout", middleware.RequireSameOrigin(), h.Logout)
	router.DELETE("/web/books/:id", middleware.CookieAuthMiddleware(), middleware.RequireRole("librarian", "admin"), middleware.RequireSameOrigin(), h.Delete)
	return nil
}

func (h *Handler) Catalog(c *gin.Context) {
	rows, err := h.search(c, "")
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to load catalog")
		return
	}
	page := catalogPage{Books: rows, Count: len(rows)}
	if claims, err := tokenFromRequest(c.Request); err == nil {
		if user, userErr := h.users.User(c.Request.Context(), claims.UserID); userErr == nil {
			page.SignedIn = true
			page.Email = user.Email
			page.Role = user.Role
			page.CanDelete = user.Role == "librarian" || user.Role == "admin"
		}
	}
	for index := range rows {
		rows[index].CanDelete = page.CanDelete
	}
	page.Books = rows
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(c.Writer, "catalog", page); err != nil {
		c.String(http.StatusInternalServerError, "Unable to render catalog")
	}
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, "", -1, "/", "", c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https", true)
	c.Redirect(http.StatusSeeOther, "/catalog")
}

func (h *Handler) Search(c *gin.Context) {
	rows, err := h.search(c, c.Query("q"))
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to search catalog")
		return
	}
	claims, _ := tokenFromRequest(c.Request)
	canDelete := claims != nil && (claims.Role == "librarian" || claims.Role == "admin")
	for _, row := range rows {
		row.CanDelete = canDelete
		if err := h.templates.ExecuteTemplate(c.Writer, "book_row", row); err != nil {
			return
		}
	}
	if len(rows) == 0 {
		_, _ = c.Writer.Write([]byte(`<tr><td class="empty" colspan="6">No books match this search.</td></tr>`))
	}
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := h.library.DeleteBook(c.Request.Context(), uint(id)); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) search(c *gin.Context, query string) ([]catalogRow, error) {
	books, err := h.library.SearchBooks(c.Request.Context(), strings.TrimSpace(query), 100)
	if err != nil {
		return nil, err
	}
	rows := make([]catalogRow, 0, len(books))
	for _, book := range books {
		row := catalogRow{Book: book, Total: len(book.Copies)}
		for _, copy := range book.Copies {
			if copy.Status == "available" {
				row.Available++
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func tokenFromRequest(request *http.Request) (*auth.Claims, error) {
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}
	return auth.ParseToken(cookie.Value)
}
