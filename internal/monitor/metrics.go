package monitor

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const shortConnThreshold = 5 * time.Second

// TunnelLimits holds per-tunnel rate limits from the user's Plan.
// Values: >0 = limit, 0 = use global default, -1 = unlimited.
type TunnelLimits struct {
	TCPConnPerMin    int
	UDPPacketsPerSec int
	HTTPReqPerMin    int
}

// Default limits when plan specifies 0.
const (
	DefaultTCPConnPerMin    = 1800
	DefaultUDPPacketsPerSec = 10000
	DefaultHTTPReqPerMin    = 3600
)

type TunnelMetrics struct {
	TunnelID   string
	TunnelType string

	rateLimiter *SlidingWindow

	uniqueIPs map[string]struct{}
	ipMu      sync.Mutex

	totalConns atomic.Int64
	shortConns atomic.Int64
	bytesIn    atomic.Int64
	bytesOut   atomic.Int64
	denied     atomic.Int64
}

func NewTunnelMetrics(tunnelID, tunnelType string, limits TunnelLimits) *TunnelMetrics {
	var limit int64
	var window time.Duration

	switch tunnelType {
	case "tcp":
		limit = resolveLimit(int64(limits.TCPConnPerMin), DefaultTCPConnPerMin)
		window = time.Minute
	case "udp":
		limit = resolveLimit(int64(limits.UDPPacketsPerSec), DefaultUDPPacketsPerSec)
		window = time.Second
	default:
		limit = resolveLimit(int64(limits.HTTPReqPerMin), DefaultHTTPReqPerMin)
		window = time.Minute
	}

	return &TunnelMetrics{
		TunnelID:    tunnelID,
		TunnelType:  tunnelType,
		rateLimiter: NewSlidingWindow(limit, window),
		uniqueIPs:   make(map[string]struct{}),
	}
}

// resolveLimit: <0 -> 0 (unlimited in SlidingWindow), 0 -> defaultVal, >0 -> value as-is.
func resolveLimit(planValue, defaultVal int64) int64 {
	if planValue < 0 {
		return 0
	}
	if planValue == 0 {
		return defaultVal
	}
	return planValue
}

func (m *TunnelMetrics) AllowConnection() bool {
	if !m.rateLimiter.Allow() {
		m.denied.Add(1)
		return false
	}
	return true
}

func (m *TunnelMetrics) RecordConnection(remoteAddr string) {
	m.totalConns.Add(1)
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	m.ipMu.Lock()
	m.uniqueIPs[host] = struct{}{}
	m.ipMu.Unlock()
}

func (m *TunnelMetrics) RecordConnectionDone(duration time.Duration, bytesIn, bytesOut int64) {
	if duration < shortConnThreshold {
		m.shortConns.Add(1)
	}
	m.bytesIn.Add(bytesIn)
	m.bytesOut.Add(bytesOut)
}

func (m *TunnelMetrics) UniqueIPCount() int {
	m.ipMu.Lock()
	defer m.ipMu.Unlock()
	return len(m.uniqueIPs)
}

func (m *TunnelMetrics) TotalConnections() int64 { return m.totalConns.Load() }
func (m *TunnelMetrics) ShortConnections() int64  { return m.shortConns.Load() }
func (m *TunnelMetrics) BytesIn() int64           { return m.bytesIn.Load() }
func (m *TunnelMetrics) BytesOut() int64          { return m.bytesOut.Load() }
func (m *TunnelMetrics) DeniedCount() int64       { return m.denied.Load() }
func (m *TunnelMetrics) CurrentRate() int64       { return m.rateLimiter.Count() }
