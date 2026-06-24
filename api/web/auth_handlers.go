package web

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/masudur-rahman/kazi-ancestry/configs"
	"github.com/masudur-rahman/kazi-ancestry/infra/metrics"
	"github.com/masudur-rahman/kazi-ancestry/modules/auth"

	"golang.org/x/oauth2"
)

const stateCookie = "kazi_oauth_state"

// secureCookies reports whether cookies should be marked Secure (https redirect).
func secureCookies() bool {
	return strings.HasPrefix(configs.KaziConfig.Auth.RedirectURL, "https://")
}

func setSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: auth.SessionCookie, Value: value, Path: "/",
		MaxAge: maxAge, HttpOnly: true, Secure: secureCookies(), SameSite: http.SameSiteLaxMode,
	})
}

// HandleLogin starts the Google OAuth flow.
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if !configs.KaziConfig.Auth.Enabled() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	state := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name: stateCookie, Value: state, Path: "/",
		MaxAge: 600, HttpOnly: true, Secure: secureCookies(), SameSite: http.SameSiteLaxMode,
	})
	url := auth.GoogleConfig().AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusFound)
}

// HandleCallback completes OAuth: verifies state, resolves the allowlist role,
// and sets the session cookie. Non-allowlisted accounts are denied.
func HandleCallback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(stateCookie)
	if err != nil || c.Value == "" || c.Value != r.URL.Query().Get("state") {
		WriteError(w, http.StatusBadRequest, "bad_state", "invalid oauth state")
		return
	}
	// clear the state cookie
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Value: "", Path: "/", MaxAge: -1})

	user, err := auth.FetchUser(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		metrics.LoginResult("error")
		WriteError(w, http.StatusBadGateway, "oauth_error", err.Error())
		return
	}
	if !user.VerifiedEmail {
		metrics.LoginResult("denied")
		WriteError(w, http.StatusForbidden, "unverified", "email is not verified")
		return
	}

	// Any authenticated account gets a session; RoleFor decides admin /
	// contributor / viewer. Non-allowlisted users are read-only viewers, not
	// denied — guests can view the tree anyway.
	role := configs.KaziConfig.Auth.RoleFor(user.Email)
	sess := auth.NewSession(user.Email, user.Name, role)
	setSessionCookie(w, auth.Sign(sess, configs.KaziConfig.Auth.SessionSecret), auth.MaxAgeSeconds())
	metrics.LoginResult("success")
	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleLogout clears the session cookie.
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	setSessionCookie(w, "", -1)
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
