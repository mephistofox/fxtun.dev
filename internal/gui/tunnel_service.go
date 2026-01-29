package gui

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
)

// TunnelService handles tunnel operations
type TunnelService struct {
	app *App
	log zerolog.Logger
}

// NewTunnelService creates a new tunnel service
func NewTunnelService(app *App) *TunnelService {
	return &TunnelService{
		app: app,
		log: app.log.With().Str("service", "tunnel").Logger(),
	}
}

// TunnelInfo represents tunnel information for the frontend
type TunnelInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	LocalPort     int    `json:"local_port"`
	RemoteAddr    string `json:"remote_addr,omitempty"`
	URL           string `json:"url,omitempty"`
	Connected     string `json:"connected"`
	BytesSent     int64  `json:"bytes_sent"`
	BytesReceived int64  `json:"bytes_received"`
}

// TunnelConfig represents tunnel configuration from the frontend
type TunnelConfig struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	LocalPort  int    `json:"local_port"`
	Subdomain  string `json:"subdomain,omitempty"`
	RemotePort int    `json:"remote_port,omitempty"`
}

// GetActiveTunnels returns all active tunnels
func (s *TunnelService) GetActiveTunnels() []TunnelInfo {
	if s.app.client == nil {
		return []TunnelInfo{}
	}

	tunnels := s.app.client.GetTunnels()
	result := make([]TunnelInfo, len(tunnels))

	for i, t := range tunnels {
		result[i] = TunnelInfo{
			ID:            t.ID,
			Name:          t.Config.Name,
			Type:          t.Config.Type,
			LocalPort:     t.Config.LocalPort,
			RemoteAddr:    t.RemoteAddr,
			URL:           t.URL,
			Connected:     t.Connected.Format(time.RFC3339),
			BytesSent:     t.BytesSent.Load(),
			BytesReceived: t.BytesReceived.Load(),
		}
	}

	return result
}

// CreateTunnel creates a new tunnel
func (s *TunnelService) CreateTunnel(cfg TunnelConfig) (*TunnelInfo, error) {
	if s.app.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	tunnelCfg := config.TunnelConfig{
		Name:       cfg.Name,
		Type:       cfg.Type,
		LocalPort:  cfg.LocalPort,
		Subdomain:  cfg.Subdomain,
		RemotePort: cfg.RemotePort,
	}

	// Try to create tunnel with auto-subdomain modification on conflict
	maxRetries := 3
	originalSubdomain := cfg.Subdomain

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := s.app.client.RequestTunnel(tunnelCfg); err != nil {
			// Check if it's a subdomain conflict error
			if cfg.Type == "http" && attempt < maxRetries && isSubdomainConflict(err) {
				// Modify subdomain with random suffix
				tunnelCfg.Subdomain = s.modifySubdomain(originalSubdomain)
				s.log.Info().
					Str("original", originalSubdomain).
					Str("modified", tunnelCfg.Subdomain).
					Msg("Subdomain taken, trying with modified name")
				continue
			}
			return nil, err
		}
		break
	}

	// Find the created tunnel
	tunnels := s.app.client.GetTunnels()
	for _, t := range tunnels {
		if t.Config.Name == cfg.Name {
			info := &TunnelInfo{
				ID:            t.ID,
				Name:          t.Config.Name,
				Type:          t.Config.Type,
				LocalPort:     t.Config.LocalPort,
				RemoteAddr:    t.RemoteAddr,
				URL:           t.URL,
				Connected:     t.Connected.Format(time.RFC3339),
				BytesSent:     t.BytesSent.Load(),
				BytesReceived: t.BytesReceived.Load(),
			}

			// Record connection in history and track for disconnect
			historyEntry, err := s.app.HistoryService.RecordConnect(cfg.Name, cfg.Type, cfg.LocalPort, t.RemoteAddr, t.URL)
			if err == nil && historyEntry != nil {
				s.app.TrackTunnelHistory(t.ID, historyEntry.ID)
			}

			return info, nil
		}
	}

	return nil, fmt.Errorf("tunnel created but not found")
}

// CloseTunnel closes a specific tunnel
func (s *TunnelService) CloseTunnel(tunnelID string) error {
	if s.app.client == nil {
		return fmt.Errorf("not connected")
	}

	if err := s.app.client.CloseTunnel(tunnelID); err != nil {
		s.log.Error().Err(err).Str("tunnel_id", tunnelID).Msg("Failed to close tunnel")
		return err
	}

	s.log.Info().Str("tunnel_id", tunnelID).Msg("Tunnel close requested")
	return nil
}

// GetConnectionStatus returns the current connection status
func (s *TunnelService) GetConnectionStatus() string {
	if s.app.client == nil {
		return "disconnected"
	}
	// The client could expose a status method
	return "connected"
}

// Disconnect disconnects from the server
func (s *TunnelService) Disconnect() error {
	if s.app.client == nil {
		return nil
	}

	// Record disconnect for all active tunnels
	for _, t := range s.app.client.GetTunnels() {
		s.app.recordTunnelDisconnect(t.ID, t.BytesSent.Load(), t.BytesReceived.Load())
	}

	s.app.client.Close()
	s.app.client = nil
	return nil
}

// modifySubdomain adds a random suffix to the subdomain
func (s *TunnelService) modifySubdomain(subdomain string) string {
	suffixes := []string{
		"fox", "oak", "sky", "red", "sun", "moon", "star", "wave", "wind", "leaf",
		"blue", "pine", "snow", "rain", "fire", "ice", "gold", "jade", "ruby", "onyx",
	}

	suffix := suffixes[rand.Intn(len(suffixes))]
	if subdomain == "" {
		return suffix
	}
	return fmt.Sprintf("%s-%s", subdomain, suffix)
}

// isSubdomainConflict checks if the error indicates a subdomain conflict
func isSubdomainConflict(err error) bool {
	errStr := err.Error()
	return contains(errStr, "subdomain") && (contains(errStr, "taken") || contains(errStr, "in use") || contains(errStr, "already"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
