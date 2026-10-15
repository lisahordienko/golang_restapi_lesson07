package app_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"libratrack/internal/auth"
	"libratrack/internal/model"
)

func TestAuthIntegration(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-only-secret-with-enough-entropy")
	server, db := newSecurityTestServer(t)

	t.Run("register and duplicate email", func(t *testing.T) {
		cases := []struct {
			name       string
			body       string
			wantStatus int
		}{
			{name: "first registration", body: `{"email":"reader@example.test","password":"strong-password"}`, wantStatus: http.StatusCreated},
			{name: "duplicate email", body: `{"email":"reader@example.test","password":"strong-password"}`, wantStatus: http.StatusUnprocessableEntity},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				response := requestJSON(t, http.MethodPost, server.URL+"/api/v1/auth/register", testCase.body, "")
				if response.StatusCode != testCase.wantStatus {
					t.Fatalf("status = %d, want %d", response.StatusCode, testCase.wantStatus)
				}
				if testCase.name == "first registration" {
					var stored model.User
					if err := db.Where("email = ?", "reader@example.test").First(&stored).Error; err != nil {
						t.Fatalf("fetch registered user: %v", err)
					}
					if stored.Role != "reader" || bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("strong-password")) != nil {
						t.Fatal("registration must store a bcrypt hash and assign the reader role")
					}
					body, _ := io.ReadAll(response.Body)
					var payload map[string]any
					if err := json.Unmarshal(body, &payload); err != nil {
						t.Fatalf("decode registration response: %v", err)
					}
					encoded, _ := json.Marshal(payload)
					if string(encoded) == "" || strings.Contains(string(encoded), "password_hash") {
						t.Fatalf("registration response exposed password hash: %s", encoded)
					}
				}
			})
		}
	})

	t.Run("valid login returns signed jwt", func(t *testing.T) {
		response := requestJSON(t, http.MethodPost, server.URL+"/api/v1/auth/login", `{"email":"reader@example.test","password":"strong-password"}`, "")
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", response.StatusCode)
		}
		var payload struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil || payload.Token == "" {
			t.Fatalf("login response has no token: err=%v", err)
		}
		claims, err := auth.ParseToken(payload.Token)
		if err != nil {
			t.Fatalf("parse issued token: %v", err)
		}
		if claims.UserID == 0 || claims.Role != "reader" || claims.ExpiresAt == nil {
			t.Fatalf("unexpected claims: %+v", claims)
		}
		if claims.IssuedAt == nil || claims.ExpiresAt.Sub(claims.IssuedAt.Time) != 24*time.Hour {
			t.Fatalf("token lifetime = %v, want 24h", claims.ExpiresAt.Sub(claims.IssuedAt.Time))
		}
	})

	t.Run("unknown email and wrong password are identical", func(t *testing.T) {
		cases := []struct {
			name string
			body string
		}{
			{name: "wrong password", body: `{"email":"reader@example.test","password":"incorrect-password"}`},
			{name: "unknown email", body: `{"email":"nobody@example.test","password":"incorrect-password"}`},
		}
		var message string
		for index, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				response := requestJSON(t, http.MethodPost, server.URL+"/api/v1/auth/login", testCase.body, "")
				if response.StatusCode != http.StatusUnauthorized {
					t.Fatalf("status = %d, want 401", response.StatusCode)
				}
				var payload struct {
					Error struct {
						Message string `json:"message"`
					} `json:"error"`
				}
				if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if payload.Error.Message != "invalid credentials" {
					t.Fatalf("message = %q, want generic invalid credentials", payload.Error.Message)
				}
				if index == 0 {
					message = payload.Error.Message
				} else if payload.Error.Message != message {
					t.Fatalf("unknown email message differs: %q vs %q", payload.Error.Message, message)
				}
			})
		}
	})

	t.Run("oversized registration body is rejected", func(t *testing.T) {
		body := `{"email":"large@example.test","password":"` + strings.Repeat("a", (1<<20)+1) + `"}`
		response := requestJSON(t, http.MethodPost, server.URL+"/api/v1/auth/register", body, "")
		if response.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("oversized request status = %d, want 422", response.StatusCode)
		}
	})
}

func expiredToken(t *testing.T) string {
	t.Helper()
	claims := auth.Claims{UserID: 1, Role: "reader", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Second))}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-only-secret-with-enough-entropy"))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	return token
}
