package e2e

import (
	"net"
	"strings"
	"testing"
	"time"
)

// Fix round 1, task 7: runClient's pre-connect "not logged in" guard used to
// fire for ANY server address with no token. That broke self-hosted servers
// run with auth.enabled: false — they accept token-less clients (see the
// "No auth required" path in internal/server/core/auth_handler.go) and used
// to work fine before the guard was added. The guard must fire only for the
// public SaaS address; any other address dials and lets the server decide.

// cliArgsNoToken is cliArgs without --token, for testing the no-login path.
func cliArgsNoToken(h *E2EHarness, args ...string) []string {
	return append(args, "--server", h.ServerAddr, "--insecure")
}

func TestCLINoTokenAgainstAuthDisabledServer(t *testing.T) {
	h := NewHarness(t)
	h.ServerCfg.Auth.Enabled = false
	h.Start()
	defer h.Stop()

	p := startCLI(t, cliArgsNoToken(h, "http", "1")...)
	p.waitFor(t, "Ready")
	if strings.Contains(p.out.String(), "not logged in") {
		t.Errorf("refused a token-less connect against an auth-disabled server:\n%s", p.out.String())
	}
}

func TestCLINoTokenAgainstAuthEnabledServer(t *testing.T) {
	h := NewHarness(t) // Auth.Enabled: true, set by NewHarness
	h.Start()
	defer h.Stop()

	p := startCLI(t, cliArgsNoToken(h, "http", "1")...)
	if code := p.exitCode(10 * time.Second); code != 1 {
		t.Fatalf("exit code %d, want 1; output:\n%s", code, p.out.String())
	}
	if !strings.Contains(p.out.String(), "not logged in") {
		t.Errorf("missing login hint in output:\n%s", p.out.String())
	}
}

// Final review #3: a plain network failure is not a login problem — the hint
// belongs only to an auth rejection.
func TestCLINoTokenUnreachableServerNoLoginHint(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close() // nothing listens here any more

	p := startCLI(t, "http", "1", "--server", addr, "--insecure")
	if code := p.exitCode(30 * time.Second); code != 1 {
		t.Fatalf("exit code %d, want 1; output:\n%s", code, p.out.String())
	}
	if strings.Contains(p.out.String(), "not logged in") {
		t.Errorf("login hint on a network error:\n%s", p.out.String())
	}
}
