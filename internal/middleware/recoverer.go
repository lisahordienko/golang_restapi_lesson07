package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// Recoverer converts unexpected panics into a safe JSON error response.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := &recoveryResponseWriter{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				if !response.wrote {
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"error": map[string]any{
							"code":    "internal_server_error",
							"message": "Internal server error",
						},
					})
				}
				log.Printf("panic recovered: %v\n%s", recovered, debug.Stack())
			}
		}()
		next.ServeHTTP(response, r)
	})
}

type recoveryResponseWriter struct {
	http.ResponseWriter
	wrote bool
}

func (w *recoveryResponseWriter) WriteHeader(statusCode int) {
	w.wrote = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *recoveryResponseWriter) Write(p []byte) (int, error) {
	w.wrote = true
	return w.ResponseWriter.Write(p)
}
