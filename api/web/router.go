package web

import (
	"net/http"

	"github.com/masudur-rahman/kazi-ancestry/infra/metrics"

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
	r.Use(metrics.Middleware)
	r.Use(middleware.Recoverer)
	r.Use(SessionMiddleware)

	pg := newPage(webDir)

	r.Get("/healthz", HandleHealthz)
	r.Get("/", pg.handleIndex)
	r.Get("/app.js", pg.serveAsset("app.js"))
	r.Get("/style.css", pg.serveAsset("style.css"))

	r.Get("/auth/login", HandleLogin)
	r.Get("/auth/callback", HandleCallback)
	r.Post("/auth/logout", HandleLogout)
	r.Post("/auth/dev-login", HandleDevLogin) // dev-only mock login; inert when OAuth is configured

	r.Route("/api/v1", func(r chi.Router) {
		// Only allowlisted contributors (and admins) may suggest and view their
		// own submissions; logged-in viewers are read-only.
		r.With(RequireContributor).Post("/suggestions", HandleSubmitSuggestion)
		r.With(RequireContributor).Get("/suggestions/mine", HandleMySuggestions)

		// Direct tree edits and the review inbox are admin-only.
		r.Group(func(r chi.Router) {
			r.Use(RequireAdmin)
			r.Post("/people", HandleCreatePerson)
			r.Post("/people/reorder", HandleReorderPerson)
			r.Put("/people/{id}", HandleUpdatePerson)
			r.Delete("/people/{id}", HandleDeletePerson)

			r.Get("/suggestions", HandleListSuggestions)
			r.Post("/suggestions/{id}/approve", HandleApproveSuggestion)
			r.Post("/suggestions/{id}/reject", HandleRejectSuggestion)
		})
	})

	return r
}

// HandleHealthz is a liveness probe.
func HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
