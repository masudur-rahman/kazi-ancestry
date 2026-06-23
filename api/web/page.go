package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/masudur-rahman/kazi-ancestry/models"
	"github.com/masudur-rahman/kazi-ancestry/services/all"
)

// bootstrapMarker in index.html is replaced with the injected initial state.
const bootstrapMarker = "<!--KAZI_BOOTSTRAP-->"

// bootstrap is the per-request initial state embedded in the page. The tree is
// injected here (only for authorized requests) rather than served from a public
// JSON endpoint, so there is no scrapable data API.
type bootstrap struct {
	People      []models.Person     `json:"people"`
	User        *models.User        `json:"user"`
	Suggestions []models.Suggestion `json:"suggestions"`
}

// page serves index.html with server-rendered initial state and the static
// assets the app needs. It never serves the data files (family*.json).
type page struct {
	dir      string
	mu       sync.RWMutex
	template []byte
}

func newPage(dir string) *page { return &page{dir: dir} }

func (p *page) indexTemplate() ([]byte, error) {
	p.mu.RLock()
	if p.template != nil {
		t := p.template
		p.mu.RUnlock()
		return t, nil
	}
	p.mu.RUnlock()

	b, err := os.ReadFile(filepath.Join(p.dir, "index.html"))
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	p.template = b
	p.mu.Unlock()
	return b, nil
}

// handleIndex renders the SPA shell with the injected bootstrap state.
func (p *page) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := p.indexTemplate()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "template_error", err.Error())
		return
	}

	user := userFromContext(r.Context())

	// Anonymous requests get the login wall: the shell renders, but no tree data
	// is embedded — there is nothing to scrape until the user authenticates.
	var (
		people      []models.Person
		suggestions []models.Suggestion
	)
	if user != nil {
		var err error
		if people, err = all.GetServices().Person.List(); err != nil {
			WriteError(w, http.StatusInternalServerError, "load_error", err.Error())
			return
		}
		// The review inbox is admin-only.
		if user.Role == "admin" {
			suggestions, _ = all.GetServices().Suggestion.List()
		}
	}

	data := bootstrap{People: people, User: user, Suggestions: suggestions}
	blob, _ := json.Marshal(data)
	script := []byte(`<script id="kazi-bootstrap" type="application/json">` + string(blob) + `</script>`)

	html := bytes.Replace(tmpl, []byte(bootstrapMarker), script, 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(html)
}

// serveAsset serves a single whitelisted static file from the web dir.
func (p *page) serveAsset(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(p.dir, name))
	}
}
