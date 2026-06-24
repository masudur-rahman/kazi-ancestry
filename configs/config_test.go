package configs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrecedence(t *testing.T) {
	yaml := `
database:
  postgres:
    host: yaml-host
    password: yaml-pass
server:
  port: 9000
auth:
  admins: [a@x.com]
seedPath: yaml/seed.json
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	CfgFile = path
	t.Cleanup(func() { CfgFile = "" })

	// env overrides the YAML password (credential priority); host stays from YAML.
	t.Setenv("PGPASSWORD", "env-pass")
	t.Setenv("ADMIN_EMAILS", "boss@x.com")

	Load()

	if got := KaziConfig.Database.Postgres.Host; got != "yaml-host" {
		t.Errorf("host: got %q, want yaml-host (from file)", got)
	}
	if got := KaziConfig.Database.Postgres.Password; got != "env-pass" {
		t.Errorf("password: got %q, want env-pass (env overrides file)", got)
	}
	if got := KaziConfig.Server.Port; got != 9000 {
		t.Errorf("port: got %d, want 9000 (from file)", got)
	}
	if got := KaziConfig.Database.Postgres.Name; got != "kazi" {
		t.Errorf("name: got %q, want kazi (built-in default)", got)
	}
	if got := KaziConfig.Auth.Admins; len(got) != 1 || got[0] != "boss@x.com" {
		t.Errorf("admins: got %v, want [boss@x.com] (env overrides file)", got)
	}
}
