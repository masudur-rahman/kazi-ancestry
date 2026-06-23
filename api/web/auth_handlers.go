package web

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/masudur-rahman/kazi-ancestry/configs"
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
		WriteError(w, http.StatusBadGateway, "oauth_error", err.Error())
		return
	}
	if !user.VerifiedEmail {
		WriteError(w, http.StatusForbidden, "unverified", "email is not verified")
		return
	}

	role := configs.KaziConfig.Auth.RoleFor(user.Email)
	if role == "" {
		// authenticated but not on the allowlist
		setSessionCookie(w, "", -1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8"><body style="font-family:serif;background:#fbf6ea;color:#5c4a2c;text-align:center;padding:4rem">` +
			`<h2>প্রবেশাধিকার নেই</h2><p>আপনার অ্যাকাউন্ট অনুমোদিত তালিকায় নেই।</p><p><a href="/">ফিরে যান</a></p></body>`))
		return
	}

	sess := auth.NewSession(user.Email, user.Name, role)
	setSessionCookie(w, auth.Sign(sess, configs.KaziConfig.Auth.SessionSecret), auth.MaxAgeSeconds())
	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleLogout clears the session cookie.
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	setSessionCookie(w, "", -1)
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
