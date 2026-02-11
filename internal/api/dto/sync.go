package dto

import (
	"time"

	"github.com/mephistofox/fxtun.dev/internal/database"
)

// SyncRequest represents a sync request from client
type SyncRequest struct {
	Bundles  []BundleSyncItem   `json:"bundles,omitempty"`
	History  []HistorySyncItem  `json:"history,omitempty"`
	Settings []SettingSyncItem  `json:"settings,omitempty"`
}

// BundleSyncItem represents a bundle for sync
type BundleSyncItem struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	LocalPort   int       `json:"local_port"`
	Subdomain   string    `json:"subdomain,omitempty"`
	RemotePort  int       `json:"remote_port,omitempty"`
	AutoConnect bool      `json:"auto_connect"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Deleted     bool      `json:"deleted,omitempty"`
}

// HistorySyncItem represents a history entry for sync
type HistorySyncItem struct {
	BundleName     string     `json:"bundle_name,omitempty"`
	TunnelType     string     `json:"tunnel_type"`
	LocalPort      int        `json:"local_port"`
	RemoteAddr     string     `json:"remote_addr,omitempty"`
	URL            string     `json:"url,omitempty"`
	ConnectedAt    time.Time  `json:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at,omitempty"`
	BytesSent      int64      `json:"bytes_sent"`
	BytesReceived  int64      `json:"bytes_received"`
}

// SettingSyncItem represents a setting for sync
type SettingSyncItem struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SyncResponse represents sync response to client
type SyncResponse struct {
	Bundles  []BundleDTO  `json:"bundles"`
	History  []HistoryDTO `json:"history"`
	Settings []SettingDTO `json:"settings"`
}

// BundleDTO represents a bundle in API responses
type BundleDTO struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	LocalPort   int       `json:"local_port"`
	Subdomain   string    `json:"subdomain,omitempty"`
	RemotePort  int       `json:"remote_port,omitempty"`
	AutoConnect bool      `json:"auto_connect"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// HistoryDTO represents a history entry in API responses
type HistoryDTO struct {
	ID             int64      `json:"id"`
	BundleName     string     `json:"bundle_name,omitempty"`
	TunnelType     string     `json:"tunnel_type"`
	LocalPort      int        `json:"local_port"`
	RemoteAddr     string     `json:"remote_addr,omitempty"`
	URL            string     `json:"url,omitempty"`
	ConnectedAt    time.Time  `json:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at,omitempty"`
	BytesSent      int64      `json:"bytes_sent"`
	BytesReceived  int64      `json:"bytes_received"`
}

// SettingDTO represents a setting in API responses
type SettingDTO struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BundleDTOFromModel converts database model to DTO
func BundleDTOFromModel(bundle *database.UserBundle) BundleDTO {
	return BundleDTO{
		ID:          bundle.ID,
		Name:        bundle.Name,
		Type:        bundle.Type,
		LocalPort:   bundle.LocalPort,
		Subdomain:   bundle.Subdomain,
		RemotePort:  bundle.RemotePort,
		AutoConnect: bundle.AutoConnect,
		CreatedAt:   bundle.CreatedAt,
		UpdatedAt:   bundle.UpdatedAt,
	}
}

// HistoryDTOFromModel converts database model to DTO
func HistoryDTOFromModel(entry *database.UserHistoryEntry) HistoryDTO {
	return HistoryDTO{
		ID:             entry.ID,
		BundleName:     entry.BundleName,
		TunnelType:     entry.TunnelType,
		LocalPort:      entry.LocalPort,
		RemoteAddr:     entry.RemoteAddr,
		URL:            entry.URL,
		ConnectedAt:    entry.ConnectedAt,
		DisconnectedAt: entry.DisconnectedAt,
		BytesSent:      entry.BytesSent,
		BytesReceived:  entry.BytesReceived,
	}
}

// SettingDTOFromModel converts database model to DTO
func SettingDTOFromModel(setting *database.UserSetting) SettingDTO {
	return SettingDTO{
		Key:       setting.Key,
		Value:     setting.Value,
		UpdatedAt: setting.UpdatedAt,
	}
}

// ToUserBundle converts sync item to database model
func (b *BundleSyncItem) ToUserBundle(userID int64) *database.UserBundle {
	return &database.UserBundle{
		UserID:      userID,
		Name:        b.Name,
		Type:        b.Type,
		LocalPort:   b.LocalPort,
		Subdomain:   b.Subdomain,
		RemotePort:  b.RemotePort,
		AutoConnect: b.AutoConnect,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
}

// ToUserHistoryEntry converts sync item to database model
func (h *HistorySyncItem) ToUserHistoryEntry(userID int64) *database.UserHistoryEntry {
	return &database.UserHistoryEntry{
		UserID:         userID,
		BundleName:     h.BundleName,
		TunnelType:     h.TunnelType,
		LocalPort:      h.LocalPort,
		RemoteAddr:     h.RemoteAddr,
		URL:            h.URL,
		ConnectedAt:    h.ConnectedAt,
		DisconnectedAt: h.DisconnectedAt,
		BytesSent:      h.BytesSent,
		BytesReceived:  h.BytesReceived,
	}
}

// ToUserSetting converts sync item to database model
func (s *SettingSyncItem) ToUserSetting(userID int64) *database.UserSetting {
	return &database.UserSetting{
		UserID:    userID,
		Key:       s.Key,
		Value:     s.Value,
		UpdatedAt: s.UpdatedAt,
	}
}

// SyncBundlesRequest represents a request to sync only bundles
type SyncBundlesRequest struct {
	Bundles []BundleSyncItem `json:"bundles"`
}

// SyncSettingsRequest represents a request to sync only settings
type SyncSettingsRequest struct {
	Settings []SettingSyncItem `json:"settings"`
}

// SyncHistoryRequest represents a request to add history entries
type SyncHistoryRequest struct {
	History []HistorySyncItem `json:"history"`
}

// HistoryStatsDTO represents history statistics
type HistoryStatsDTO struct {
	TotalConnections   int   `json:"total_connections"`
	TotalBytesSent     int64 `json:"total_bytes_sent"`
	TotalBytesReceived int64 `json:"total_bytes_received"`
}
