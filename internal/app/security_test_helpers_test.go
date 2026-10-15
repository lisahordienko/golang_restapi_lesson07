package app_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"libratrack/internal/app"
)

func newSecurityTestServer(t *testing.T) (*httptest.Server, *gorm.DB) {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite test database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	statements := []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL DEFAULT '', role TEXT NOT NULL DEFAULT 'reader', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE categories (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE authors (id INTEGER PRIMARY KEY AUTOINCREMENT, full_name TEXT NOT NULL, bio TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE books (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, isbn TEXT NOT NULL UNIQUE, category_id INTEGER NOT NULL, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE book_authors (book_id INTEGER NOT NULL, author_id INTEGER NOT NULL, author_order INTEGER NOT NULL DEFAULT 1, PRIMARY KEY (book_id, author_id))`,
		`CREATE TABLE copies (id INTEGER PRIMARY KEY AUTOINCREMENT, book_id INTEGER NOT NULL, inventory_no TEXT NOT NULL UNIQUE, status TEXT NOT NULL DEFAULT 'available', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE loans (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, copy_id INTEGER NOT NULL, loaned_at DATETIME NOT NULL, due_at DATETIME NOT NULL, returned_at DATETIME)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create test schema: %v", err)
		}
	}
	if err := db.Exec(`INSERT INTO categories (id, name) VALUES (1, 'Test Category')`).Error; err != nil {
		t.Fatalf("seed category: %v", err)
	}
	if err := db.Exec(`INSERT INTO authors (id, full_name) VALUES (1, 'Test Author')`).Error; err != nil {
		t.Fatalf("seed author: %v", err)
	}
	server := httptest.NewServer(app.NewRouterWithDB(db))
	t.Cleanup(func() {
		server.Close()
		_ = sqlDB.Close()
	})
	return server, db
}

func requestJSON(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	t.Cleanup(func() { response.Body.Close() })
	return response
}
