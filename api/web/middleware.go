package web

import (
	"context"
	"net/http"
	"time"

	"github.com/masudur-rahman/kazi-ancestry/infra/logr"
	"github.com/masudur-rahman/kazi-ancestry/models"

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
