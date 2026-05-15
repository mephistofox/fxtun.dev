package monitor

import "fmt"

// AlertType categorizes the type of suspicious activity detected.
type AlertType string

const (
	AlertProxy         AlertType = "proxy"
	AlertAmplification AlertType = "amplification"
	AlertRateLimit     AlertType = "rate_limit"
)

// AlertSeverity indicates the urgency of the alert.
type AlertSeverity string

const (
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// Alert represents a detected suspicious pattern for a tunnel.
type Alert struct {
	Type       AlertType
	Severity   AlertSeverity
	TunnelID   string
	TunnelType string
	Message    string
}

// DetectionConfig holds thresholds for heuristic detection.
type DetectionConfig struct {
	UniqueIPsThreshold     int
	ShortConnRatio         float64
	UDPAmplificationFactor int
}

// DefaultDetectionConfig returns default detection thresholds.
func DefaultDetectionConfig() DetectionConfig {
	return DetectionConfig{
		UniqueIPsThreshold:     200,
		ShortConnRatio:         0.8,
		UDPAmplificationFactor: 10,
	}
}

// Detect analyzes tunnel metrics and returns alerts for suspicious patterns.
func Detect(m *TunnelMetrics, cfg DetectionConfig) []Alert {
	var alerts []Alert

	// Proxy detection: many unique IPs + high ratio of short connections
	if m.TunnelType == "tcp" {
		uniqueIPs := m.UniqueIPCount()
		totalConns := m.TotalConnections()
		shortConns := m.ShortConnections()

		if uniqueIPs >= cfg.UniqueIPsThreshold && totalConns > 0 {
			shortRatio := float64(shortConns) / float64(totalConns)
			if shortRatio >= cfg.ShortConnRatio {
				alerts = append(alerts, Alert{
					Type:       AlertProxy,
					Severity:   SeverityCritical,
					TunnelID:   m.TunnelID,
					TunnelType: m.TunnelType,
					Message: fmt.Sprintf(
						"proxy pattern: %d unique IPs, %.0f%% short connections",
						uniqueIPs, shortRatio*100,
					),
				})
			}
		}
	}

	// UDP amplification detection
	if m.TunnelType == "udp" {
		in := m.BytesIn()
		out := m.BytesOut()
		if in > 0 && cfg.UDPAmplificationFactor > 0 {
			ratio := float64(out) / float64(in)
			if ratio >= float64(cfg.UDPAmplificationFactor) {
				alerts = append(alerts, Alert{
					Type:       AlertAmplification,
					Severity:   SeverityCritical,
					TunnelID:   m.TunnelID,
					TunnelType: m.TunnelType,
					Message: fmt.Sprintf(
						"UDP amplification: %.1fx ratio (in=%d, out=%d)",
						ratio, in, out,
					),
				})
			}
		}
	}

	// Rate limit exceeded (informational)
	if m.DeniedCount() > 0 {
		alerts = append(alerts, Alert{
			Type:       AlertRateLimit,
			Severity:   SeverityWarning,
			TunnelID:   m.TunnelID,
			TunnelType: m.TunnelType,
			Message: fmt.Sprintf(
				"rate limit hit: %d denied, current rate %d",
				m.DeniedCount(), m.CurrentRate(),
			),
		})
	}

	return alerts
}
