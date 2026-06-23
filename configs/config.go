package configs

import (
	"os"
	"strings"

	pglib "github.com/masudur-rahman/styx/sql/postgres/lib"
)

// KaziConfig is the process-wide configuration, populated by Load.
//
// NOTE: configuration is env-driven for now; the viper/yaml config-file loader
// used by expense-tracker-bot is part of the deferred peripheral conversion.
var KaziConfig Configuration

type Configuration struct {
	Database DatabaseConfig `json:"database" yaml:"database"`
	Server   ServerConfig   `json:"server" yaml:"server"`
	Auth     AuthConfig     `json:"auth" yaml:"auth"`
	SeedPath string         `json:"seedPath" yaml:"seedPath"`
}

// AuthConfig holds Google OAuth + allowlist settings. An empty GoogleClientID
// disables auth (dev mode): every request is treated as an admin so the app is
// usable without OAuth credentials.
type AuthConfig struct {
	GoogleClientID     string   `json:"googleClientId" yaml:"googleClientId"`
	GoogleClientSecret string   `json:"googleClientSecret" yaml:"googleClientSecret"`
	RedirectURL        string   `json:"redirectUrl" yaml:"redirectUrl"`
	SessionSecret      string   `json:"sessionSecret" yaml:"sessionSecret"`
	Allowlist          []string `json:"allowlist" yaml:"allowlist"`   // emails allowed as contributors
	Admins             []string `json:"admins" yaml:"admins"`         // emails granted admin
}

// Enabled reports whether OAuth is configured.
func (a AuthConfig) Enabled() bool { return a.GoogleClientID != "" && a.GoogleClientSecret != "" }

// RoleFor returns the role for an email: "admin", "contributor", or "" (denied).
// An empty allowlist admits any authenticated Google account as a contributor.
func (a AuthConfig) RoleFor(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	for _, e := range a.Admins {
		if strings.ToLower(strings.TrimSpace(e)) == email {
			return "admin"
		}
	}
	if len(a.Allowlist) == 0 {
		return "contributor"
	}
	for _, e := range a.Allowlist {
		if strings.ToLower(strings.TrimSpace(e)) == email {
			return "contributor"
		}
	}
	return ""
}

type DatabaseConfig struct {
	Postgres pglib.PostgresConfig `json:"postgres" yaml:"postgres"`
}

type ServerConfig struct {
	Host   string `json:"host" yaml:"host"`
	Port   int    `json:"port" yaml:"port"`
	WebDir string `json:"webDir" yaml:"webDir"`
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load reads configuration from the environment into KaziConfig.
func Load() {
	KaziConfig = Configuration{
		Database: DatabaseConfig{
			Postgres: pglib.PostgresConfig{
				Name:     env("PGDATABASE", "kazi"),
				Host:     env("PGHOST", "localhost"),
				Port:     env("PGPORT", "5432"),
				User:     env("PGUSER", "postgres"),
				Password: env("PGPASSWORD", "postgres"),
				SSLMode:  env("PGSSLMODE", "disable"),
			},
		},
		Server: ServerConfig{
			Host:   env("HTTP_HOST", "0.0.0.0"),
			Port:   5294,
			WebDir: env("WEB_DIR", "web"),
		},
		Auth: AuthConfig{
			GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:        env("OAUTH_REDIRECT_URL", "http://localhost:5294/auth/callback"),
			SessionSecret:      env("SESSION_SECRET", "dev-insecure-secret-change-me"),
			Allowlist:          splitList(os.Getenv("ALLOWLIST")),
			Admins:             splitList(os.Getenv("ADMIN_EMAILS")),
		},
		SeedPath: env("SEED_PATH", "web/family.local.json"),
	}
}

// splitList parses a comma-separated env value into a trimmed, non-empty slice.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
