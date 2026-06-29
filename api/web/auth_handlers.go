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

// HandleLogin starts the Google OAuth flow. When OAuth is not configured (dev),
// it serves the mock login form instead so local logout/role testing works.
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if !configs.KaziConfig.Auth.Enabled() {
		serveDevLogin(w)
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

// HandleDevLogin mints a session from the mock login form. It is a dev-only
// convenience: when real OAuth is configured it refuses and redirects home, so
// it is inert in production. The role is taken straight from the form so any
// role can be exercised locally.
func HandleDevLogin(w http.ResponseWriter, r *http.Request) {
	if configs.KaziConfig.Auth.Enabled() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10) // bound the form body (dev login)
	_ = r.ParseForm()
	email := strings.TrimSpace(r.FormValue("email"))
	if email == "" {
		email = "dev@local"
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = "Dev"
	}
	// Viewer is the logged-out default, so the mock form only offers admin /
	// contributor; anything else falls back to admin.
	role := r.FormValue("role")
	if role != "contributor" {
		role = "admin"
	}
	sess := auth.NewSession(email, name, role)
	setSessionCookie(w, auth.Sign(sess, configs.KaziConfig.Auth.SessionSecret), auth.MaxAgeSeconds())
	metrics.LoginResult("success")
	http.Redirect(w, r, "/", http.StatusFound)
}

// devLoginPage is the self-contained mock login form (dev only). It posts
// form-encoded fields (name, email, role) to /auth/dev-login. No user input is
// interpolated, so a static string is safe.
const devLoginPage = `<!doctype html>
<html lang="bn"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>ডেভ লগ ইন · Kazi Ancestry</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Spectral:ital,wght@0,400;0,500;0,600;0,700;1,400;1,500&family=Noto+Serif+Bengali:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
  body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;
    background:#f4ecd6;color:#3b2f21;font-family:'Spectral','Noto Serif Bengali',Georgia,serif}
  form{background:#fbf6ea;border:1px solid #d4c096;border-radius:14px;padding:28px 26px;
    width:320px;box-shadow:0 16px 40px rgba(70,48,18,.18)}
  h1{margin:0 0 4px;font-size:20px;color:#9c4326}
  p{margin:0 0 18px;font-size:12.5px;color:#8a7146}
  label{display:block;font-size:12.5px;margin:12px 0 5px;color:#5c4a2c}
  input,select{width:100%;box-sizing:border-box;padding:9px 11px;font-size:14px;
    border:1px solid #cdb988;border-radius:8px;background:#fdf9ee;color:#3b2f21;font-family:inherit}
  button{margin-top:20px;width:100%;padding:11px;font-size:14.5px;font-weight:600;
    border:1px solid #9c4326;border-radius:8px;background:#9c4326;color:#fbf5e7;cursor:pointer}
  button:hover{background:#85371f}
  .note{margin-top:14px;font-size:11px;color:#a8854a;text-align:center}
</style></head>
<body>
<form method="post" action="/auth/dev-login">
  <h1>ডেভ লগ ইন</h1>
  <p>স্থানীয় পরীক্ষার জন্য মক লগ ইন (OAuth নেই)।</p>
  <label for="name">নাম</label>
  <input id="name" name="name" value="Dev" autocomplete="off">
  <label for="email">ইমেইল</label>
  <input id="email" name="email" type="email" value="dev@local" autocomplete="off">
  <label for="role">ভূমিকা</label>
  <select id="role" name="role">
    <option value="admin">কর্তৃপক্ষ (admin)</option>
    <option value="contributor">সদস্য (contributor)</option>
  </select>
  <button type="submit">লগ ইন</button>
  <div class="note">শুধু ডেভেলপমেন্টের জন্য</div>
</form>
</body></html>`

func serveDevLogin(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(devLoginPage))
}
