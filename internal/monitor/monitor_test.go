package monitor

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMonitor_AllowTCPConnection(t *testing.T) {
	cfg := DefaultConfig()
	mon := New(cfg, nil)
	defer mon.Stop()

	mon.RegisterTunnel("t1", "tcp", TunnelLimits{TCPConnPerMin: 5})

	for i := 0; i < 5; i++ {
		if !mon.AllowTCPConnection("t1", "1.2.3.4:1000") {
			t.Fatalf("connection %d should be allowed", i)
		}
	}
	if mon.AllowTCPConnection("t1", "1.2.3.4:1000") {
		t.Fatal("should be denied over limit")
	}
}

func TestMonitor_UnregisteredTunnelAllowed(t *testing.T) {
	mon := New(DefaultConfig(), nil)
	defer mon.Stop()

	if !mon.AllowTCPConnection("unknown", "1.2.3.4:1000") {
		t.Fatal("unknown tunnel should be allowed (fail-open)")
	}
}

func TestMonitor_RemoveTunnel(t *testing.T) {
	cfg := DefaultConfig()
	mon := New(cfg, nil)
	defer mon.Stop()

	mon.RegisterTunnel("t1", "tcp", TunnelLimits{TCPConnPerMin: 2})
	mon.AllowTCPConnection("t1", "x")
	mon.AllowTCPConnection("t1", "x")
	if mon.AllowTCPConnection("t1", "x") {
		t.Fatal("should be denied")
	}

	mon.RemoveTunnel("t1")
	mon.RegisterTunnel("t1", "tcp", TunnelLimits{TCPConnPerMin: 2})
	if !mon.AllowTCPConnection("t1", "x") {
		t.Fatal("should be allowed after re-register")
	}
}

func TestMonitor_DetectionRunsPeriodically(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DetectionInterval = 100 * time.Millisecond
	cfg.Detection.UniqueIPsThreshold = 3
	cfg.Detection.ShortConnRatio = 0.5

	var alerts []Alert
	var mu sync.Mutex
	mon := New(cfg, func(a Alert) {
		mu.Lock()
		alerts = append(alerts, a)
		mu.Unlock()
	})
	defer mon.Stop()

	mon.RegisterTunnel("t1", "tcp", TunnelLimits{TCPConnPerMin: 100000})
	for i := 0; i < 5; i++ {
		mon.AllowTCPConnection("t1", fmt.Sprintf("10.0.0.%d:1000", i))
	}
	for i := 0; i < 4; i++ {
		mon.RecordTCPConnectionDone("t1", 1*time.Second, 100, 100)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, a := range alerts {
		if a.Type == AlertProxy {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected proxy alert, got %d alerts: %v", len(alerts), alerts)
	}
}

func TestMonitor_AllowUDPPacket(t *testing.T) {
	mon := New(DefaultConfig(), nil)
	defer mon.Stop()

	mon.RegisterTunnel("u1", "udp", TunnelLimits{UDPPacketsPerSec: 3})
	for i := 0; i < 3; i++ {
		if !mon.AllowUDPPacket("u1", "1.2.3.4:53", 64) {
			t.Fatalf("packet %d should be allowed", i)
		}
	}
	if mon.AllowUDPPacket("u1", "1.2.3.4:53", 64) {
		t.Fatal("should be denied over limit")
	}
}

func TestMonitor_AllowHTTPRequest(t *testing.T) {
	mon := New(DefaultConfig(), nil)
	defer mon.Stop()

	mon.RegisterTunnel("h1", "http", TunnelLimits{HTTPReqPerMin: 3})
	for i := 0; i < 3; i++ {
		if !mon.AllowHTTPRequest("h1", "1.2.3.4:80") {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if mon.AllowHTTPRequest("h1", "1.2.3.4:80") {
		t.Fatal("should be denied over limit")
	}
}
