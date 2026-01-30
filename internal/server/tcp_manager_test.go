package server

import (
	"net"
	"os"
	"testing"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
)

func newTestTCPManager(portMin, portMax int) (*TCPManager, *Server) {
	log := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	cfg := &config.ServerConfig{
		Server: config.ServerSettings{
			ControlPort:  14443,
			HTTPPort:     18080,
			TCPPortRange: config.PortRange{Min: portMin, Max: portMax},
			UDPPortRange: config.PortRange{Min: 31001, Max: 32000},
		},
		Domain: config.DomainSettings{Base: "test.local"},
	}
	srv := New(cfg, log)
	return srv.tcpManager, srv
}

func TestTCPAllocateSpecificPort(t *testing.T) {
	mgr, srv := newTestTCPManager(40000, 40100)
	defer srv.cancel()

	port, listener, err := mgr.AllocatePort(40050)
	if err != nil {
		t.Fatalf("AllocatePort: %v", err)
	}
	defer listener.Close()
	defer mgr.ReleasePort(port)

	if port != 40050 {
		t.Fatalf("expected port 40050, got %d", port)
	}

	// Verify listener is functional
	addr := listener.Addr().(*net.TCPAddr)
	if addr.Port != 40050 {
		t.Fatalf("listener on wrong port: %d", addr.Port)
	}
}

func TestTCPAllocateAutoAssign(t *testing.T) {
	mgr, srv := newTestTCPManager(40200, 40210)
	defer srv.cancel()

	port, listener, err := mgr.AllocatePort(0)
	if err != nil {
		t.Fatalf("AllocatePort(0): %v", err)
	}
	defer listener.Close()
	defer mgr.ReleasePort(port)

	if port < 40200 || port > 40210 {
		t.Fatalf("auto-assigned port %d outside range [40200, 40210]", port)
	}
}

func TestTCPAllocateDuplicate(t *testing.T) {
	mgr, srv := newTestTCPManager(40300, 40310)
	defer srv.cancel()

	port, listener, err := mgr.AllocatePort(40305)
	if err != nil {
		t.Fatalf("first AllocatePort: %v", err)
	}
	defer listener.Close()
	defer mgr.ReleasePort(port)

	_, _, err = mgr.AllocatePort(40305)
	if err == nil {
		t.Fatal("expected error for duplicate port allocation")
	}
}

func TestTCPReleasePort(t *testing.T) {
	mgr, srv := newTestTCPManager(40400, 40410)
	defer srv.cancel()

	port, listener, err := mgr.AllocatePort(40405)
	if err != nil {
		t.Fatalf("AllocatePort: %v", err)
	}
	listener.Close()
	mgr.ReleasePort(port)

	// Should be able to allocate the same port again
	port2, listener2, err := mgr.AllocatePort(40405)
	if err != nil {
		t.Fatalf("re-AllocatePort after release: %v", err)
	}
	defer listener2.Close()
	defer mgr.ReleasePort(port2)

	if port2 != 40405 {
		t.Fatalf("expected port 40405, got %d", port2)
	}
}

func TestTCPAllocateOutOfRange(t *testing.T) {
	mgr, srv := newTestTCPManager(40500, 40510)
	defer srv.cancel()

	_, _, err := mgr.AllocatePort(99999)
	if err == nil {
		t.Fatal("expected error for out-of-range port")
	}
}
