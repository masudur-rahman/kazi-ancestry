package configs

import (
	"os"
	"strconv"
	"strings"

	"github.com/masudur-rahman/kazi-ancestry/infra/logr"

	pglib "github.com/masudur-rahman/styx/sql/postgres/lib"
	"gopkg.in/yaml.v3"
)

var (
	// KaziConfig is the process-wide configuration, populated by Load.
	KaziConfig Configuration
	// CfgFile is the YAML config path (set by the --config flag); falls back to
	// $CONFIG_FILE, then config.yaml.
	CfgFile string
)

type Configuration struct {
	Database DatabaseConfig `json:"database" yaml:"database"`
	Server   ServerConfig   `json:"server" yaml:"server"`
	Auth     AuthConfig     `json:"auth" yaml:"auth"`
	SeedPath string         `json:"seedPath" yaml:"seedPath"`
}

type DatabaseConfig struct {
	Postgres pglib.PostgresConfig `json:"postgres" yaml:"postgres"`
}

type ServerConfig struct {
	Host   string `json:"host" yaml:"host"`
	Port   int    `json:"port" yaml:"port"`
	WebDir string `json:"webDir" yaml:"webDir"`
}

// AuthConfig holds Google OAuth + allowlist settings. An empty GoogleClientID
// disables auth (dev mode): every request is treated as an admin so the app is
// usable without OAuth credentials.
type AuthConfig struct {
	GoogleClientID     string   `json:"googleClientId" yaml:"googleClientId"`
	GoogleClientSecret string   `json:"googleClientSecret" yaml:"googleClientSecret"`
	RedirectURL        string   `json:"redirectUrl" yaml:"redirectUrl"`
	SessionSecret      string   `json:"sessionSecret" yaml:"sessionSecret"`
	Allowlist          []string `json:"allowlist" yaml:"allowlist"` // emails allowed as contributors
	Admins             []string `json:"admins" yaml:"admins"`       // emails granted admin
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

// Load builds KaziConfig with precedence: built-in defaults < YAML file < env.
// Env wins so deployments can keep credentials out of the committed YAML.
func Load() {
	cfg := defaults()

	path := CfgFile
	if path == "" {
		path = env("CONFIG_FILE", "config.yaml")
	}
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			logr.DefaultLogger.Warnf("config: parse %s: %v", path, err)
		}
	} else if CfgFile != "" {
		logr.DefaultLogger.Warnf("config: read %s: %v", path, err)
	}

	applyEnvOverrides(&cfg)
	KaziConfig = cfg
}

func defaults() Configuration {
	return Configuration{
		Database: DatabaseConfig{Postgres: pglib.PostgresConfig{
			Name: "kazi", Host: "localhost", Port: "5432",
			User: "postgres", Password: "postgres", SSLMode: "disable",
		}},
		Server:   ServerConfig{Host: "0.0.0.0", Port: 5294, WebDir: "web"},
		Auth:     AuthConfig{RedirectURL: "http://localhost:5294/auth/callback", SessionSecret: "dev-insecure-secret-change-me"},
		SeedPath: "web/family.local.json",
	}
}

// applyEnvOverrides lets environment variables override YAML values. Only
// non-empty env vars take effect, so partial env config is fine.
func applyEnvOverrides(cfg *Configuration) {
	pg := &cfg.Database.Postgres
	setStr(&pg.Name, "PGDATABASE")
	setStr(&pg.Host, "PGHOST")
	setStr(&pg.Port, "PGPORT")
	setStr(&pg.User, "PGUSER")
	setStr(&pg.Password, "PGPASSWORD")
	setStr(&pg.SSLMode, "PGSSLMODE")

	setStr(&cfg.Server.Host, "HTTP_HOST")
	setInt(&cfg.Server.Port, "HTTP_PORT_INTERNAL")
	setStr(&cfg.Server.WebDir, "WEB_DIR")

	setStr(&cfg.Auth.GoogleClientID, "GOOGLE_CLIENT_ID")
	setStr(&cfg.Auth.GoogleClientSecret, "GOOGLE_CLIENT_SECRET")
	setStr(&cfg.Auth.RedirectURL, "OAUTH_REDIRECT_URL")
	setStr(&cfg.Auth.SessionSecret, "SESSION_SECRET")
	if v := os.Getenv("ALLOWLIST"); v != "" {
		cfg.Auth.Allowlist = splitList(v)
	}
	if v := os.Getenv("ADMIN_EMAILS"); v != "" {
		cfg.Auth.Admins = splitList(v)
	}

	setStr(&cfg.SeedPath, "SEED_PATH")
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func setStr(dst *string, key string) {
	if v := os.Getenv(key); v != "" {
		*dst = v
	}
}

func setInt(dst *int, key string) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*dst = n
		}
	}
}

// splitList parses a comma-separated value into a trimmed, non-empty slice.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
