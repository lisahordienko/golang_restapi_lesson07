package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerWritesStructuredAccessLog(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/books?secret=top-secret", strings.NewReader("request-body"))
	req.Header.Set("X-Request-ID", "request-123")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusCreated)
	}
	if got := res.Header().Get("X-Request-ID"); got != "request-123" {
		t.Fatalf("X-Request-ID = %q, want %q", got, "request-123")
	}

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("log is not valid JSON: %v\n%s", err, output.String())
	}
	if entry["msg"] != "HTTP request" {
		t.Fatalf("message = %v, want HTTP request", entry["msg"])
	}
	if entry["method"] != http.MethodPost {
		t.Fatalf("method = %v", entry["method"])
	}
	if entry["path"] != "/api/v1/books" {
		t.Fatalf("path = %v", entry["path"])
	}
	if entry["status"] != float64(http.StatusCreated) {
		t.Fatalf("status = %v", entry["status"])
	}
	if entry["request_id"] != "request-123" {
		t.Fatalf("request_id = %v", entry["request_id"])
	}
	if _, ok := entry["duration"]; !ok {
		t.Fatal("duration is missing")
	}
	if strings.Contains(output.String(), "secret") || strings.Contains(output.String(), "request-body") {
		t.Fatalf("sensitive request data leaked into log: %s", output.String())
	}
}
