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
		"nobody@x.com":  "", // denied
	}
	for email, want := range cases {
		if got := cfg.RoleFor(email); got != want {
			t.Errorf("RoleFor(%q) = %q, want %q", email, got, want)
		}
	}

	// empty allowlist admits any authenticated account as contributor
	open := AuthConfig{Admins: []string{"boss@x.com"}}
	if got := open.RoleFor("anyone@x.com"); got != "contributor" {
		t.Errorf("open allowlist: got %q, want contributor", got)
	}
}
