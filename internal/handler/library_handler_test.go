package handler

import (
	"net/http/httptest"
	"testing"
)

func TestWriteErrorUsesJSONEnvelope(t *testing.T) {
	response := httptest.NewRecorder()
	WriteError(response, 404, "not_found", "Book not found")
	if response.Code != 404 || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("unexpected error response: status=%d content-type=%q", response.Code, response.Header().Get("Content-Type"))
	}
}
