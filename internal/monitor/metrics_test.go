package monitor

import (
	"testing"
	"time"
)

func TestTunnelMetrics_RecordConnection(t *testing.T) {
	m := NewTunnelMetrics("t1", "tcp", TunnelLimits{TCPConnPerMin: 1800})
	m.RecordConnection("1.2.3.4:1000")
	m.RecordConnection("5.6.7.8:2000")
	m.RecordConnection("1.2.3.4:3000")
	if m.UniqueIPCount() != 2 {
		t.Fatalf("expected 2 unique IPs, got %d", m.UniqueIPCount())
	}
	if m.TotalConnections() != 3 {
		t.Fatalf("expected 3 total, got %d", m.TotalConnections())
	}
}

func TestTunnelMetrics_RecordConnectionDone(t *testing.T) {
	m := NewTunnelMetrics("t1", "tcp", TunnelLimits{TCPConnPerMin: 1800})
	m.RecordConnectionDone(500*time.Millisecond, 100, 200)
	m.RecordConnectionDone(10*time.Second, 1000, 2000)
	if m.ShortConnections() != 1 {
		t.Fatalf("expected 1 short, got %d", m.ShortConnections())
	}
	if m.BytesIn() != 1100 || m.BytesOut() != 2200 {
		t.Fatalf("bytes mismatch: in=%d out=%d", m.BytesIn(), m.BytesOut())
	}
}

func TestTunnelMetrics_PlanLimitEnforced(t *testing.T) {
	m := NewTunnelMetrics("t1", "tcp", TunnelLimits{TCPConnPerMin: 3})
	for i := 0; i < 3; i++ {
		if !m.AllowConnection() {
			t.Fatalf("connection %d should be allowed", i)
		}
	}
	if m.AllowConnection() {
		t.Fatal("should be denied over plan limit")
	}
}

func TestTunnelMetrics_UnlimitedPlan(t *testing.T) {
	m := NewTunnelMetrics("t1", "tcp", TunnelLimits{TCPConnPerMin: -1})
	for i := 0; i < 10000; i++ {
		if !m.AllowConnection() {
			t.Fatalf("unlimited plan denied request %d", i)
		}
	}
}

func TestTunnelMetrics_DefaultLimit(t *testing.T) {
	m := NewTunnelMetrics("t1", "tcp", TunnelLimits{}) // all zeros = use defaults
	// Default TCP is 1800/min, so 1800 should be allowed
	for i := 0; i < 1800; i++ {
		if !m.AllowConnection() {
			t.Fatalf("request %d should be allowed within default limit", i)
		}
	}
	if m.AllowConnection() {
		t.Fatal("should be denied after default limit")
	}
}

func TestResolveLimit(t *testing.T) {
	if r := resolveLimit(-1, 1800); r != 0 {
		t.Fatalf("expected 0 (unlimited), got %d", r)
	}
	if r := resolveLimit(0, 1800); r != 1800 {
		t.Fatalf("expected 1800 (default), got %d", r)
	}
	if r := resolveLimit(500, 1800); r != 500 {
		t.Fatalf("expected 500 (custom), got %d", r)
	}
}
