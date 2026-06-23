package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the HTTP router: the SSR shell, whitelisted static assets,
// and the JSON mutation API. The tree is delivered only via page injection — no
// GET endpoint exposes the full dataset.
//
// Phase 3 wraps the mutation/suggestion routes (and gates the page + assets)
// with the OAuth session middleware.
func NewRouter(webDir string) chi.Router {
	r := chi.NewRouter()
	r.Use(RequestLogger)
	r.Use(middleware.Recoverer)

	pg := newPage(webDir)

	r.Get("/healthz", HandleHealthz)
	r.Get("/", pg.handleIndex)
	r.Get("/app.js", pg.serveAsset("app.js"))
	r.Get("/style.css", pg.serveAsset("style.css"))

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/people", HandleCreatePerson)
		r.Put("/people/{id}", HandleUpdatePerson)
		r.Delete("/people/{id}", HandleDeletePerson)

		r.Post("/suggestions", HandleSubmitSuggestion)
		r.Get("/suggestions", HandleListSuggestions)
		r.Post("/suggestions/{id}/approve", HandleApproveSuggestion)
		r.Post("/suggestions/{id}/reject", HandleRejectSuggestion)
	})

	return r
}

// HandleHealthz is a liveness probe.
func HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
