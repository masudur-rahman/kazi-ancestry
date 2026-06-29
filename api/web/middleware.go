package web

import (
	"context"
	"net/http"
	"time"

	"github.com/masudur-rahman/kazi-ancestry/configs"
	"github.com/masudur-rahman/kazi-ancestry/infra/logr"
	"github.com/masudur-rahman/kazi-ancestry/models"
	"github.com/masudur-rahman/kazi-ancestry/modules/auth"

	"github.com/go-chi/chi/v5/middleware"
)

type ctxKey string

const userCtxKey ctxKey = "kazi.user"

// userFromContext returns the authenticated user, or nil for anonymous requests.
// Phase 3 (OAuth/session middleware) populates this; until then it is always nil.
func userFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(userCtxKey).(*models.User)
	return u
}

// withUser stores the authenticated user on the request context.
func withUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

// SessionMiddleware resolves the request's user from the signed session cookie
// and stores it on the context; an absent or invalid cookie means anonymous.
// In dev (no OAuth configured) the cookie is minted by the mock login at
// /auth/login (HandleDevLogin), signed with the same SessionSecret, so login and
// logout behave exactly as in production.
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user *models.User
		cfg := configs.KaziConfig.Auth
		if c, err := r.Cookie(auth.SessionCookie); err == nil {
			if s, err := auth.Verify(c.Value, cfg.SessionSecret); err == nil {
				user = &models.User{ID: s.Email, Email: s.Email, Name: s.Name, Role: s.Role}
			}
		}
		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
	})
}

// RequireAuth rejects anonymous requests.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userFromContext(r.Context()) == nil {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "login required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireContributor rejects requests that may not contribute: anonymous → 401,
// authenticated users who may not suggest → 403. Whether non-allowlisted viewers
// may suggest is governed by Auth.OpenSuggestions (CanSuggest).
func RequireContributor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		if u == nil {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "login required")
			return
		}
		if !configs.KaziConfig.Auth.CanSuggest(u.Role) {
			WriteError(w, http.StatusForbidden, "forbidden", "not allowed to contribute")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin rejects non-admin requests.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		if u == nil || u.Role != "admin" {
			WriteError(w, http.StatusForbidden, "forbidden", "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequestLogger logs the method, path, and status of each request.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		logr.DefaultLogger.Infow("request",
			"method", r.Method, "path", r.URL.Path,
			"status", ww.Status(), "took", time.Since(start).String())
	})
}
