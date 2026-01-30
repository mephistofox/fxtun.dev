package server

import (
	"net"
	"os"
	"testing"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
)

func newTestUDPManager(portMin, portMax int) (*UDPManager, *Server) {
	log := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	cfg := &config.ServerConfig{
		Server: config.ServerSettings{
			ControlPort:  14443,
			HTTPPort:     18080,
			TCPPortRange: config.PortRange{Min: 30000, Max: 31000},
			UDPPortRange: config.PortRange{Min: portMin, Max: portMax},
		},
		Domain: config.DomainSettings{Base: "test.local"},
	}
	srv := New(cfg, log)
	return srv.udpManager, srv
}

func TestUDPAllocateSpecificPort(t *testing.T) {
	mgr, srv := newTestUDPManager(41000, 41100)
	defer srv.cancel()

	port, conn, err := mgr.AllocatePort(41050)
	if err != nil {
		t.Fatalf("AllocatePort: %v", err)
	}
	defer conn.Close()
	defer mgr.ReleasePort(port)

	if port != 41050 {
		t.Fatalf("expected port 41050, got %d", port)
	}

	addr := conn.LocalAddr().(*net.UDPAddr)
	if addr.Port != 41050 {
		t.Fatalf("conn on wrong port: %d", addr.Port)
	}
}

func TestUDPAllocateAutoAssign(t *testing.T) {
	mgr, srv := newTestUDPManager(41200, 41210)
	defer srv.cancel()

	port, conn, err := mgr.AllocatePort(0)
	if err != nil {
		t.Fatalf("AllocatePort(0): %v", err)
	}
	defer conn.Close()
	defer mgr.ReleasePort(port)

	if port < 41200 || port > 41210 {
		t.Fatalf("auto-assigned port %d outside range [41200, 41210]", port)
	}
}

func TestUDPAllocateDuplicate(t *testing.T) {
	mgr, srv := newTestUDPManager(41300, 41310)
	defer srv.cancel()

	port, conn, err := mgr.AllocatePort(41305)
	if err != nil {
		t.Fatalf("first AllocatePort: %v", err)
	}
	defer conn.Close()
	defer mgr.ReleasePort(port)

	_, _, err = mgr.AllocatePort(41305)
	if err == nil {
		t.Fatal("expected error for duplicate port allocation")
	}
}

func TestUDPReleasePort(t *testing.T) {
	mgr, srv := newTestUDPManager(41400, 41410)
	defer srv.cancel()

	port, conn, err := mgr.AllocatePort(41405)
	if err != nil {
		t.Fatalf("AllocatePort: %v", err)
	}
	conn.Close()
	mgr.ReleasePort(port)

	port2, conn2, err := mgr.AllocatePort(41405)
	if err != nil {
		t.Fatalf("re-AllocatePort after release: %v", err)
	}
	defer conn2.Close()
	defer mgr.ReleasePort(port2)

	if port2 != 41405 {
		t.Fatalf("expected port 41405, got %d", port2)
	}
}

func TestUDPAllocateOutOfRange(t *testing.T) {
	mgr, srv := newTestUDPManager(41500, 41510)
	defer srv.cancel()

	_, _, err := mgr.AllocatePort(99999)
	if err == nil {
		t.Fatal("expected error for out-of-range port")
	}
}

func TestHashAddr(t *testing.T) {
	addr1 := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
	addr2 := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
	addr3 := &net.UDPAddr{IP: net.ParseIP("127.0.0.2"), Port: 1234}

	h1 := hashAddr(addr1)
	h2 := hashAddr(addr2)
	h3 := hashAddr(addr3)

	if h1 != h2 {
		t.Fatalf("same address should produce same hash: %d != %d", h1, h2)
	}
	if h1 == h3 {
		t.Fatal("different addresses should likely produce different hashes")
	}
}
