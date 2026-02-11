package server

import (
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/config"
)

// ViolationType identifies the kind of violation.
type ViolationType int

const (
	ViolationAuth  ViolationType = iota // failed authentication
	ViolationFlood                      // connection flood
)

type violation struct {
	events []time.Time
}

type banEntry struct {
	until      time.Time
	banCount   int // for exponential backoff
}

// IPBanManager tracks violations per IP and bans offenders automatically.
type IPBanManager struct {
	mu         sync.RWMutex
	bans       map[string]*banEntry
	violations map[string]map[ViolationType]*violation
	cfg        config.IPBanConfig
	log        zerolog.Logger
	stopCh     chan struct{}
}

// NewIPBanManager creates a new IP ban manager and starts the cleanup goroutine.
func NewIPBanManager(cfg config.IPBanConfig, log zerolog.Logger) *IPBanManager {
	m := &IPBanManager{
		bans:       make(map[string]*banEntry),
		violations: make(map[string]map[ViolationType]*violation),
		cfg:        cfg,
		log:        log.With().Str("component", "ipban").Logger(),
		stopCh:     make(chan struct{}),
	}
	go m.cleanupLoop()
	return m
}

// Stop stops the cleanup goroutine.
func (m *IPBanManager) Stop() {
	close(m.stopCh)
}

// IsBanned returns true if the IP is currently banned.
func (m *IPBanManager) IsBanned(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ban, ok := m.bans[ip]
	if !ok {
		return false
	}
	return time.Now().Before(ban.until)
}

// RecordViolation records a violation for an IP and may trigger a ban.
func (m *IPBanManager) RecordViolation(ip string, vtype ViolationType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Get or create violation map for this IP
	vmap, ok := m.violations[ip]
	if !ok {
		vmap = make(map[ViolationType]*violation)
		m.violations[ip] = vmap
	}

	v, ok := vmap[vtype]
	if !ok {
		v = &violation{}
		vmap[vtype] = v
	}

	v.events = append(v.events, now)

	// Check threshold
	threshold, window := m.thresholdFor(vtype)
	if threshold <= 0 {
		return
	}

	// Count events within window
	cutoff := now.Add(-window)
	count := 0
	for _, t := range v.events {
		if t.After(cutoff) {
			count++
		}
	}

	if count >= threshold {
		m.banLocked(ip)
		// Reset violation events after ban
		v.events = nil
	}
}

func (m *IPBanManager) thresholdFor(vtype ViolationType) (int, time.Duration) {
	switch vtype {
	case ViolationAuth:
		w := m.cfg.AuthWindow
		if w <= 0 {
			w = 5 * time.Minute
		}
		t := m.cfg.AuthThreshold
		if t <= 0 {
			t = 5
		}
		return t, w
	case ViolationFlood:
		w := m.cfg.FloodWindow
		if w <= 0 {
			w = 10 * time.Second
		}
		t := m.cfg.FloodThreshold
		if t <= 0 {
			t = 20
		}
		return t, w
	default:
		return 0, 0
	}
}

// banLocked applies a ban to the IP. Must be called with m.mu held.
func (m *IPBanManager) banLocked(ip string) {
	baseDur := m.cfg.BanDuration
	if baseDur <= 0 {
		baseDur = time.Hour
	}
	maxDur := m.cfg.MaxBanDuration
	if maxDur <= 0 {
		maxDur = 24 * time.Hour
	}

	ban, exists := m.bans[ip]
	if !exists {
		ban = &banEntry{}
		m.bans[ip] = ban
	}

	// Exponential backoff: baseDur * 2^banCount
	dur := baseDur
	for i := 0; i < ban.banCount; i++ {
		dur *= 2
		if dur > maxDur {
			dur = maxDur
			break
		}
	}

	ban.until = time.Now().Add(dur)
	ban.banCount++

	m.log.Warn().Str("ip", ip).Dur("duration", dur).Int("ban_count", ban.banCount).Msg("IP banned")
}

// cleanupLoop periodically removes expired bans and stale violations.
func (m *IPBanManager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

func (m *IPBanManager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Remove expired bans
	for ip, ban := range m.bans {
		if now.After(ban.until) {
			delete(m.bans, ip)
		}
	}

	// Remove stale violations (older than max window)
	maxWindow := 10 * time.Minute
	cutoff := now.Add(-maxWindow)
	for ip, vmap := range m.violations {
		empty := true
		for vtype, v := range vmap {
			// Compact events
			var kept []time.Time
			for _, t := range v.events {
				if t.After(cutoff) {
					kept = append(kept, t)
				}
			}
			v.events = kept
			if len(kept) == 0 {
				delete(vmap, vtype)
			} else {
				empty = false
			}
		}
		if empty {
			delete(m.violations, ip)
		}
	}
}
