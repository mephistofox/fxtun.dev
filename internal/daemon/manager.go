package daemon

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mephistofox/fxtun.dev/internal/client"
	"github.com/mephistofox/fxtun.dev/internal/config"
)

// ClientManager adapts a client.Client to the TunnelManager interface.
type ClientManager struct {
	client  *client.Client
	sigChan chan os.Signal
}

// NewClientManager creates a new ClientManager and sets up signal handling.
func NewClientManager(c *client.Client) *ClientManager {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	return &ClientManager{
		client:  c,
		sigChan: sig,
	}
}

// GetTunnels converts active tunnels from the client into TunnelInfo slices.
func (m *ClientManager) GetTunnels() []TunnelInfo {
	active := m.client.GetTunnels()
	infos := make([]TunnelInfo, 0, len(active))
	for _, t := range active {
		infos = append(infos, tunnelInfoFrom(t))
	}
	return infos
}

// RequestTunnel requests a new tunnel and returns its info.
func (m *ClientManager) RequestTunnel(cfg config.TunnelConfig) (TunnelInfo, error) {
	if cfg.Name == "" {
		cfg.Name = fmt.Sprintf("%s-%d", cfg.Type, cfg.LocalPort)
	}

	if err := m.client.RequestTunnel(cfg); err != nil {
		return TunnelInfo{}, err
	}

	// Find the newly created tunnel by matching type and port.
	for _, t := range m.client.GetTunnels() {
		if t.Config.Type == cfg.Type && t.Config.LocalPort == cfg.LocalPort {
			return tunnelInfoFrom(t), nil
		}
	}

	return TunnelInfo{}, fmt.Errorf("tunnel created but not found in active list")
}

// CloseTunnel closes a tunnel by ID.
func (m *ClientManager) CloseTunnel(id string) error {
	return m.client.CloseTunnel(id)
}

// Shutdown closes the client and sends SIGTERM to the signal channel.
func (m *ClientManager) Shutdown() {
	m.client.Close()
	m.sigChan <- syscall.SIGTERM
}

// SigChan returns the signal channel for the main loop.
func (m *ClientManager) SigChan() <-chan os.Signal {
	return m.sigChan
}

func tunnelInfoFrom(t *client.ActiveTunnel) TunnelInfo {
	return TunnelInfo{
		ID:         t.ID,
		Type:       t.Config.Type,
		LocalPort:  t.Config.LocalPort,
		RemotePort: t.Config.RemotePort,
		Subdomain:  t.Config.Subdomain,
		URL:        t.URL,
		RemoteAddr: t.RemoteAddr,
	}
}
