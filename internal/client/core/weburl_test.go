package core

import "testing"

func TestWebHost(t *testing.T) {
	cases := map[string]string{
		"tunnel.fxtun.dev:443": "fxtun.dev",
		"tunnel.fxtun.ru:443":  "fxtun.ru",
		"tunnel.fxtun.dev":     "fxtun.dev",
		"fxtun.dev:4443":       "fxtun.dev",
		"fxtun.dev":            "fxtun.dev",
		"localhost:4443":       "localhost",
		"my.host.example:443":  "my.host.example",
	}
	for in, want := range cases {
		if got := WebHost(in); got != want {
			t.Errorf("WebHost(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWebBaseURL(t *testing.T) {
	if got := WebBaseURL("tunnel.fxtun.dev:443"); got != "https://fxtun.dev" {
		t.Errorf("WebBaseURL = %q, want https://fxtun.dev", got)
	}
}
