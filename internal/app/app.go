// Package app wires the whole HTTP API together.
package app

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"libratrack"

	"libratrack/internal/auth"
	"libratrack/internal/handler"
	webhandler "libratrack/internal/handler/web"
	"libratrack/internal/middleware"
	"libratrack/internal/model"
	"libratrack/internal/repository"
	"libratrack/internal/service"
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

// NewRouterWithDB builds the production Gin API with auth and RBAC route groups.
func NewRouterWithDB(db *gorm.DB) http.Handler {
	gin.SetMode(gin.ReleaseMode)

	library := service.NewLibrary(
		repository.NewGormBookRepository(db),
		repository.NewGormAuthorRepository(db),
		repository.NewGormCategoryRepository(db),
		repository.NewGormCopyRepository(db),
	)
	authService := auth.NewService(repository.NewGormUserRepository(db))
	loanService := service.NewLoanService(repository.NewGormLoanRepository(db))

	libraryHandler := handler.NewLibraryHandler(library)
	authHandler := handler.NewAuthHandler(authService)
	loanHandler := handler.NewLoanHandler(loanService)
	webHandler, err := webhandler.New(library, authService)
	if err != nil {
		panic(err)
	}

	router := gin.New()
	router.Use(middleware.BodyLimit(1 << 20))
	router.POST("/web/login", middleware.RequireSameOrigin(), authHandler.BrowserLogin)
	if err := webHandler.Register(router); err != nil {
		panic(err)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/swagger/index.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<!doctype html><title>LibraTrack API</title><h1>LibraTrack API</h1>"))
	})

	api := router.Group("/api/v1")

	public := api.Group("")
	{
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/login", authHandler.Login)
		public.GET("/books", libraryHandler.ListBooks)
		public.GET("/books/:id", libraryHandler.GetBook)
		public.GET("/books/:id/copies", libraryHandler.ListCopies)
		public.GET("/books/:id/availability", libraryHandler.Availability)
		public.GET("/categories", libraryHandler.ListCategories)
		public.GET("/categories/:id", libraryHandler.GetCategory)
		public.GET("/authors", libraryHandler.ListAuthors)
		public.GET("/authors/:id", libraryHandler.GetAuthor)
	}

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", authHandler.Me)
		protected.PUT("/me", authHandler.UpdateMe)
		protected.GET("/me/loans", loanHandler.Mine)
	}

	staff := api.Group("")
	staff.Use(middleware.AuthMiddleware(), middleware.RequireRole("librarian", "admin"))
	{
		staff.POST("/books", libraryHandler.CreateBook)
		staff.PUT("/books/:id", libraryHandler.UpdateBook)
		staff.DELETE("/books/:id", libraryHandler.DeleteBook)
		staff.POST("/books/:id/copies", libraryHandler.CreateCopy)
		staff.POST("/categories", libraryHandler.CreateCategory)
		staff.POST("/authors", libraryHandler.CreateAuthor)
		staff.POST("/loans", loanHandler.Create)
		staff.POST("/loans/:id/return", loanHandler.Return)
	}

	admin := api.Group("")
	admin.Use(middleware.AuthMiddleware(), middleware.RequireRole("admin"))
	{
		admin.GET("/users", authHandler.ListUsers)
		admin.PUT("/users/:id/role", authHandler.UpdateUserRole)
	}

	return middleware.RequestLogger(slog.Default())(
		middleware.CORS(middleware.Recoverer(router)),
	)
}

func loadVariant() model.Variant {
	data, err := libratrack.Files.ReadFile("VARIANT")
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
