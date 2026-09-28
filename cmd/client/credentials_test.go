package main

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/mephistofox/fxtun.dev/internal/config"
)

// resetGlobals clears the package-level flag vars so tests don't leak into
// each other, restoring them on cleanup.
func resetGlobals(t *testing.T) {
	t.Helper()
	prevTok, prevSrv, prevCfg := token, serverAddr, configFile
	prevIns, prevNoInsp, prevInspAddr := insecureFlag, noInspect, inspectAddr
	prevLvl, prevFmt, prevGlobal := logLevel, logFormat, zerolog.GlobalLevel()
	token, serverAddr, configFile = "", "", ""
	t.Cleanup(func() {
		token, serverAddr, configFile = prevTok, prevSrv, prevCfg
		insecureFlag, noInspect, inspectAddr = prevIns, prevNoInsp, prevInspAddr
		logLevel, logFormat = prevLvl, prevFmt
		zerolog.SetGlobalLevel(prevGlobal)
	})
}

// isolate points HOME and the keyring at nothing so a test can never read the
// developer's real saved login (an empty DBUS_SESSION_BUS_ADDRESS is not
// enough: dbus falls back to /run/user/$UID/bus). It also clears the env
// overrides and runs the test in an empty directory.
func isolate(t *testing.T) string {
	t.Helper()
	resetGlobals(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/fxt-test-bus")
	for _, k := range []string{"FXTUNNEL_TOKEN", "FXTUNNEL_SERVER_TOKEN", "FXTUNNEL_SERVER_ADDRESS"} {
		t.Setenv(k, "")
	}
	t.Chdir(t.TempDir())
	return home
}

// flagCmd builds a command carrying the global flags that loadConfig reads,
// parsed from args, so Changed() behaves as on the real CLI.
func flagCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	f := cmd.Flags()
	f.StringVarP(&serverAddr, "server", "s", "", "")
	f.StringVarP(&token, "token", "t", "", "")
	f.StringVar(&logLevel, "log-level", "warn", "")
	f.StringVar(&logFormat, "log-format", "console", "")
	f.StringVar(&inspectAddr, "inspect-addr", "", "")
	f.BoolVar(&noInspect, "no-inspect", false, "")
	f.BoolVar(&insecureFlag, "insecure", false, "")
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func writeTestFile(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// saveLogin writes the keyring-less saved login file ~/.fxtunnel/client.yaml.
func saveLogin(t *testing.T, home, content string) {
	t.Helper()
	writeTestFile(t, filepath.Join(home, ".fxtunnel", "client.yaml"), content)
}

const projTunnels = "tunnels:\n  - {name: w, type: http, local_port: 8080}\n"

// B1: the saved login must not override server/token from the project config.
func TestLoadConfig_ProjectBeatsSavedLogin(t *testing.T) {
	home := isolate(t)
	saveLogin(t, home, "server:\n  address: saved.example.com:4443\n  token: saved_tok\n")
	configFile = writeTestFile(t, "proj.yaml",
		"server:\n  address: 127.0.0.1:7000\n  token: proj_tok\n"+projTunnels)

	cfg, _, err := loadConfig(flagCmd(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Address != "127.0.0.1:7000" || cfg.Server.Token != "proj_tok" {
		t.Fatalf("got addr=%q token=%q, want 127.0.0.1:7000 / proj_tok", cfg.Server.Address, cfg.Server.Token)
	}
}

// B1 + final review #1: a saved login's token goes only to the server it was
// saved for. A token-only login is for the public server, so a project that
// names its own server keeps its address and gets no token.
func TestLoadConfig_SavedTokenOnlyKeepsProjectAddress(t *testing.T) {
	home := isolate(t)
	saveLogin(t, home, "server:\n  token: saved_tok\n")
	configFile = writeTestFile(t, "proj.yaml", "server:\n  address: 127.0.0.1:7000\n"+projTunnels)

	cfg, _, err := loadConfig(flagCmd(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Address != "127.0.0.1:7000" || cfg.Server.Token != "" {
		t.Fatalf("got addr=%q token=%q, want 127.0.0.1:7000 / no token", cfg.Server.Address, cfg.Server.Token)
	}
}

// Final review #1 (security): a project fxtunnel.yaml naming another server
// must not receive the saved token, for quick commands and config runs alike.
func TestLoadConfig_SavedTokenNotSentToProjectServer(t *testing.T) {
	home := isolate(t)
	saveLogin(t, home, "server:\n  address: saved.example.com:4443\n  token: saved_tok\n")
	writeTestFile(t, "fxtunnel.yaml", "server:\n  address: evil.example:4443\n"+projTunnels)

	quick := []config.TunnelConfig{{Name: "http-3000", Type: "http", LocalPort: 3000}}
	for name, tunnels := range map[string][]config.TunnelConfig{"quick": quick, "config": nil} {
		cfg, _, err := loadConfig(flagCmd(t), tunnels)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Server.Address != "evil.example:4443" || cfg.Server.Token != "" {
			t.Fatalf("%s: got addr=%q token=%q, want evil.example:4443 / no token", name, cfg.Server.Address, cfg.Server.Token)
		}
	}
}

// The saved token is still used when the project names the saved server
// (a bare host gets the default port first), and for the public server when
// the login has no address.
func TestLoadConfig_SavedTokenForItsOwnServer(t *testing.T) {
	home := isolate(t)
	saveLogin(t, home, "server:\n  address: saved.example.com:4443\n  token: saved_tok\n")
	configFile = writeTestFile(t, "proj.yaml", "server:\n  address: saved.example.com\n"+projTunnels)
	cfg, _, err := loadConfig(flagCmd(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Token != "saved_tok" {
		t.Fatalf("same server: got token %q, want saved_tok", cfg.Server.Token)
	}

	saveLogin(t, home, "server:\n  token: saved_tok\n")
	configFile = writeTestFile(t, "proj.yaml", projTunnels)
	cfg, _, err = loadConfig(flagCmd(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !config.IsPublicAddress(cfg.Server.Address) || cfg.Server.Token != "saved_tok" {
		t.Fatalf("public: got addr=%q token=%q, want public / saved_tok", cfg.Server.Address, cfg.Server.Token)
	}
}

// --server pointing elsewhere must not carry the saved token either, also
// when ~/.fxtunnel/client.yaml is picked up as the config file itself.
func TestLoadConfig_ServerFlagDropsSavedToken(t *testing.T) {
	home := isolate(t)
	saveLogin(t, home, "server:\n  address: saved.example.com:4443\n  token: saved_tok\n"+projTunnels)

	cfg, _, err := loadConfig(flagCmd(t, "--server", "other.example:4443"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Token != "" {
		t.Fatalf("got token %q for --server other.example, want none", cfg.Server.Token)
	}
}

// The saved login still fills what the project leaves out.
func TestLoadConfig_SavedLoginFillsGaps(t *testing.T) {
	home := isolate(t)
	saveLogin(t, home, "server:\n  address: saved.example.com:4443\n  token: saved_tok\n")
	configFile = writeTestFile(t, "proj.yaml", projTunnels)

	cfg, _, err := loadConfig(flagCmd(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Address != "saved.example.com:4443" || cfg.Server.Token != "saved_tok" {
		t.Fatalf("got addr=%q token=%q, want saved.example.com:4443 / saved_tok", cfg.Server.Address, cfg.Server.Token)
	}
	if cfg.Server.FallbackAddress != "" {
		t.Fatalf("self-hosted saved address got public fallback %q", cfg.Server.FallbackAddress)
	}
}

// Env beats the project file; flags that were set beat both.
func TestLoadConfig_FlagsAndEnvBeatProject(t *testing.T) {
	isolate(t)
	configFile = writeTestFile(t, "proj.yaml",
		"server:\n  address: 127.0.0.1:7000\n  token: proj_tok\n"+projTunnels)

	t.Setenv("FXTUNNEL_TOKEN", "env_tok")
	t.Setenv("FXTUNNEL_SERVER_ADDRESS", "env.example.com:4443")
	cfg, _, err := loadConfig(flagCmd(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Address != "env.example.com:4443" || cfg.Server.Token != "env_tok" {
		t.Fatalf("env: got addr=%q token=%q", cfg.Server.Address, cfg.Server.Token)
	}

	cfg, _, err = loadConfig(flagCmd(t, "--server", "flag.example.com", "--token", "flag_tok", "--insecure", "--no-inspect"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Address != "flag.example.com:4443" || cfg.Server.Token != "flag_tok" || !cfg.Server.Insecure || cfg.Inspect.Enabled {
		t.Fatalf("flags: got %+v inspect=%v", cfg.Server, cfg.Inspect.Enabled)
	}
}

// B5: the logging section of the config file applies unless flags override it.
func TestLoadConfig_LoggingFromFile(t *testing.T) {
	isolate(t)
	configFile = writeTestFile(t, "proj.yaml",
		"server:\n  address: 127.0.0.1:7000\nlogging:\n  level: debug\n"+projTunnels)

	if _, _, err := loadConfig(flagCmd(t), nil); err != nil {
		t.Fatal(err)
	}
	if got := zerolog.GlobalLevel(); got != zerolog.DebugLevel {
		t.Fatalf("file logging.level ignored: global level %v, want debug", got)
	}

	if _, _, err := loadConfig(flagCmd(t, "--log-level", "error"), nil); err != nil {
		t.Fatal(err)
	}
	if got := zerolog.GlobalLevel(); got != zerolog.ErrorLevel {
		t.Fatalf("--log-level lost to the file: global level %v, want error", got)
	}
}

// SUSPECT: `--log-level bogus` was silently treated as info instead of
// rejected.
func TestLoadConfig_InvalidLogLevel(t *testing.T) {
	isolate(t)
	configFile = writeTestFile(t, "proj.yaml",
		"server:\n  address: 127.0.0.1:7000\n  token: t\n"+projTunnels)

	if _, _, err := loadConfig(flagCmd(t, "--log-level", "bogus"), nil); err == nil {
		t.Fatal("expected error for invalid --log-level")
	}

	// A bad level from the file is not a --log-level flag problem.
	configFile = writeTestFile(t, "proj.yaml",
		"server:\n  address: 127.0.0.1:7000\n  token: t\nlogging:\n  level: bogus\n"+projTunnels)
	_, _, err := loadConfig(flagCmd(t), nil)
	if err == nil || strings.Contains(err.Error(), "--log-level") {
		t.Fatalf("got %v, want an invalid log level error not naming the flag", err)
	}
}

// SUSPECT: with no token anywhere, the client used to dial the server and
// only learn about it from the server's "invalid token" reply. runClient
// must refuse before connecting, with a message pointing at `fxtunnel
// login` — but only for the public server, which always requires a token.
// A self-hosted --server may run with auth disabled (fix round 1, task 7:
// this used to block that case too); that half is covered by the CLI-binary
// tests in internal/e2e (TestCLINoTokenAgainstAuthDisabledServer,
// TestCLINoTokenAgainstAuthEnabledServer), since it needs a real dial.
func TestRunClient_NoToken(t *testing.T) {
	cfg := &config.ClientConfig{
		Server:  config.ClientServerSettings{Address: config.DefaultServerAddress},
		Tunnels: []config.TunnelConfig{{Name: "w", Type: "http", LocalPort: 8080}},
	}
	err := runClient(cfg, zerolog.Nop())
	if !errors.Is(err, errNotLoggedIn) {
		t.Fatalf("got %v, want errNotLoggedIn", err)
	}
}

// A quick command validates only its own tunnel: a broken unrelated entry in
// the project file must not block it, but still fails the config-file run.
func TestLoadConfig_QuickTunnelSkipsFileTunnels(t *testing.T) {
	isolate(t)
	configFile = writeTestFile(t, "proj.yaml", "server:\n  address: 127.0.0.1:7000\n"+
		"tunnels:\n  - {name: bad, type: http, local_port: 8080, auto_close: 2s}\n")

	quick := config.TunnelConfig{Name: "http-3000", Type: "http", LocalPort: 3000, BasicAuthHash: "$2a$hash"}
	cfg, _, err := loadConfig(flagCmd(t), []config.TunnelConfig{quick})
	if err != nil {
		t.Fatalf("quick command blocked by an unrelated file tunnel: %v", err)
	}
	if len(cfg.Tunnels) != 1 || cfg.Tunnels[0].Name != "http-3000" || cfg.Tunnels[0].BasicAuthHash != "$2a$hash" {
		t.Fatalf("got tunnels %+v, want only the quick tunnel with its auth hash", cfg.Tunnels)
	}

	if _, _, err := loadConfig(flagCmd(t), nil); err == nil {
		t.Fatal("config-file run accepted auto_close: 2s")
	}
}

// B12: the daemon must honour --insecure (and the other global flags). The
// fake server records the first byte the client sends: 0x16 is a TLS
// ClientHello, anything else is the plaintext compression preference.
func TestRunDaemonForeground_HonoursInsecure(t *testing.T) {
	isolate(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	first := make(chan byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := []byte{0}
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if _, err := conn.Read(buf); err == nil {
			first <- buf[0]
		}
	}()
	configFile = writeTestFile(t, "proj.yaml",
		"server:\n  address: "+ln.Addr().String()+"\n  token: x\n"+projTunnels)

	if err := runDaemonForeground(flagCmd(t, "--insecure")); err == nil {
		t.Fatal("expected connect failure against the fake server")
	}
	select {
	case b := <-first:
		if b == 0x16 {
			t.Fatal("daemon ignored --insecure: sent a TLS ClientHello")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("fake server saw no bytes")
	}
}

// Final review #1: --server naming another server must not get the saved
// token (e.g. `domains` sending it as a Bearer header); naming the saved
// server, bare host included, still does.
func TestCheckAuth_ServerFlagWithSavedToken(t *testing.T) {
	home := isolate(t)
	saveLogin(t, home, "server:\n  address: saved.example.com:4443\n  token: saved_tok\n")

	serverAddr = "flag.example.com:4443"
	if tok, _, ok := checkAuth(); ok || tok != "" {
		t.Fatalf("got (%q, %v), want no saved token for another server", tok, ok)
	}

	serverAddr = "saved.example.com"
	if tok, addr, ok := checkAuth(); !ok || tok != "saved_tok" || addr != "saved.example.com" {
		t.Fatalf("got (%q, %q, %v), want (saved_tok, saved.example.com, true)", tok, addr, ok)
	}
}

func TestCredentialsFromFile_NoDefaultAddress(t *testing.T) {
	home := isolate(t)
	saveLogin(t, home, "server:\n  token: saved_tok\n")

	tok, addr, ok := credentialsFromFile()
	if !ok || tok != "saved_tok" || addr != "" {
		t.Fatalf("got (%q, %q, %v), want (saved_tok, \"\", true)", tok, addr, ok)
	}
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
