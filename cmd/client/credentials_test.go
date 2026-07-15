package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mephistofox/fxtunnel/internal/config"
)

// resetGlobals clears the package-level flag vars so tests don't leak into
// each other, restoring them on cleanup.
func resetGlobals(t *testing.T) {
	t.Helper()
	prevTok, prevSrv := token, serverAddr
	token, serverAddr = "", ""
	t.Cleanup(func() {
		token, serverAddr = prevTok, prevSrv
	})
}

func TestCheckAuth_FlagPriority(t *testing.T) {
	resetGlobals(t)
	t.Setenv("FXTUNNEL_TOKEN", "envtok") // must be ignored when flag is set
	token = "flagtok"
	serverAddr = "flag.example.com:443"

	tok, addr, ok := checkAuth()
	if !ok || tok != "flagtok" || addr != "flag.example.com:443" {
		t.Fatalf("flag priority: got (%q, %q, %v), want (flagtok, flag.example.com:443, true)", tok, addr, ok)
	}
}

func TestCheckAuth_EnvFallback(t *testing.T) {
	resetGlobals(t)
	// Point HOME at an empty dir so the keyring/file lookups can't succeed.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("FXTUNNEL_TOKEN", "envtok")
	t.Setenv("FXTUNNEL_SERVER_ADDRESS", "env.example.com:443")

	tok, addr, ok := checkAuth()
	if !ok || tok != "envtok" || addr != "env.example.com:443" {
		t.Fatalf("env fallback: got (%q, %q, %v), want (envtok, env.example.com:443, true)", tok, addr, ok)
	}
}

func TestSaveCredentialsToFile_RoundTrip(t *testing.T) {
	resetGlobals(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := saveCredentialsToFile("filetok", "file.example.com:443")
	if err != nil {
		t.Fatalf("saveCredentialsToFile: %v", err)
	}
	if want := filepath.Join(home, ".fxtunnel", "client.yaml"); path != want {
		t.Fatalf("path: got %q, want %q", path, want)
	}

	// File must be readable through the normal config loader.
	cfg, err := config.LoadClientConfig(path)
	if err != nil {
		t.Fatalf("LoadClientConfig: %v", err)
	}
	if cfg.Server.Token != "filetok" || cfg.Server.Address != "file.example.com:443" {
		t.Fatalf("loaded config: got token=%q addr=%q, want filetok / file.example.com:443",
			cfg.Server.Token, cfg.Server.Address)
	}

	// Perms must be 0600 — the file holds a plaintext token.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("perms: got %o, want 600", perm)
	}
}

func TestSaveCredentialsToFile_PreservesExisting(t *testing.T) {
	resetGlobals(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".fxtunnel")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "client.yaml")
	existing := "tunnels:\n  - name: web\n    type: http\n    local_port: 8080\n"
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := saveCredentialsToFile("filetok", ""); err != nil {
		t.Fatalf("saveCredentialsToFile: %v", err)
	}

	cfg, err := config.LoadClientConfig(path)
	if err != nil {
		t.Fatalf("LoadClientConfig: %v", err)
	}
	if cfg.Server.Token != "filetok" {
		t.Fatalf("token not saved: got %q", cfg.Server.Token)
	}
	if len(cfg.Tunnels) != 1 || cfg.Tunnels[0].Name != "web" {
		t.Fatalf("existing tunnels not preserved: %+v", cfg.Tunnels)
	}
}

func TestSaveCredentialsToFile_CorruptYAML(t *testing.T) {
	resetGlobals(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".fxtunnel")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "client.yaml")
	// Invalid YAML must not be silently clobbered.
	if err := os.WriteFile(path, []byte("server: [this is: broken"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := saveCredentialsToFile("filetok", ""); err == nil {
		t.Fatal("expected error on corrupt existing YAML, got nil")
	}
}

func TestClearCredentialsFile(t *testing.T) {
	resetGlobals(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	// No file yet: must be a no-op, not an error.
	if err := clearCredentialsFile(); err != nil {
		t.Fatalf("clear on missing file: %v", err)
	}

	// Seed a file with a token plus an unrelated tunnel.
	if _, err := saveCredentialsToFile("filetok", "file.example.com:443"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".fxtunnel", "client.yaml")
	data, _ := os.ReadFile(path)
	merged := string(data) + "tunnels:\n  - name: web\n    type: http\n    local_port: 8080\n"
	if err := os.WriteFile(path, []byte(merged), 0600); err != nil {
		t.Fatal(err)
	}

	if err := clearCredentialsFile(); err != nil {
		t.Fatalf("clearCredentialsFile: %v", err)
	}

	// Token must be gone; the tunnel must survive.
	tok, _, ok := credentialsFromFile()
	if ok || tok != "" {
		t.Fatalf("token not cleared: got tok=%q ok=%v", tok, ok)
	}
	cfg, err := config.LoadClientConfig(path)
	if err != nil {
		t.Fatalf("LoadClientConfig after clear: %v", err)
	}
	if len(cfg.Tunnels) != 1 || cfg.Tunnels[0].Name != "web" {
		t.Fatalf("tunnels not preserved after clear: %+v", cfg.Tunnels)
	}
}
