package configs

import "testing"

func TestRoleFor(t *testing.T) {
	cfg := AuthConfig{
		Allowlist: []string{"Contrib@X.com", "two@x.com"},
		Admins:    []string{"boss@x.com"},
	}
	cases := map[string]string{
		"boss@x.com":    "admin",
		"contrib@x.com": "contributor", // case-insensitive
		"two@x.com":     "contributor",
		"nobody@x.com":  "viewer", // logged in, not allowlisted -> read-only
	}
	for email, want := range cases {
		if got := cfg.RoleFor(email); got != want {
			t.Errorf("RoleFor(%q) = %q, want %q", email, got, want)
		}
	}

	// empty allowlist: only admins act; everyone else is a viewer
	closed := AuthConfig{Admins: []string{"boss@x.com"}}
	if got := closed.RoleFor("anyone@x.com"); got != "viewer" {
		t.Errorf("empty allowlist: got %q, want viewer", got)
	}
}
