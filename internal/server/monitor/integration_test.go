package monitor

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestIntegration_PlanLimitsEnforced(t *testing.T) {
	cfg := DefaultConfig()
	mon := New(cfg, nil)
	defer mon.Stop()

	// Free plan: 300 TCP/min
	mon.RegisterTunnel("free-tunnel", "tcp", TunnelLimits{TCPConnPerMin: 300})
	// Pro plan: 1800 TCP/min
	mon.RegisterTunnel("pro-tunnel", "tcp", TunnelLimits{TCPConnPerMin: 1800})
	// Admin: unlimited
	mon.RegisterTunnel("admin-tunnel", "tcp", TunnelLimits{TCPConnPerMin: -1})

	// Free plan: 300 allowed, then denied (use different IPs to avoid per-IP limits)
	allowed := 0
	for i := 0; i < 400; i++ {
		addr := fmt.Sprintf("10.%d.%d.%d:1234", (i>>16)&0xFF, (i>>8)&0xFF, i&0xFF)
		if mon.AllowTCPConnection("free-tunnel", addr) {
			allowed++
		}
	}
	if allowed != 300 {
		t.Fatalf("free plan: expected 300 allowed, got %d", allowed)
	}

	// Pro plan: 1800 allowed (use different IPs)
	allowed = 0
	for i := 0; i < 2000; i++ {
		addr := fmt.Sprintf("10.%d.%d.%d:1234", (i>>16)&0xFF, (i>>8)&0xFF, i&0xFF)
		if mon.AllowTCPConnection("pro-tunnel", addr) {
			allowed++
		}
	}
	if allowed != 1800 {
		t.Fatalf("pro plan: expected 1800 allowed, got %d", allowed)
	}

	// Admin: all allowed (use same IP - unlimited means no per-IP either)
	allowed = 0
	for i := 0; i < 10000; i++ {
		if mon.AllowTCPConnection("admin-tunnel", "10.0.0.1:1234") {
			allowed++
		}
	}
	if allowed != 10000 {
		t.Fatalf("admin plan: expected 10000 allowed, got %d", allowed)
	}
}

func TestIntegration_ProxyDetectionTriggersAlert(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DetectionInterval = 100 * time.Millisecond
	cfg.Detection.UniqueIPsThreshold = 10
	cfg.Detection.ShortConnRatio = 0.6

	var alerts []Alert
	var mu sync.Mutex
	mon := New(cfg, func(a Alert) {
		mu.Lock()
		alerts = append(alerts, a)
		mu.Unlock()
	})
	defer mon.Stop()

	mon.RegisterTunnel("proxy-tunnel", "tcp", TunnelLimits{TCPConnPerMin: 100000})

	// Simulate proxy traffic: 20 unique IPs, short connections
	for i := 0; i < 20; i++ {
		addr := fmt.Sprintf("203.0.113.%d:9999", i)
		mon.AllowTCPConnection("proxy-tunnel", addr)
		mon.RecordTCPConnectionDone("proxy-tunnel", 2*time.Second, 500, 500)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	var proxyAlert *Alert
	for i := range alerts {
		if alerts[i].Type == AlertProxy {
			proxyAlert = &alerts[i]
			break
		}
	}
	if proxyAlert == nil {
		t.Fatalf("expected proxy alert, got %d alerts: %v", len(alerts), alerts)
	}
	if proxyAlert.Severity != SeverityCritical {
		t.Fatalf("expected critical severity, got %s", proxyAlert.Severity)
	}
}

func TestIntegration_UDPAmplificationDetected(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DetectionInterval = 100 * time.Millisecond
	cfg.Detection.UDPAmplificationFactor = 5

	var alerts []Alert
	var mu sync.Mutex
	mon := New(cfg, func(a Alert) {
		mu.Lock()
		alerts = append(alerts, a)
		mu.Unlock()
	})
	defer mon.Stop()

	mon.RegisterTunnel("dns-tunnel", "udp", TunnelLimits{UDPPacketsPerSec: 100000})

	// Simulate DNS amplification: small queries, large responses
	for i := 0; i < 50; i++ {
		addr := fmt.Sprintf("10.0.0.%d:53", i%250)
		mon.AllowUDPPacket("dns-tunnel", addr, 64)
		mon.RecordUDPBytes("dns-tunnel", 64, 0)
		mon.RecordUDPBytes("dns-tunnel", 0, 4096)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	found := false
	for _, a := range alerts {
		if a.Type == AlertAmplification {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected amplification alert, got %v", alerts)
	}
}

func TestIntegration_UnlimitedPlanNoRateLimit(t *testing.T) {
	cfg := DefaultConfig()
	mon := New(cfg, nil)
	defer mon.Stop()

	// Business/Admin plan: -1 = unlimited for all protocols
	mon.RegisterTunnel("biz-tcp", "tcp", TunnelLimits{TCPConnPerMin: -1})
	mon.RegisterTunnel("biz-udp", "udp", TunnelLimits{UDPPacketsPerSec: -1})
	mon.RegisterTunnel("biz-http", "http", TunnelLimits{HTTPReqPerMin: -1})

	for i := 0; i < 50000; i++ {
		if !mon.AllowTCPConnection("biz-tcp", "10.0.0.1:1") {
			t.Fatalf("TCP: unlimited plan denied at %d", i)
		}
	}
	for i := 0; i < 50000; i++ {
		if !mon.AllowUDPPacket("biz-udp", "10.0.0.1:1", 64) {
			t.Fatalf("UDP: unlimited plan denied at %d", i)
		}
	}
	for i := 0; i < 50000; i++ {
		if !mon.AllowHTTPRequest("biz-http", "10.0.0.1:1") {
			t.Fatalf("HTTP: unlimited plan denied at %d", i)
		}
	}
}

func TestIntegration_DefaultLimitsApplied(t *testing.T) {
	cfg := DefaultConfig()
	mon := New(cfg, nil)
	defer mon.Stop()

	// Plan with 0 = use defaults (TCP: 1800/min)
	mon.RegisterTunnel("default-tunnel", "tcp", TunnelLimits{})

	// Use different IPs to avoid per-IP limits
	allowed := 0
	for i := 0; i < 2000; i++ {
		addr := fmt.Sprintf("10.%d.%d.%d:1", (i>>16)&0xFF, (i>>8)&0xFF, i&0xFF)
		if mon.AllowTCPConnection("default-tunnel", addr) {
			allowed++
		}
	}
	if allowed != 1800 {
		t.Fatalf("expected default limit 1800, got %d allowed", allowed)
	}
}
