package api

import (
	"net/http"
	"testing"
)

// The install scripts are piped into a shell, and Go's Host validation allows
// characters a shell acts on ($ ( ) & ;). A request with Host "$(id).example"
// would otherwise produce a script containing command substitution that runs on
// the victim's machine.
func TestInstallScriptHostRejectsShellMetacharacters(t *testing.T) {
	srv := &Server{baseDomain: "fxtun.ru"}

	hostile := []string{
		"$(id).fxtun.ru",
		"a;id;.fxtun.ru",
		"a&calc.fxtun.ru",
		"a`id`.fxtun.ru",
		"a'x'.fxtun.ru",
		"",
	}
	for _, h := range hostile {
		r := newHostRequest(h)
		if got := srv.installScriptHost(r); got != "fxtun.ru" {
			t.Errorf("installScriptHost(%q) = %q, want the configured domain", h, got)
		}
	}

	for _, h := range []string{"fxtun.ru", "www.fxtun.ru", "my-host.example.com"} {
		r := newHostRequest(h)
		if got := srv.installScriptHost(r); got != h {
			t.Errorf("installScriptHost(%q) = %q, want it unchanged", h, got)
		}
	}
}

func newHostRequest(host string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "http://placeholder/install.sh", nil)
	r.Host = host
	return r
}
