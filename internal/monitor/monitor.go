package monitor

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// AlertFunc is called when suspicious activity is detected.
type AlertFunc func(Alert)

// Monitor tracks per-tunnel metrics and runs periodic detection.
type Monitor struct {
	cfg     Config
	tunnels sync.Map // tunnelID -> *TunnelMetrics
	alertFn AlertFunc
	log     zerolog.Logger
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// New creates a new Monitor. If alertFn is nil, alerts are only logged.
func New(cfg Config, alertFn AlertFunc) *Monitor {
	m := &Monitor{
		cfg:     cfg,
		alertFn: alertFn,
		log:     log.With().Str("component", "monitor").Logger(),
		stopCh:  make(chan struct{}),
	}
	if cfg.Enabled && cfg.DetectionInterval > 0 {
		m.wg.Add(1)
		go m.detectionLoop()
	}
	return m
}

// Stop shuts down the detection loop.
func (m *Monitor) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// RegisterTunnel registers a tunnel with plan-based rate limits.
func (m *Monitor) RegisterTunnel(tunnelID, tunnelType string, limits TunnelLimits) {
	metrics := NewTunnelMetrics(tunnelID, tunnelType, limits)
	m.tunnels.Store(tunnelID, metrics)
	m.log.Debug().Str("tunnel", tunnelID).Str("type", tunnelType).Msg("tunnel registered with monitor")
}

// RemoveTunnel removes a tunnel from monitoring.
func (m *Monitor) RemoveTunnel(tunnelID string) {
	m.tunnels.Delete(tunnelID)
}

func (m *Monitor) getMetrics(tunnelID string) *TunnelMetrics {
	v, ok := m.tunnels.Load(tunnelID)
	if !ok {
		return nil
	}
	return v.(*TunnelMetrics)
}

// AllowTCPConnection checks rate limit and records the connection.
// Returns true if the connection should proceed. Fail-open for unknown tunnels.
func (m *Monitor) AllowTCPConnection(tunnelID, remoteAddr string) bool {
	metrics := m.getMetrics(tunnelID)
	if metrics == nil {
		return true
	}
	if !metrics.AllowConnection() {
		m.log.Warn().Str("tunnel", tunnelID).Str("remote", remoteAddr).Msg("TCP connection rate limited")
		return false
	}
	metrics.RecordConnection(remoteAddr)
	return true
}

// AllowUDPPacket checks rate limit for a UDP packet.
func (m *Monitor) AllowUDPPacket(tunnelID, remoteAddr string, size int) bool {
	metrics := m.getMetrics(tunnelID)
	if metrics == nil {
		return true
	}
	if !metrics.AllowConnection() {
		return false
	}
	metrics.RecordConnection(remoteAddr)
	return true
}

// AllowHTTPRequest checks rate limit for an HTTP request.
func (m *Monitor) AllowHTTPRequest(tunnelID, remoteAddr string) bool {
	metrics := m.getMetrics(tunnelID)
	if metrics == nil {
		return true
	}
	if !metrics.AllowConnection() {
		m.log.Warn().Str("tunnel", tunnelID).Str("remote", remoteAddr).Msg("HTTP request rate limited")
		return false
	}
	metrics.RecordConnection(remoteAddr)
	return true
}

// RecordTCPConnectionDone records connection completion metrics.
func (m *Monitor) RecordTCPConnectionDone(tunnelID string, duration time.Duration, bytesIn, bytesOut int64) {
	metrics := m.getMetrics(tunnelID)
	if metrics == nil {
		return
	}
	metrics.RecordConnectionDone(duration, bytesIn, bytesOut)
}

// RecordUDPBytes records bytes transferred through a UDP tunnel.
func (m *Monitor) RecordUDPBytes(tunnelID string, bytesIn, bytesOut int64) {
	metrics := m.getMetrics(tunnelID)
	if metrics == nil {
		return
	}
	metrics.bytesIn.Add(bytesIn)
	metrics.bytesOut.Add(bytesOut)
}

func (m *Monitor) detectionLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.cfg.DetectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.runDetection()
		}
	}
}

func (m *Monitor) runDetection() {
	m.tunnels.Range(func(key, value any) bool {
		metrics := value.(*TunnelMetrics)
		alerts := Detect(metrics, m.cfg.Detection)
		for _, alert := range alerts {
			m.log.Warn().
				Str("tunnel", alert.TunnelID).
				Str("type", string(alert.Type)).
				Str("severity", string(alert.Severity)).
				Msg(alert.Message)
			if m.alertFn != nil {
				m.alertFn(alert)
			}
		}
		return true
	})
}
