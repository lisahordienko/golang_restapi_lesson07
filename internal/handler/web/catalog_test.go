package web_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"libratrack/internal/app"
	"libratrack/internal/auth"
	"libratrack/internal/model"
)

func TestCatalogWebIntegration(t *testing.T) {
	t.Setenv("JWT_SECRET", "catalog-test-secret-with-enough-entropy")
	gin.SetMode(gin.TestMode)
	server, db := newCatalogServer(t)
	defer server.Close()
	defer func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}()

	readerToken, err := auth.IssueToken(model.User{ID: 1, Role: "reader"})
	if err != nil {
		t.Fatalf("issue reader token: %v", err)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		cookie     string
		hxRequest  bool
		wantStatus int
		contains   string
		notContain string
	}{
		{
			name:       "catalog renders layout and book table",
			method:     http.MethodGet,
			path:       "/catalog",
			wantStatus: http.StatusOK,
			contains:   "<table>",
		},
		{
			name:       "HTMX search renders row fragments only",
			method:     http.MethodGet,
			path:       "/web/books?q=Go",
			hxRequest:  true,
			wantStatus: http.StatusOK,
			contains:   "Go Patterns",
			notContain: "<!doctype html>",
		},
		{
			name:       "delete without session is unauthorized",
			method:     http.MethodDelete,
			path:       "/web/books/1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "reader cannot delete a book",
			method:     http.MethodDelete,
			path:       "/web/books/1",
			cookie:     readerToken,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "static CSS served from embedded sub-filesystem",
			method:     http.MethodGet,
			path:       "/static/catalog.css",
			wantStatus: http.StatusOK,
			contains:   "--ink:#192824",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(test.method, server.URL+test.path, nil)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			if test.hxRequest {
				request.Header.Set("HX-Request", "true")
			}
			if test.cookie != "" {
				request.AddCookie(&http.Cookie{Name: "libratrack_session", Value: test.cookie})
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatalf("send request: %v", err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status=%d, want %d; body=%s", response.StatusCode, test.wantStatus, body)
			}
			if test.contains != "" && !strings.Contains(string(body), test.contains) {
				t.Errorf("response does not contain %q: %s", test.contains, body)
			}
			if test.notContain != "" && strings.Contains(string(body), test.notContain) {
				t.Errorf("response unexpectedly contains %q: %s", test.notContain, body)
			}
		})
	}
}

func newCatalogServer(t *testing.T) (*httptest.Server, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get test database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	statements := []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT, password_hash TEXT, role TEXT)`,
		`CREATE TABLE categories (id INTEGER PRIMARY KEY, name TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE authors (id INTEGER PRIMARY KEY, full_name TEXT, bio TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE books (id INTEGER PRIMARY KEY, title TEXT, isbn TEXT, category_id INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE book_authors (book_id INTEGER, author_id INTEGER, author_order INTEGER, PRIMARY KEY (book_id, author_id))`,
		`CREATE TABLE copies (id INTEGER PRIMARY KEY, book_id INTEGER, inventory_no TEXT, status TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE loans (id INTEGER PRIMARY KEY, user_id INTEGER, copy_id INTEGER, loaned_at DATETIME, due_at DATETIME, returned_at DATETIME)`,
		`INSERT INTO categories (id, name) VALUES (1, 'Test Category')`,
		`INSERT INTO authors (id, full_name) VALUES (1, 'Test Author')`,
		`INSERT INTO books (id, title, isbn, category_id) VALUES (1, 'Go Patterns', '9780000000001', 1)`,
		`INSERT INTO book_authors (book_id, author_id, author_order) VALUES (1, 1, 1)`,
		`INSERT INTO copies (id, book_id, inventory_no, status) VALUES (1, 1, 'COPY-1', 'available')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("initialize test database: %v", err)
		}
	}
	server := httptest.NewServer(app.NewRouterWithDB(db))
	return server, db
}
