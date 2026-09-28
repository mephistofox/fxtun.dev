package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	servercore "github.com/mephistofox/fxtun.dev/internal/server/core"
)

// daemonCLI runs CLI commands that share one isolated HOME, so they find the
// same daemon (~/.fxtunnel/daemon.json).
type daemonCLI struct {
	t    *testing.T
	bin  string
	home string
}

func newDaemonCLI(t *testing.T) *daemonCLI {
	d := &daemonCLI{t: t, bin: buildCLI(t), home: t.TempDir()}
	t.Cleanup(func() {
		// A detached daemon must never outlive the test.
		data, err := os.ReadFile(filepath.Join(d.home, ".fxtunnel", "daemon.json"))
		if err != nil {
			return
		}
		var st struct{ PID int }
		if json.Unmarshal(data, &st) == nil && st.PID > 0 {
			_ = syscall.Kill(st.PID, syscall.SIGKILL)
		}
	})
	return d
}

func (d *daemonCLI) run(args ...string) (string, int) {
	d.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, d.bin, args...)
	cmd.Dir = d.home
	cmd.Env = cliEnv(d.home)
	out, _ := cmd.CombinedOutput()
	return string(out), cmd.ProcessState.ExitCode()
}

// hotAdd adds a tunnel to the running daemon. The dead --server makes a missed
// daemon fail instead of falling back to a real connection.
func (d *daemonCLI) hotAdd(args ...string) (string, int) {
	d.t.Helper()
	return d.run(append(args, "--server", "127.0.0.1:1", "--token", "x")...)
}

func (d *daemonCLI) config(content string) string {
	d.t.Helper()
	path := filepath.Join(d.t.TempDir(), "fxtunnel.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		d.t.Fatal(err)
	}
	return path
}

func (d *daemonCLI) up(args ...string) {
	d.t.Helper()
	out, code := d.run(append([]string{"up"}, args...)...)
	if code != 0 || !strings.Contains(out, "Daemon started") {
		d.t.Fatalf("up: exit %d\n%s", code, out)
	}
}

var statusURL = regexp.MustCompile(`HTTP: (\S+)`)

// urls returns the sorted tunnel URLs `status` reports.
func (d *daemonCLI) urls() []string {
	out, _ := d.run("status")
	var urls []string
	for _, m := range statusURL.FindAllStringSubmatch(out, -1) {
		urls = append(urls, m[1])
	}
	sort.Strings(urls)
	return urls
}

// B11: `up` against a dead server says why and exits 1, the log is kept under HOME.
func TestDaemonUpFailsWhenServerUnreachable(t *testing.T) {
	d := newDaemonCLI(t)
	cfg := d.config("server:\n  address: 127.0.0.1:1\n  token: x\n  insecure: true\ntunnels:\n  - {name: w, type: http, local_port: 1}\n")

	out, code := d.run("up", "-c", cfg)
	if code != 1 {
		t.Fatalf("exit %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "connect") {
		t.Errorf("output does not say why the daemon failed:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(d.home, ".fxtunnel", "daemon.log")); err != nil {
		t.Errorf("daemon log: %v", err)
	}
}

// B12, B13, B14: the daemon honours --insecure; each hot-add prints its own URL;
// a rejected hot-add exits 1 at once with the server's reason.
func TestDaemonHotAdd(t *testing.T) {
	h := NewHarness(t)
	h.Start()
	defer h.Stop()
	d := newDaemonCLI(t)
	// No insecure in the file: only the flag lets the daemon reach the plain harness.
	cfg := d.config(fmt.Sprintf("server:\n  address: %s\n  token: %s\ntunnels:\n  - {name: base, type: http, local_port: 1, subdomain: base}\n", h.ServerAddr, h.Token))
	d.up("-c", cfg, "--insecure")

	added := regexp.MustCompile(`Tunnel added: (\S+)`)
	seen := map[string]bool{"http://base." + testDomain: true}
	for i := 0; i < 2; i++ {
		out, code := d.hotAdd("http", "1")
		m := added.FindStringSubmatch(out)
		if code != 0 || m == nil {
			t.Fatalf("hot-add %d: exit %d\n%s", i, code, out)
		}
		if seen[m[1]] {
			t.Errorf("hot-add %d printed %s, a URL of another tunnel", i, m[1])
		}
		seen[m[1]] = true
	}

	start := time.Now()
	out, code := d.hotAdd("http", "1", "-d", "base")
	if code != 1 {
		t.Errorf("rejected hot-add: exit %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "subdomain") {
		t.Errorf("rejected hot-add does not carry the server's reason:\n%s", out)
	}
	if el := time.Since(start); el > 5*time.Second {
		t.Errorf("rejected hot-add took %s, want < 5s", el)
	}
}

// B15: tunnels added to a running daemon come back after it reconnects.
func TestDaemonKeepsHotAddedTunnelsAcrossReconnect(t *testing.T) {
	h := NewHarness(t)
	h.Start()
	d := newDaemonCLI(t)
	cfg := d.config(fmt.Sprintf("server:\n  address: %s\n  token: %s\n  insecure: true\nreconnect:\n  enabled: true\n  interval: 200ms\ntunnels:\n  - {name: base, type: http, local_port: 1, subdomain: base}\n", h.ServerAddr, h.Token))
	d.up("-c", cfg)
	if out, code := d.hotAdd("http", "1"); code != 0 {
		t.Fatalf("hot-add: exit %d\n%s", code, out)
	}
	before := d.urls()
	if len(before) != 2 {
		t.Fatalf("tunnels before reconnect: %v, want 2", before)
	}

	h.Stop()
	srv := servercore.New(h.ServerCfg, h.log)
	if err := srv.Start(); err != nil {
		t.Fatalf("restart server: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Count on the new server: the daemon's own list is stale until it notices the drop.
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(300 * time.Millisecond) {
		if len(srv.GetAllTunnels()) >= 2 {
			break
		}
	}
	if after := d.urls(); strings.Join(after, " ") != strings.Join(before, " ") {
		t.Errorf("tunnels after reconnect: %v, want %v", after, before)
	}
}

// B12: reconnect.enabled: false is respected; the daemon stops with the server.
func TestDaemonStopsWhenReconnectDisabled(t *testing.T) {
	h := NewHarness(t)
	h.Start()
	d := newDaemonCLI(t)
	cfg := d.config(fmt.Sprintf("server:\n  address: %s\n  token: %s\n  insecure: true\nreconnect:\n  enabled: false\ntunnels:\n  - {name: base, type: http, local_port: 1}\n", h.ServerAddr, h.Token))
	d.up("-c", cfg)

	h.Stop()
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(300 * time.Millisecond) {
		if out, _ := d.run("status"); strings.Contains(out, "not running") {
			return
		}
	}
	t.Error("daemon still running after the server went away with reconnect disabled")
}
