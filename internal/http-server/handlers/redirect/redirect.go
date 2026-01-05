package redirect

import (
	"errors"
	"log/slog"
	"net/http"
	resp "urlShortener/internal/lib/api/response"
	"urlShortener/internal/lib/logger/sl"
	"urlShortener/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type URLGetter interface {
	GetURL(alias string) (string, error)
}

func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.redirect.New"

		log = log.With(slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())))

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Error("alias is empty")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid request"))
			return
		}

		resURL, err := urlGetter.GetURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", "alias", alias)
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.Error("not found"))
			return
		}

		if err != nil {
			log.Error("failed to get url", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		log.Info("got url", slog.String("url", resURL))

		http.Redirect(w, r, resURL, http.StatusFound)
	}
}
