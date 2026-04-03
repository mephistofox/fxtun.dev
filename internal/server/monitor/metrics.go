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

	// Per-source-IP rate limiting
	ipLimiters   map[string]*SlidingWindow
	ipLimitersMu sync.Mutex
	perIPLimit   int64
	perIPWindow  time.Duration

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

	// Per-IP limit: 1/10 of tunnel limit, minimum 1
	perIPLimit := limit / 10
	if perIPLimit < 1 && limit > 0 {
		perIPLimit = 1
	}

	return &TunnelMetrics{
		TunnelID:    tunnelID,
		TunnelType:  tunnelType,
		rateLimiter: NewSlidingWindow(limit, window),
		uniqueIPs:   make(map[string]struct{}),
		ipLimiters:  make(map[string]*SlidingWindow),
		perIPLimit:  perIPLimit,
		perIPWindow: window,
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

// AllowConnectionFromIP checks both tunnel-level and per-source-IP rate limits.
func (m *TunnelMetrics) AllowConnectionFromIP(remoteAddr string) bool {
	// Tunnel-level check
	if !m.rateLimiter.Allow() {
		m.denied.Add(1)
		return false
	}

	// Per-IP check (skip if unlimited)
	if m.perIPLimit <= 0 {
		return true
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	m.ipLimitersMu.Lock()
	limiter, ok := m.ipLimiters[host]
	if !ok {
		limiter = NewSlidingWindow(m.perIPLimit, m.perIPWindow)
		m.ipLimiters[host] = limiter
	}
	m.ipLimitersMu.Unlock()

	if !limiter.Allow() {
		m.denied.Add(1)
		return false
	}
	return true
}

// CleanupIPLimiters removes IP limiters with no active events.
func (m *TunnelMetrics) CleanupIPLimiters() {
	m.ipLimitersMu.Lock()
	defer m.ipLimitersMu.Unlock()
	for ip, lim := range m.ipLimiters {
		if lim.Count() == 0 {
			delete(m.ipLimiters, ip)
		}
	}
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
