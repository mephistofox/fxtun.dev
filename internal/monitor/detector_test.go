package monitor

import (
	"fmt"
	"testing"
	"time"
)

func TestDetector_ProxyPattern(t *testing.T) {
	cfg := DefaultDetectionConfig()
	cfg.UniqueIPsThreshold = 5
	cfg.ShortConnRatio = 0.7

	m := NewTunnelMetrics("t1", "tcp", TunnelLimits{TCPConnPerMin: 100000})

	for i := 0; i < 10; i++ {
		m.RecordConnection(fmt.Sprintf("1.2.3.%d:1000", i))
	}
	for i := 0; i < 8; i++ {
		m.RecordConnectionDone(1*time.Second, 100, 100)
	}
	for i := 0; i < 2; i++ {
		m.RecordConnectionDone(30*time.Second, 1000, 1000)
	}

	alerts := Detect(m, cfg)
	found := false
	for _, a := range alerts {
		if a.Type == AlertProxy {
			found = true
		}
	}
	if !found {
		t.Fatal("expected AlertProxy type")
	}
}

func TestDetector_UDPAmplification(t *testing.T) {
	cfg := DefaultDetectionConfig()
	cfg.UDPAmplificationFactor = 5

	m := NewTunnelMetrics("t1", "udp", TunnelLimits{UDPPacketsPerSec: 100000})
	m.bytesIn.Store(100)
	m.bytesOut.Store(1000)

	alerts := Detect(m, cfg)
	found := false
	for _, a := range alerts {
		if a.Type == AlertAmplification {
			found = true
		}
	}
	if !found {
		t.Fatal("expected amplification alert")
	}
}

func TestDetector_NoFalsePositive(t *testing.T) {
	cfg := DefaultDetectionConfig()
	cfg.UniqueIPsThreshold = 100
	m := NewTunnelMetrics("t1", "tcp", TunnelLimits{TCPConnPerMin: 100000})

	for i := 0; i < 5; i++ {
		m.RecordConnection(fmt.Sprintf("10.0.0.%d:1000", i))
		m.RecordConnectionDone(2*time.Minute, 50000, 50000)
	}

	alerts := Detect(m, cfg)
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts, got %d: %v", len(alerts), alerts)
	}
}

func TestDetector_RateLimitAlert(t *testing.T) {
	cfg := DefaultDetectionConfig()
	m := NewTunnelMetrics("t1", "tcp", TunnelLimits{TCPConnPerMin: 2})

	m.AllowConnection()
	m.AllowConnection()
	m.AllowConnection() // denied

	alerts := Detect(m, cfg)
	found := false
	for _, a := range alerts {
		if a.Type == AlertRateLimit {
			found = true
		}
	}
	if !found {
		t.Fatal("expected rate limit alert")
	}
}
