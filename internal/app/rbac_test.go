package app_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"libratrack/internal/auth"
	"libratrack/internal/model"
)

func TestRBACIntegration(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-only-secret-with-enough-entropy")
	server, db := newSecurityTestServer(t)
	roles := map[string]string{"reader": "reader", "librarian": "librarian", "admin": "admin"}
	tokens := make(map[string]string, len(roles))
	for roleName, role := range roles {
		token, err := auth.IssueToken(model.User{ID: 1, Role: role})
		if err != nil {
			t.Fatalf("issue %s token: %v", roleName, err)
		}
		tokens[roleName] = token
	}

	t.Run("missing and expired tokens are unauthorized", func(t *testing.T) {
		cases := []struct {
			name  string
			token string
		}{
			{name: "missing token"},
			{name: "expired token", token: expiredToken(t)},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				response := requestJSON(t, http.MethodGet, server.URL+"/api/v1/me/loans", "", testCase.token)
				if response.StatusCode != http.StatusUnauthorized {
					t.Fatalf("status = %d, want 401", response.StatusCode)
				}
			})
		}
	})

	t.Run("reader forbidden and staff can create books", func(t *testing.T) {
		cases := []struct {
			role       string
			wantStatus int
			isbn       string
		}{
			{role: "reader", wantStatus: http.StatusForbidden, isbn: "9000000000001"},
			{role: "admin", wantStatus: http.StatusCreated, isbn: "9000000000002"},
			{role: "librarian", wantStatus: http.StatusCreated, isbn: "9000000000003"},
		}
		for _, testCase := range cases {
			t.Run(testCase.role, func(t *testing.T) {
				body := `{"title":"RBAC test book","isbn":"` + testCase.isbn + `","category_id":1,"author_ids":[1]}`
				response := requestJSON(t, http.MethodPost, server.URL+"/api/v1/books", body, tokens[testCase.role])
				if response.StatusCode != testCase.wantStatus {
					t.Fatalf("status = %d, want %d", response.StatusCode, testCase.wantStatus)
				}
			})
		}
	})

	t.Run("admin only can list users and hashes are omitted", func(t *testing.T) {
		if err := db.Exec(`INSERT INTO users (email, password_hash, role) VALUES ('hash-check@example.test', 'secret-hash', 'reader')`).Error; err != nil {
			t.Fatalf("insert user: %v", err)
		}
		readerResponse := requestJSON(t, http.MethodGet, server.URL+"/api/v1/users", "", tokens["reader"])
		if readerResponse.StatusCode != http.StatusForbidden {
			t.Fatalf("reader status = %d, want 403", readerResponse.StatusCode)
		}
		adminResponse := requestJSON(t, http.MethodGet, server.URL+"/api/v1/users", "", tokens["admin"])
		if adminResponse.StatusCode != http.StatusOK {
			t.Fatalf("admin status = %d, want 200", adminResponse.StatusCode)
		}
		var payload map[string]any
		if err := json.NewDecoder(adminResponse.Body).Decode(&payload); err != nil {
			t.Fatalf("decode user list: %v", err)
		}
		encoded, _ := json.Marshal(payload)
		if stringContains(string(encoded), "secret-hash") {
			t.Fatalf("user response exposed password hash: %s", encoded)
		}
	})

	t.Run("profile DTO cannot mass assign role or password hash", func(t *testing.T) {
		response := requestJSON(t, http.MethodPut, server.URL+"/api/v1/me", `{"email":"updated@example.test","role":"admin","password_hash":"attacker-controlled"}`, tokens["reader"])
		if response.StatusCode != http.StatusOK {
			t.Fatalf("profile update status = %d, want 200", response.StatusCode)
		}
		var user model.User
		if err := db.Where("id = ?", 1).First(&user).Error; err != nil {
			t.Fatalf("fetch updated profile: %v", err)
		}
		if user.Email != "updated@example.test" || user.Role != "reader" || user.PasswordHash != "secret-hash" {
			t.Fatalf("profile DTO changed protected fields: %+v", user)
		}
	})

	t.Run("only admin may change user roles", func(t *testing.T) {
		readerResponse := requestJSON(t, http.MethodPut, server.URL+"/api/v1/users/1/role", `{"role":"librarian"}`, tokens["reader"])
		if readerResponse.StatusCode != http.StatusForbidden {
			t.Fatalf("reader role update status = %d, want 403", readerResponse.StatusCode)
		}
		adminResponse := requestJSON(t, http.MethodPut, server.URL+"/api/v1/users/1/role", `{"role":"librarian"}`, tokens["admin"])
		if adminResponse.StatusCode != http.StatusNoContent {
			t.Fatalf("admin role update status = %d, want 204", adminResponse.StatusCode)
		}
	})

	t.Run("staff loan workflow and reader loan access", func(t *testing.T) {
		if err := db.Exec(`INSERT INTO copies (id, book_id, inventory_no, status) VALUES (1, 1, 'RBAC-COPY-1', 'available')`).Error; err != nil {
			t.Fatalf("insert copy: %v", err)
		}
		readerPost := requestJSON(t, http.MethodPost, server.URL+"/api/v1/loans", `{"copy_id":1}`, tokens["reader"])
		if readerPost.StatusCode != http.StatusForbidden {
			t.Fatalf("reader loan status = %d, want 403", readerPost.StatusCode)
		}
		loanResponse := requestJSON(t, http.MethodPost, server.URL+"/api/v1/loans", `{"copy_id":1}`, tokens["librarian"])
		if loanResponse.StatusCode != http.StatusCreated {
			t.Fatalf("staff loan status = %d, want 201", loanResponse.StatusCode)
		}
		var loanPayload struct {
			Loan struct {
				ID uint `json:"id"`
			} `json:"loan"`
		}
		if err := json.NewDecoder(loanResponse.Body).Decode(&loanPayload); err != nil {
			t.Fatalf("decode loan response: %v", err)
		}
		mine := requestJSON(t, http.MethodGet, server.URL+"/api/v1/me/loans", "", tokens["reader"])
		if mine.StatusCode != http.StatusOK {
			t.Fatalf("reader loans status = %d, want 200", mine.StatusCode)
		}
		returned := requestJSON(t, http.MethodPost, server.URL+"/api/v1/loans/"+itoa(loanPayload.Loan.ID)+"/return", "{}", tokens["admin"])
		if returned.StatusCode != http.StatusOK {
			t.Fatalf("return status = %d, want 200", returned.StatusCode)
		}
	})
}

func stringContains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}

func itoa(value uint) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[index:])
}
