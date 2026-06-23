package configs

import (
	"os"

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
	SeedPath string         `json:"seedPath" yaml:"seedPath"`
}

type DatabaseConfig struct {
	Postgres pglib.PostgresConfig `json:"postgres" yaml:"postgres"`
}

type ServerConfig struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
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
			Host: env("HTTP_HOST", "0.0.0.0"),
			Port: 5294,
		},
		SeedPath: env("SEED_PATH", "web/family.local.json"),
	}
}
