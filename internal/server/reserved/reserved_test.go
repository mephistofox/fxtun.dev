package reserved

import "testing"

// A tunnel or reservation on one of these names lets a stranger speak for the
// service: admin.fxtun.ru asking for a password needs no further explanation,
// and taking www or ns1 shadows real infrastructure.
func TestIsReserved(t *testing.T) {
	blocked := []string{
		"www", "admin", "cp", "tunnel", "tunnels", "api", "ns1", "mail",
		"login", "billing", "support", "blog", "status", "staging", "fxtun",
	}
	for _, name := range blocked {
		if !IsReserved(name) {
			t.Errorf("IsReserved(%q) = false, want true", name)
		}
	}

	// Hostnames are case-insensitive, so a check that is not would be
	// sidestepped by capitalising a letter.
	for _, name := range []string{"Admin", "WWW", "TuNnEl", "  admin  "} {
		if !IsReserved(name) {
			t.Errorf("IsReserved(%q) = false, want true — case and spacing must not matter", name)
		}
	}

	for _, name := range []string{"myapp", "johns-demo-site", "shop", "woof", "admin2", "my-api"} {
		if IsReserved(name) {
			t.Errorf("IsReserved(%q) = true, want false — ordinary names must stay available", name)
		}
	}
}
