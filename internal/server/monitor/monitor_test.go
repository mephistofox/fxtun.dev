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

	// Use different IPs to avoid per-IP rate limiting
	for i := 0; i < 5; i++ {
		addr := fmt.Sprintf("10.0.0.%d:1000", i)
		if !mon.AllowTCPConnection("t1", addr) {
			t.Fatalf("connection %d should be allowed", i)
		}
	}
	if mon.AllowTCPConnection("t1", "10.0.0.100:1000") {
		t.Fatal("should be denied over limit")
	}
}

func TestMonitor_UnregisteredTunnelDefaultLimits(t *testing.T) {
	mon := New(DefaultConfig(), nil)
	defer mon.Stop()

	// Unknown tunnels now get default rate limits (fail-closed).
	// Default TCP limit is 1800/min, so first request should succeed.
	if !mon.AllowTCPConnection("unknown", "1.2.3.4:1000") {
		t.Fatal("unknown tunnel should be allowed with default limits")
	}
}

func TestMonitor_RemoveTunnel(t *testing.T) {
	cfg := DefaultConfig()
	mon := New(cfg, nil)
	defer mon.Stop()

	mon.RegisterTunnel("t1", "tcp", TunnelLimits{TCPConnPerMin: 2})
	mon.AllowTCPConnection("t1", "10.0.0.1:1")
	mon.AllowTCPConnection("t1", "10.0.0.2:1")
	if mon.AllowTCPConnection("t1", "10.0.0.3:1") {
		t.Fatal("should be denied")
	}

	mon.RemoveTunnel("t1")
	mon.RegisterTunnel("t1", "tcp", TunnelLimits{TCPConnPerMin: 2})
	if !mon.AllowTCPConnection("t1", "10.0.0.4:1") {
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
	// Use different IPs to avoid per-IP rate limiting
	for i := 0; i < 3; i++ {
		addr := fmt.Sprintf("10.0.0.%d:53", i)
		if !mon.AllowUDPPacket("u1", addr, 64) {
			t.Fatalf("packet %d should be allowed", i)
		}
	}
	if mon.AllowUDPPacket("u1", "10.0.0.100:53", 64) {
		t.Fatal("should be denied over limit")
	}
}

func TestMonitor_AllowHTTPRequest(t *testing.T) {
	mon := New(DefaultConfig(), nil)
	defer mon.Stop()

	mon.RegisterTunnel("h1", "http", TunnelLimits{HTTPReqPerMin: 3})
	// Use different IPs to avoid per-IP rate limiting
	for i := 0; i < 3; i++ {
		addr := fmt.Sprintf("10.0.0.%d:80", i)
		if !mon.AllowHTTPRequest("h1", addr) {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if mon.AllowHTTPRequest("h1", "10.0.0.100:80") {
		t.Fatal("should be denied over limit")
	}
}

func TestMonitor_PerIPRateLimiting(t *testing.T) {
	mon := New(DefaultConfig(), nil)
	defer mon.Stop()

	// Tunnel limit 100/min, per-IP should be 10/min (100/10)
	mon.RegisterTunnel("t1", "tcp", TunnelLimits{TCPConnPerMin: 100})

	// Single IP should hit per-IP limit before tunnel limit
	allowed := 0
	for i := 0; i < 20; i++ {
		if mon.AllowTCPConnection("t1", "10.0.0.1:1234") {
			allowed++
		}
	}
	if allowed >= 20 {
		t.Fatalf("per-IP limiting should cap single IP, got %d allowed", allowed)
	}
	if allowed != 10 {
		t.Fatalf("per-IP limit should be 10 (100/10), got %d allowed", allowed)
	}
}
