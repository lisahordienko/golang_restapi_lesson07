// Package app wires the whole HTTP API together.
package app

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"homework/internal/handler"
	"homework/internal/middleware"
	"homework/internal/model"
	"homework/internal/repository"
)

// NewRouter creates a fresh, isolated API handler for the selected variant.
func NewRouter() http.Handler {
	variant := loadVariant()
	store := repository.New(variant)
	resource := handler.New(
		store,
		variant.Resource,
		variant.RequiredFields,
		variant.OptionalInt,
		variant.OptionalText,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.Handle("/api/v1/"+variant.Resource, resource)
	mux.Handle("/api/v1/"+variant.Resource+"/", resource)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotFound, "not_found", "Resource not found")
	})

	return middleware.RequestLogger(slog.Default())(
		middleware.CORS(middleware.Recoverer(mux)),
	)
}

func loadVariant() model.Variant {
	_, file, _, _ := runtime.Caller(0)
	variantPath := filepath.Join(filepath.Dir(file), "..", "..", "VARIANT")
	data, err := os.ReadFile(variantPath)
	if err != nil {
		panic(err)
	}
	name := strings.ToLower(strings.TrimSpace(string(data)))
	variants := map[string]model.Variant{
		"books":   {Name: "books", Resource: "books", RequiredFields: []string{"title", "isbn", "author", "category"}, OptionalInt: "published_year", OptionalText: "description"},
		"movies":  {Name: "movies", Resource: "movies", RequiredFields: []string{"title", "director", "country", "genre"}, OptionalInt: "release_year", OptionalText: "synopsis"},
		"tasks":   {Name: "tasks", Resource: "tasks", RequiredFields: []string{"title", "assignee", "project", "status"}, OptionalInt: "priority", OptionalText: "notes"},
		"recipes": {Name: "recipes", Resource: "recipes", RequiredFields: []string{"name", "author", "difficulty", "cuisine"}, OptionalInt: "cook_minutes", OptionalText: "instructions"},
		"devices": {Name: "devices", Resource: "devices", RequiredFields: []string{"name", "serial", "manufacturer", "type"}, OptionalInt: "warranty_months", OptionalText: "comment"},
		"courses": {Name: "courses", Resource: "courses", RequiredFields: []string{"title", "teacher", "level", "subject"}, OptionalInt: "hours", OptionalText: "program"},
		"albums":  {Name: "albums", Resource: "albums", RequiredFields: []string{"title", "artist", "label", "genre"}, OptionalInt: "release_year", OptionalText: "notes"},
		"pets":    {Name: "pets", Resource: "pets", RequiredFields: []string{"name", "owner", "breed", "species"}, OptionalInt: "age", OptionalText: "notes"},
	}
	if variant, ok := variants[name]; ok {
		return variant
	}
	panic("unknown VARIANT: " + name)
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
