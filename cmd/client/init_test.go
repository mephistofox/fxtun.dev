package main

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/mephistofox/fxtun.dev/internal/config"
)

// B16: closed stdin must stop init with an error, not loop forever re-asking
// on the first invalid-input branch (empty type is invalid, so the old code
// spun forever printing "Invalid type").
func TestRunInitInteractive_EOFStopsPromptLoop(t *testing.T) {
	isolate(t)
	scanner := bufio.NewScanner(strings.NewReader(""))

	err := runInitInteractive(scanner)
	if err != errInputClosed {
		t.Fatalf("got err=%v, want errInputClosed", err)
	}
	if _, statErr := os.Stat(projectConfigFile); statErr == nil {
		t.Fatal("fxtunnel.yaml written despite closed stdin")
	}
}

// B16: EOF partway through a tunnel's prompts (after picking a type) must
// also stop cleanly rather than loop on the next prompt.
func TestRunInitInteractive_EOFMidTunnelStops(t *testing.T) {
	isolate(t)
	scanner := bufio.NewScanner(strings.NewReader("http\n"))

	if err := runInitInteractive(scanner); err != errInputClosed {
		t.Fatalf("got err=%v, want errInputClosed", err)
	}
}

// B17: adding a tunnel to an existing fxtunnel.yaml must preserve every other
// top-level section and its comments, not just the tunnels the old code
// happened to decode.
func TestRunInitInteractive_AddPreservesOtherSections(t *testing.T) {
	isolate(t)
	existing := `# project config
server:
  address: 127.0.0.1:7000 # dev server
tunnels:
  - name: web
    type: http
    local_port: 8080
`
	if err := os.WriteFile(projectConfigFile, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	input := "a\nhttp\napi\n9090\n\nn\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	if err := runInitInteractive(scanner); err != nil {
		t.Fatalf("runInitInteractive: %v", err)
	}

	data, err := os.ReadFile(projectConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	if !strings.Contains(out, "# project config") {
		t.Errorf("top-level comment lost:\n%s", out)
	}
	if !strings.Contains(out, "127.0.0.1:7000") || !strings.Contains(out, "# dev server") {
		t.Errorf("server section lost:\n%s", out)
	}

	var cfg struct {
		Server  struct{ Address string } `yaml:"server"`
		Tunnels []config.TunnelConfig    `yaml:"tunnels"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Address != "127.0.0.1:7000" {
		t.Errorf("server.address = %q, want 127.0.0.1:7000", cfg.Server.Address)
	}
	if len(cfg.Tunnels) != 2 || cfg.Tunnels[0].Name != "web" || cfg.Tunnels[1].Name != "api" {
		t.Errorf("tunnels = %+v, want [web api]", cfg.Tunnels)
	}
}

// SUSPECT: init must reject an out-of-range remote port instead of writing
// fxtunnel.yaml with remote_port: 99999 (or a negative value).
func TestRunInitInteractive_RejectsOutOfRangeRemotePort(t *testing.T) {
	isolate(t)
	// 1st attempt: tcp, default name, port 22, remote port 99999 -> invalid,
	// discards the whole tunnel and restarts at the type prompt.
	// 2nd attempt: same, remote port 2222 -> valid. Then no more tunnels.
	input := "tcp\n\n22\n99999\ntcp\n\n22\n2222\nn\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	if err := runInitInteractive(scanner); err != nil {
		t.Fatalf("runInitInteractive: %v", err)
	}

	data, err := os.ReadFile(projectConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	var cfg projectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Tunnels) != 1 {
		t.Fatalf("got %d tunnels, want 1 (retry after invalid remote port must not add a second)", len(cfg.Tunnels))
	}
	if cfg.Tunnels[0].RemotePort != 2222 {
		t.Errorf("remote_port = %d, want 2222", cfg.Tunnels[0].RemotePort)
	}
}
