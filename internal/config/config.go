package config

import (
	"os"

	pglib "github.com/masudur-rahman/styx/sql/postgres/lib"
)

// Config holds runtime configuration sourced from the environment.
type Config struct {
	Postgres pglib.PostgresConfig
	SeedPath string // path to the integer-id seed JSON used to (re)initialize the DB
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load reads configuration from the environment, applying dev-friendly defaults.
func Load() Config {
	return Config{
		Postgres: pglib.PostgresConfig{
			Name:     env("PGDATABASE", "kazi"),
			Host:     env("PGHOST", "localhost"),
			Port:     env("PGPORT", "5432"),
			User:     env("PGUSER", "postgres"),
			Password: env("PGPASSWORD", "postgres"),
			SSLMode:  env("PGSSLMODE", "disable"),
		},
		SeedPath: env("SEED_PATH", "web/family.local.json"),
	}
}
