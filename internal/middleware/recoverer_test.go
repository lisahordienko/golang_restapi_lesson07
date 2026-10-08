package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecovererConvertsPanicToInternalServerError(t *testing.T) {
	handler := Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("unexpected failure")
	}))

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusInternalServerError)
	}
	if got := res.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", got)
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	errorBody, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error body = %#v, want object", body["error"])
	}
	if got := errorBody["code"]; got != "internal_server_error" {
		t.Fatalf("error code = %v, want internal_server_error", got)
	}
}

func TestRecovererDoesNotCorruptResponseWrittenBeforePanic(t *testing.T) {
	handler := Recoverer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial response"))
		panic("unexpected failure after writing")
	}))

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Body.String(); got != "partial response" {
		t.Fatalf("body = %q, want only the original response", got)
	}
}
