package server

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/protocol"
)

// ClientManager manages connected clients and user-client mappings.
type ClientManager struct {
	clients       map[string]*Client
	clientsMu     sync.RWMutex
	userClients   map[int64][]string // userID -> clientIDs
	userClientsMu sync.RWMutex
	log           zerolog.Logger
}

// NewClientManager creates a new ClientManager.
func NewClientManager(log zerolog.Logger) *ClientManager {
	return &ClientManager{
		clients:     make(map[string]*Client),
		userClients: make(map[int64][]string),
		log:         log,
	}
}

func (cm *ClientManager) addClient(clientID string, client *Client) {
	cm.clientsMu.Lock()
	cm.clients[clientID] = client
	cm.clientsMu.Unlock()
}

func (cm *ClientManager) removeClient(clientID string) {
	cm.clientsMu.Lock()
	delete(cm.clients, clientID)
	cm.clientsMu.Unlock()
}

// GetClient returns a client by ID.
func (cm *ClientManager) GetClient(clientID string) *Client {
	cm.clientsMu.RLock()
	defer cm.clientsMu.RUnlock()
	return cm.clients[clientID]
}

func (cm *ClientManager) allClients() []*Client {
	cm.clientsMu.Lock()
	clients := make([]*Client, 0, len(cm.clients))
	for _, c := range cm.clients {
		clients = append(clients, c)
	}
	cm.clientsMu.Unlock()
	return clients
}

// linkUserClient links a user ID to a client ID.
func (cm *ClientManager) linkUserClient(userID int64, clientID string) {
	if userID == 0 {
		return
	}

	cm.userClientsMu.Lock()
	defer cm.userClientsMu.Unlock()

	cm.userClients[userID] = append(cm.userClients[userID], clientID)
}

// unlinkUserClient removes a client ID from a user's client list.
func (cm *ClientManager) unlinkUserClient(userID int64, clientID string) {
	if userID == 0 {
		return
	}

	cm.userClientsMu.Lock()
	defer cm.userClientsMu.Unlock()

	clients := cm.userClients[userID]
	for i, id := range clients {
		if id == clientID {
			cm.userClients[userID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	if len(cm.userClients[userID]) == 0 {
		delete(cm.userClients, userID)
	}
}

// GetTunnelsByUserID returns all tunnels for a user.
func (cm *ClientManager) GetTunnelsByUserID(userID int64) []TunnelInfo {
	var tunnels []TunnelInfo

	cm.userClientsMu.RLock()
	clientIDs := cm.userClients[userID]
	cm.userClientsMu.RUnlock()

	cm.clientsMu.RLock()
	defer cm.clientsMu.RUnlock()

	for _, clientID := range clientIDs {
		client, ok := cm.clients[clientID]
		if !ok {
			continue
		}

		client.TunnelsMu.RLock()
		for _, tunnel := range client.Tunnels {
			tunnels = append(tunnels, TunnelInfo{
				ID:         tunnel.ID,
				Type:       string(tunnel.Type),
				Name:       tunnel.Name,
				Subdomain:  tunnel.Subdomain,
				RemotePort: tunnel.RemotePort,
				LocalPort:  tunnel.LocalPort,
				ClientID:   tunnel.ClientID,
				UserID:     client.UserID,
				CreatedAt:  tunnel.Created,
			})
		}
		client.TunnelsMu.RUnlock()
	}

	return tunnels
}

// GetAllTunnels returns all tunnels from all clients.
func (cm *ClientManager) GetAllTunnels() []TunnelInfo {
	var tunnels []TunnelInfo

	cm.clientsMu.RLock()
	defer cm.clientsMu.RUnlock()

	for _, client := range cm.clients {
		client.TunnelsMu.RLock()
		for _, tunnel := range client.Tunnels {
			tunnels = append(tunnels, TunnelInfo{
				ID:         tunnel.ID,
				Type:       string(tunnel.Type),
				Name:       tunnel.Name,
				Subdomain:  tunnel.Subdomain,
				RemotePort: tunnel.RemotePort,
				LocalPort:  tunnel.LocalPort,
				ClientID:   tunnel.ClientID,
				UserID:     client.UserID,
				CreatedAt:  tunnel.Created,
			})
		}
		client.TunnelsMu.RUnlock()
	}

	return tunnels
}

// AdminCloseTunnel closes any tunnel by ID (admin only).
func (cm *ClientManager) AdminCloseTunnel(tunnelID string) error {
	cm.clientsMu.RLock()
	defer cm.clientsMu.RUnlock()

	for _, client := range cm.clients {
		client.TunnelsMu.RLock()
		_, exists := client.Tunnels[tunnelID]
		client.TunnelsMu.RUnlock()

		if exists {
			client.closeTunnel(tunnelID)
			return nil
		}
	}

	return fmt.Errorf("tunnel not found")
}

// CloseTunnelByID closes a tunnel by ID for a specific user.
func (cm *ClientManager) CloseTunnelByID(tunnelID string, userID int64) error {
	cm.userClientsMu.RLock()
	clientIDs := cm.userClients[userID]
	cm.userClientsMu.RUnlock()

	cm.clientsMu.RLock()
	defer cm.clientsMu.RUnlock()

	for _, clientID := range clientIDs {
		client, ok := cm.clients[clientID]
		if !ok {
			continue
		}

		client.TunnelsMu.RLock()
		_, exists := client.Tunnels[tunnelID]
		client.TunnelsMu.RUnlock()

		if exists {
			client.closeTunnel(tunnelID)
			return nil
		}
	}

	return fmt.Errorf("tunnel not found")
}

// GetStats returns server statistics.
func (cm *ClientManager) GetStats() Stats {
	cm.clientsMu.RLock()
	defer cm.clientsMu.RUnlock()

	stats := Stats{
		ActiveClients: len(cm.clients),
	}

	for _, client := range cm.clients {
		client.TunnelsMu.RLock()
		for _, tunnel := range client.Tunnels {
			stats.ActiveTunnels++
			switch tunnel.Type {
			case protocol.TunnelHTTP:
				stats.HTTPTunnels++
			case protocol.TunnelTCP:
				stats.TCPTunnels++
			case protocol.TunnelUDP:
				stats.UDPTunnels++
			}
		}
		client.TunnelsMu.RUnlock()
	}

	return stats
}

