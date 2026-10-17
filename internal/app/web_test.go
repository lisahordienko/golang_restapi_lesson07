package app_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"libratrack/internal/auth"
	"libratrack/internal/model"
)

func TestCatalogAndHTMXIntegration(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-only-secret-with-enough-entropy")
	server, db := newSecurityTestServer(t)
	if err := db.Exec(`INSERT INTO users (id, email, password_hash, role) VALUES (1, 'staff@example.test', 'test-hash', 'librarian')`).Error; err != nil {
		t.Fatalf("insert librarian test user: %v", err)
	}
	if err := db.Exec(`INSERT INTO books (id, title, isbn, category_id) VALUES (1, '<img src=x onerror=alert(1)> Go Patterns', '9780000000001', 1)`).Error; err != nil {
		t.Fatalf("insert test book: %v", err)
	}
	if err := db.Exec(`INSERT INTO book_authors (book_id, author_id, author_order) VALUES (1, 1, 1)`).Error; err != nil {
		t.Fatalf("associate test author: %v", err)
	}
	if err := db.Exec(`INSERT INTO copies (book_id, inventory_no, status) VALUES (1, 'COPY-1', 'available')`).Error; err != nil {
		t.Fatalf("insert test copy: %v", err)
	}

	t.Run("catalog renders full escaped HTML", func(t *testing.T) {
		token, err := auth.IssueToken(model.User{ID: 1, Role: "librarian"})
		if err != nil {
			t.Fatalf("issue librarian token: %v", err)
		}
		request, err := http.NewRequest(http.MethodGet, server.URL+"/catalog", nil)
		if err != nil {
			t.Fatalf("build catalog request: %v", err)
		}
		request.AddCookie(&http.Cookie{Name: "libratrack_session", Value: token})
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("send catalog request: %v", err)
		}
		defer response.Body.Close()
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read catalog response: %v", err)
		}
		body := string(bodyBytes)
		if response.StatusCode != http.StatusOK || !strings.Contains(body, "<!doctype html>") || !strings.Contains(body, "<table>") {
			t.Fatalf("catalog response status/body unexpected: %d %s", response.StatusCode, body)
		}
		if !strings.Contains(body, "&lt;img src=x onerror=alert(1)&gt;") || strings.Contains(body, "<img src=x onerror=alert(1)>") {
			t.Fatalf("untrusted title was not contextually escaped: %s", body)
		}
		if !strings.Contains(body, "1 / 1 available") {
			t.Fatalf("availability not rendered: %s", body)
		}
		if !strings.Contains(body, "staff@example.test") || !strings.Contains(body, "librarian") || !strings.Contains(body, "Sign out") {
			t.Fatalf("authenticated identity is missing from header: %s", body)
		}
		for _, attribute := range []string{
			`hx-get="/web/books"`,
			`hx-trigger="keyup changed delay:300ms"`,
			`hx-target="#book-list"`,
			`hx-delete="/web/books/1"`,
			`hx-target="closest tr"`,
			`hx-swap="outerHTML"`,
			`hx-confirm="Delete this book?"`,
		} {
			if !strings.Contains(body, attribute) {
				t.Errorf("catalog missing required attribute %s", attribute)
			}
		}
	})

	t.Run("search responds with only matching row fragments", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, server.URL+"/web/books?q=Go", nil)
		if err != nil {
			t.Fatalf("build HTMX search request: %v", err)
		}
		request.Header.Set("HX-Request", "true")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("send HTMX search request: %v", err)
		}
		defer response.Body.Close()
		fragment, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read HTMX fragment: %v", err)
		}
		body := string(fragment)
		if response.StatusCode != http.StatusOK || !strings.Contains(body, "<tr>") || !strings.Contains(body, "Go Patterns") || strings.Contains(body, "<!doctype html>") {
			t.Fatalf("unexpected fragment response: status=%d body=%s", response.StatusCode, body)
		}
		_, missing := webRequest(t, server.URL, http.MethodGet, "/web/books?q=missing")
		if !strings.Contains(missing, "No books match") {
			t.Fatalf("empty search message missing: %s", missing)
		}
	})

	t.Run("static assets served from embedded filesystem", func(t *testing.T) {
		response, body := webRequest(t, server.URL, http.MethodGet, "/static/catalog.css")
		if response.StatusCode != http.StatusOK || !strings.Contains(body, "--ink:#192824") {
			t.Fatalf("static CSS response unexpected: status=%d", response.StatusCode)
		}
		htmx, _ := webRequest(t, server.URL, http.MethodGet, "/static/htmx.min.js")
		if htmx.StatusCode != http.StatusOK {
			t.Fatalf("HTMX status=%d, want 200", htmx.StatusCode)
		}
	})
}

func TestBrowserLogoutClearsSessionCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-only-secret-with-enough-entropy")
	server, _ := newSecurityTestServer(t)
	request, err := http.NewRequest(http.MethodPost, server.URL+"/web/logout", nil)
	if err != nil {
		t.Fatalf("build logout request: %v", err)
	}
	request.Header.Set("Origin", "http://"+request.Host)
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send logout request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/catalog" {
		t.Fatalf("logout status/location=%d/%q, want 303 /catalog", response.StatusCode, response.Header.Get("Location"))
	}
	var sessionCookie *http.Cookie
	for _, cookie := range response.Cookies() {
		if cookie.Name == "libratrack_session" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil || sessionCookie.MaxAge >= 0 || sessionCookie.Value != "" {
		t.Fatalf("logout cookie=%+v, want cleared session cookie", sessionCookie)
	}
}

func TestCatalogDeleteRBACAndCSRF(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-only-secret-with-enough-entropy")
	cases := []struct {
		name       string
		role       string
		wantStatus int
		origin     string
	}{
		{name: "unauthenticated", wantStatus: http.StatusUnauthorized},
		{name: "reader forbidden", role: "reader", wantStatus: http.StatusForbidden},
		{name: "librarian cross-origin forbidden", role: "librarian", wantStatus: http.StatusForbidden, origin: "https://attacker.example"},
		{name: "librarian same-origin delete", role: "librarian", wantStatus: http.StatusOK, origin: "http://example.test"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server, db := newSecurityTestServer(t)
			if err := db.Exec(`INSERT INTO books (id, title, isbn, category_id) VALUES (1, 'Delete me', '9780000000002', 1)`).Error; err != nil {
				t.Fatalf("insert book: %v", err)
			}
			var token string
			if testCase.role != "" {
				var err error
				token, err = auth.IssueToken(model.User{ID: 1, Role: testCase.role})
				if err != nil {
					t.Fatalf("issue token: %v", err)
				}
			}
			request, err := http.NewRequest(http.MethodDelete, server.URL+"/web/books/1", nil)
			if err != nil {
				t.Fatalf("build delete request: %v", err)
			}
			request.Host = "example.test"
			if token != "" {
				request.AddCookie(&http.Cookie{Name: "libratrack_session", Value: token})
			}
			if testCase.origin != "" {
				request.Header.Set("Origin", testCase.origin)
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatalf("send delete: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != testCase.wantStatus {
				t.Fatalf("status=%d, want %d", response.StatusCode, testCase.wantStatus)
			}
			if testCase.wantStatus == http.StatusOK {
				body, err := io.ReadAll(response.Body)
				if err != nil || len(body) != 0 {
					t.Fatalf("successful delete body=%q err=%v, want empty body", body, err)
				}
				var count int64
				if err := db.Model(&model.Book{}).Where("id = ?", 1).Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("book count=%d err=%v after delete", count, err)
				}
			}
		})
	}
}

func TestBrowserLoginSetsHttpOnlyCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-only-secret-with-enough-entropy")
	server, _ := newSecurityTestServer(t)
	registered := requestJSON(t, http.MethodPost, server.URL+"/api/v1/auth/register", `{"email":"web@example.test","password":"strong-password"}`, "")
	if registered.StatusCode != http.StatusCreated {
		t.Fatalf("registration status=%d, want 201", registered.StatusCode)
	}

	values := url.Values{"email": {"web@example.test"}, "password": {"strong-password"}}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/web/login", strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatalf("build browser login request: %v", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://"+request.Host)
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send browser login: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/catalog" {
		t.Fatalf("login status/location=%d/%q, want 303 /catalog", response.StatusCode, response.Header.Get("Location"))
	}
	cookies := response.Cookies()
	if len(cookies) != 1 || cookies[0].Name != "libratrack_session" || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].MaxAge != int((24*time.Hour).Seconds()) {
		t.Fatalf("unexpected browser session cookie: %+v", cookies)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read browser login response: %v", err)
	}
	if strings.Contains(string(body), cookies[0].Value) {
		t.Fatal("browser login response exposed JWT outside the HttpOnly cookie")
	}
}

func webRequest(t *testing.T, baseURL, method, path string) (*http.Response, string) {
	t.Helper()
	request, err := http.NewRequest(method, baseURL+path, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return response, string(body)
}
