package e2e

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/rs/zerolog"

	clientcore "github.com/mephistofox/fxtunnel/internal/client/core"
	"github.com/mephistofox/fxtunnel/internal/config"
	servercore "github.com/mephistofox/fxtunnel/internal/server/core"
)

const (
	// testToken is a static config token for E2E tests (no database needed).
	testToken = "sk_e2e_test_token_secret_12345"

	// testDomain is the base domain used for subdomain extraction.
	testDomain = "test.local"
)

// E2EHarness manages a real server + client for integration testing.
// It uses static config tokens so no PostgreSQL database is needed.
type E2EHarness struct {
	t          *testing.T
	Server     *servercore.Server
	ServerCfg  *config.ServerConfig
	Token      string // API token for client auth
	ServerAddr string // "127.0.0.1:PORT" (control)
	HTTPAddr   string // "127.0.0.1:PORT" (HTTP tunnels)
	HTTPPort   int    // HTTP port number
	log        zerolog.Logger
}

// NewHarness creates a new E2E test harness with random ports.
func NewHarness(t *testing.T) *E2EHarness {
	t.Helper()

	controlPort := getFreePort(t)
	httpPort := getFreePort(t)
	tcpMin := getFreePort(t)
	udpMin := getFreePort(t)

	log := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.WarnLevel).
		With().Str("test", t.Name()).Logger()

	cfg := &config.ServerConfig{
		Server: config.ServerSettings{
			ControlPort: controlPort,
			HTTPPort:    httpPort,
			TCPPortRange: config.PortRange{
				Min: tcpMin,
				Max: tcpMin + 100,
			},
			UDPPortRange: config.PortRange{
				Min: udpMin,
				Max: udpMin + 100,
			},
			CompressionEnabled: false,
		},
		Domain: config.DomainSettings{
			Base:     testDomain,
			Wildcard: true,
		},
		Auth: config.AuthSettings{
			Enabled: true,
			Tokens: []config.TokenConfig{
				{
					Name:              "e2e-test",
					Token:             testToken,
					AllowedSubdomains: []string{"*"},
					MaxTunnels:        20,
				},
			},
		},
		Logging: config.LoggingSettings{
			Level:  "warn",
			Format: "console",
		},
		Inspect: config.InspectSettings{
			Enabled:     true,
			MaxEntries:  100,
			MaxBodySize: 1 << 20, // 1MB
		},
	}

	srv := servercore.New(cfg, log)
	// No database, no auth service — uses static config tokens

	return &E2EHarness{
		t:          t,
		Server:     srv,
		ServerCfg:  cfg,
		Token:      testToken,
		ServerAddr: fmt.Sprintf("127.0.0.1:%d", controlPort),
		HTTPAddr:   fmt.Sprintf("127.0.0.1:%d", httpPort),
		HTTPPort:   httpPort,
		log:        log,
	}
}

// Start starts the tunnel server and waits until it's ready.
func (h *E2EHarness) Start() {
	h.t.Helper()

	if err := h.Server.Start(); err != nil {
		h.t.Fatalf("server.Start: %v", err)
	}

	// Wait for control port to be ready
	waitForPort(h.t, h.ServerAddr, 5*time.Second)
	// Wait for HTTP port to be ready
	waitForPort(h.t, h.HTTPAddr, 5*time.Second)
}

// Stop stops the tunnel server.
func (h *E2EHarness) Stop() {
	h.t.Helper()

	if err := h.Server.Stop(); err != nil {
		h.t.Errorf("server.Stop: %v", err)
	}
}

// NewClient creates a new tunnel client configured to connect to this harness's server.
func (h *E2EHarness) NewClient(tunnels []config.TunnelConfig) *clientcore.Client {
	h.t.Helper()

	cfg := &config.ClientConfig{
		Server: config.ClientServerSettings{
			Address:     h.ServerAddr,
			Token:       h.Token,
			Insecure:    true,
			TLSVerify:   false,
			Compression: false,
		},
		Tunnels: tunnels,
		Reconnect: config.ReconnectSettings{
			Enabled: false,
		},
		Inspect: config.InspectSettings{
			Enabled: false,
		},
		Logging: config.LoggingSettings{
			Level:  "warn",
			Format: "console",
		},
	}

	client := clientcore.New(cfg, h.log)
	return client
}

// ConnectClient creates and connects a client, returning it.
// Registers cleanup to close the client on test completion.
func (h *E2EHarness) ConnectClient(tunnels []config.TunnelConfig) *clientcore.Client {
	h.t.Helper()

	client := h.NewClient(tunnels)
	if err := client.Connect(); err != nil {
		h.t.Fatalf("client.Connect: %v", err)
	}
	h.t.Cleanup(func() { client.Close() })

	// Give the server a moment to register tunnels
	time.Sleep(200 * time.Millisecond)

	return client
}

// getFreePort returns a random available TCP port.
func getFreePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("getFreePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// waitForPort waits until a TCP port accepts connections or the timeout expires.
func waitForPort(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("waitForPort: %s not ready after %v", addr, timeout)
}
