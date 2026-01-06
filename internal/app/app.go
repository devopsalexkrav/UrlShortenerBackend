package app

import (
	"log/slog"

	"urlShortener/internal/http-server/handlers/redirect"
	"urlShortener/internal/http-server/handlers/url/delete"
	"urlShortener/internal/http-server/handlers/url/save"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Storage defines the interface for URL storage operations.
// This allows using different storage implementations (sqlite, postgres, etc.)
type Storage interface {
	SaveURL(urlToSave string, alias string) (int64, error)
	GetURL(alias string) (string, error)
	DeleteURL(alias string) error
}

// NewRouter creates and configures a chi router with all application routes.
// It accepts dependencies that can be swapped for testing.
func NewRouter(log *slog.Logger, storage Storage, user, password string) chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			user: password,
		}))

		r.Post("/", save.New(log, storage))
		r.Delete("/{alias}", delete.New(log, storage))
	})

	router.Get("/{alias}", redirect.New(log, storage))

	return router
}

