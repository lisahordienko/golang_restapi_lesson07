package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// RequestLogger records a safe, structured access log for every HTTP request.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = newRequestID()
			}
			r = r.WithContext(withRequestID(r.Context(), requestID))
			w.Header().Set("X-Request-ID", requestID)

			startedAt := time.Now()
			response := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(response, r)

			logger.Info(
				"HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", response.statusCode,
				"duration", time.Since(startedAt).Round(time.Millisecond),
				"remote_address", r.RemoteAddr,
				"protocol", r.Proto,
				"request_id", requestID,
			)
		})
	}
}

type requestIDKey struct{}

func withRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func newRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return time.Now().UTC().Format("20060102T150405.000000000Z")
	}
	return hex.EncodeToString(bytes)
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusRecorder) WriteHeader(statusCode int) {
	if w.statusCode == http.StatusOK {
		w.statusCode = statusCode
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusRecorder) Write(p []byte) (int, error) {
	if w.statusCode == http.StatusOK {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(p)
}
