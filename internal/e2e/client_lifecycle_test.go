package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	clientcore "github.com/mephistofox/fxtun.dev/internal/client/core"
	"github.com/mephistofox/fxtun.dev/internal/config"
	servercore "github.com/mephistofox/fxtun.dev/internal/server/core"
)

// A tunnel the server rejects must fail Connect at once with the server's reason,
// not after the 30 s response timeout.
func TestClientRejectedTunnelFailsFast(t *testing.T) {
	h := NewHarness(t)
	h.Start()
	defer h.Stop()

	h.ConnectClient([]config.TunnelConfig{{Name: "a", Type: "http", LocalPort: 1, Subdomain: "taken"}})

	c := h.NewClient([]config.TunnelConfig{{Name: "b", Type: "http", LocalPort: 1, Subdomain: "taken"}})
	defer c.Close()
	start := time.Now()
	err := c.Connect()
	if err == nil {
		t.Fatalf("Connect succeeded although the only tunnel was rejected")
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("Connect took %s, want < 5s", d)
	}
	if !strings.Contains(err.Error(), "subdomain") {
		t.Errorf("error %q does not carry the server's reason", err)
	}
}

// One rejected tunnel out of two: Connect succeeds with the other one.
func TestClientPartlyRejectedTunnelsConnect(t *testing.T) {
	h := NewHarness(t)
	h.Start()
	defer h.Stop()

	h.ConnectClient([]config.TunnelConfig{{Name: "a", Type: "http", LocalPort: 1, Subdomain: "taken"}})

	c := h.ConnectClient([]config.TunnelConfig{
		{Name: "b", Type: "http", LocalPort: 1, Subdomain: "taken"},
		{Name: "c", Type: "http", LocalPort: 1, Subdomain: "free"},
	})
	if n := len(c.GetTunnels()); n != 1 {
		t.Errorf("got %d tunnels, want 1", n)
	}
}

// With reconnect disabled a lost server closes the client, and Done reports it.
func TestClientDoneWhenReconnectDisabled(t *testing.T) {
	h := NewHarness(t)
	h.Start()

	c := h.ConnectClient([]config.TunnelConfig{{Name: "a", Type: "http", LocalPort: 1}})
	// Stop blocks ~2 s after announcing the shutdown; time from the announcement.
	start := time.Now()
	stopped := make(chan struct{})
	go func() { h.Stop(); close(stopped) }()
	defer func() { <-stopped }()

	select {
	case <-c.Done():
		if d := time.Since(start); d > 5*time.Second {
			t.Errorf("client done after %s, want < 5s", d)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("client not done after the server went away with reconnect disabled")
	}
}

// A token the server no longer accepts ends reconnecting after the first attempt.
func TestClientStopsReconnectingOnRejectedToken(t *testing.T) {
	h := NewHarness(t)
	h.Start()

	cfg := &config.ClientConfig{
		Server:    config.ClientServerSettings{Address: h.ServerAddr, Token: h.Token, Insecure: true},
		Tunnels:   []config.TunnelConfig{{Name: "a", Type: "http", LocalPort: 1}},
		Reconnect: config.ReconnectSettings{Enabled: true, Interval: 100 * time.Millisecond},
	}
	c := clientcore.New(cfg, h.log)
	var attempts atomic.Int32
	c.Events().Subscribe(func(e clientcore.Event) {
		if e.Type == clientcore.EventReconnecting {
			attempts.Add(1)
		}
	})
	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	// Same ports, but the client's token is gone.
	h.Stop()
	cfg2 := *h.ServerCfg
	cfg2.Auth.Tokens = []config.TokenConfig{{Name: "other", Token: "sk_e2e_other_token", AllowedSubdomains: []string{"*"}, MaxTunnels: 20}}
	srv := servercore.New(&cfg2, h.log)
	if err := srv.Start(); err != nil {
		t.Fatalf("restart server: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	select {
	case <-c.Done():
	case <-time.After(15 * time.Second):
		t.Fatalf("client still reconnecting with a rejected token (%d attempts)", attempts.Load())
	}
	time.Sleep(500 * time.Millisecond)
	if n := attempts.Load(); n > 1 {
		t.Errorf("%d reconnect attempts, want 1", n)
	}
}

// Tunnels closed on purpose (auto-close, server/admin close) stay closed after a
// reconnect; open ones come back.
func TestClientReconnectSkipsClosedTunnels(t *testing.T) {
	h := NewHarness(t)
	h.Start()

	cfg := &config.ClientConfig{
		Server: config.ClientServerSettings{Address: h.ServerAddr, Token: h.Token, Insecure: true},
		Tunnels: []config.TunnelConfig{
			{Name: "keep", Type: "http", LocalPort: 1, Subdomain: "keep"},
			{Name: "idle", Type: "http", LocalPort: 1, Subdomain: "idle", AutoClose: testAutoClose},
			{Name: "gone", Type: "http", LocalPort: 1, Subdomain: "gone"},
		},
		Reconnect: config.ReconnectSettings{Enabled: true, Interval: 200 * time.Millisecond},
	}
	c := clientcore.New(cfg, h.log)
	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()
	if n := len(c.GetTunnels()); n != 3 {
		t.Fatalf("%d tunnels open, want 3", n)
	}

	for _, tn := range h.Server.GetAllTunnels() {
		if tn.Subdomain == "gone" {
			if err := h.Server.AdminCloseTunnel(tn.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	for deadline := time.Now().Add(10 * time.Second); len(c.GetTunnels()) > 1; time.Sleep(100 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("%d tunnels still open, want only keep", len(c.GetTunnels()))
		}
	}

	h.Stop()
	srv := servercore.New(h.ServerCfg, h.log)
	if err := srv.Start(); err != nil {
		t.Fatalf("restart server: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	for deadline := time.Now().Add(15 * time.Second); len(srv.GetAllTunnels()) == 0; time.Sleep(100 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("client did not reconnect")
		}
	}
	time.Sleep(time.Second) // let any wrongly restored tunnel arrive too
	var subs []string
	for _, tn := range srv.GetAllTunnels() {
		subs = append(subs, tn.Subdomain)
	}
	if len(subs) != 1 || subs[0] != "keep" {
		t.Errorf("tunnels after reconnect: %v, want [keep]", subs)
	}
}

// ---- CLI process lifecycle ----

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

type cliProc struct {
	cmd  *exec.Cmd
	out  *syncBuf
	done chan struct{}
}

func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fxtunnel")
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/client").CombinedOutput(); err != nil {
		t.Fatalf("build cli: %v\n%s", err, out)
	}
	return bin
}

// cliEnv isolates HOME and the keyring: an empty DBUS_SESSION_BUS_ADDRESS
// falls back to the user's real session bus.
func cliEnv(home string) []string {
	return []string{"HOME=" + home, "DBUS_SESSION_BUS_ADDRESS=unix:path=/nonexistent/fxt-test-bus", "PATH=" + os.Getenv("PATH")}
}

// startCLI builds the CLI and runs it with an isolated HOME and keyring.
func startCLI(t *testing.T, args ...string) *cliProc {
	t.Helper()
	bin := buildCLI(t)
	home := t.TempDir()
	cmd := exec.Command(bin, args...)
	cmd.Dir = home
	cmd.Env = cliEnv(home)
	p := &cliProc{cmd: cmd, out: &syncBuf{}, done: make(chan struct{})}
	cmd.Stdout, cmd.Stderr = p.out, p.out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = cmd.Wait(); close(p.done) }()
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGINT)
		select {
		case <-p.done:
		case <-time.After(8 * time.Second):
			_ = cmd.Process.Kill()
		}
	})
	return p
}

// exitCode waits up to d for the process to exit and returns its code, or -1.
func (p *cliProc) exitCode(d time.Duration) int {
	select {
	case <-p.done:
		return p.cmd.ProcessState.ExitCode()
	case <-time.After(d):
		return -1
	}
}

func (p *cliProc) waitFor(t *testing.T, sub string) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if strings.Contains(p.out.String(), sub) {
			return
		}
	}
	t.Fatalf("no %q in output:\n%s", sub, p.out.String())
}

func cliArgs(h *E2EHarness, args ...string) []string {
	return append(args, "--server", h.ServerAddr, "--token", h.Token, "--insecure")
}

func TestCLIRejectedTunnelExits(t *testing.T) {
	h := NewHarness(t)
	h.Start()
	defer h.Stop()
	h.ConnectClient([]config.TunnelConfig{{Name: "a", Type: "http", LocalPort: 1, Subdomain: "taken"}})

	p := startCLI(t, cliArgs(h, "http", "1", "--subdomain", "taken")...)
	if code := p.exitCode(10 * time.Second); code != 1 {
		t.Fatalf("exit code %d, want 1; output:\n%s", code, p.out.String())
	}
	if strings.Contains(p.out.String(), "Tunnel established") {
		t.Errorf("reported an established tunnel:\n%s", p.out.String())
	}
}

func TestCLIExitsWhenClientCloses(t *testing.T) {
	h := NewHarness(t)
	h.Start()
	cfgPath := filepath.Join(t.TempDir(), "fxtunnel.yaml")
	cfg := fmt.Sprintf("server:\n  address: %s\n  token: %s\n  insecure: true\nreconnect:\n  enabled: false\ntunnels:\n  - {name: w, type: http, local_port: 1}\n", h.ServerAddr, h.Token)
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	p := startCLI(t, "-c", cfgPath)
	p.waitFor(t, "Ready")
	h.Stop()
	if code := p.exitCode(15 * time.Second); code != 1 {
		t.Fatalf("exit code %d, want 1; output:\n%s", code, p.out.String())
	}
}

func TestCLIExitsWhenLastTunnelClosed(t *testing.T) {
	h := NewHarness(t)
	h.Start()
	defer h.Stop()

	p := startCLI(t, cliArgs(h, "http", "1")...)
	p.waitFor(t, "Ready")
	// Same path as auto-close/max-lifetime: the server confirms with tunnel_closed.
	for _, tn := range h.Server.GetAllTunnels() {
		if err := h.Server.AdminCloseTunnel(tn.ID); err != nil {
			t.Fatal(err)
		}
	}
	if code := p.exitCode(10 * time.Second); code != 0 {
		t.Fatalf("exit code %d, want 0; output:\n%s", code, p.out.String())
	}
}
