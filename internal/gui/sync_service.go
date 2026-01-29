package gui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/storage"
)

// SyncService handles data synchronization with the server
type SyncService struct {
	app *App
	log zerolog.Logger

	mu         sync.Mutex
	isSyncing  bool
	lastSynced *time.Time
	lastError  error
}

// NewSyncService creates a new sync service
func NewSyncService(app *App) *SyncService {
	return &SyncService{
		app: app,
		log: app.log.With().Str("service", "sync").Logger(),
	}
}

// SyncStatus represents the current sync status
type SyncStatus struct {
	IsSyncing  bool       `json:"is_syncing"`
	LastSynced *time.Time `json:"last_synced,omitempty"`
	LastError  string     `json:"last_error,omitempty"`
}

// GetStatus returns the current sync status
func (s *SyncService) GetStatus() *SyncStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := &SyncStatus{
		IsSyncing:  s.isSyncing,
		LastSynced: s.lastSynced,
	}
	if s.lastError != nil {
		status.LastError = s.lastError.Error()
	}
	return status
}

// getAPIHost returns the hostname for API calls
func (s *SyncService) getAPIHost() string {
	addr := s.app.serverAddress
	if idx := strings.Index(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// isConnected checks if the client is connected and authenticated
func (s *SyncService) isConnected() bool {
	return s.app.client != nil && s.app.authToken != ""
}

// SyncData represents all synced data
type SyncData struct {
	Bundles  []*storage.Bundle       `json:"bundles"`
	History  []*storage.HistoryEntry `json:"history"`
	Settings map[string]string       `json:"settings"`
}

// Pull downloads all data from the server
func (s *SyncService) Pull() (*SyncData, error) {
	if !s.isConnected() {
		return nil, fmt.Errorf("not connected")
	}

	s.mu.Lock()
	s.isSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isSyncing = false
		s.mu.Unlock()
	}()

	host := s.getAPIHost()
	url := fmt.Sprintf("https://%s/api/sync", host)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.app.authToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		err := fmt.Errorf("%s", errResp.Error)
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()
		return nil, err
	}

	var serverData struct {
		Bundles  []BundleSync  `json:"bundles"`
		History  []HistorySync `json:"history"`
		Settings []SettingSync `json:"settings"`
	}
	if err := json.Unmarshal(body, &serverData); err != nil {
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()
		return nil, err
	}

	// Convert to local storage format
	result := &SyncData{
		Bundles:  make([]*storage.Bundle, len(serverData.Bundles)),
		History:  make([]*storage.HistoryEntry, len(serverData.History)),
		Settings: make(map[string]string),
	}

	for i, b := range serverData.Bundles {
		result.Bundles[i] = &storage.Bundle{
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

	for i, h := range serverData.History {
		result.History[i] = &storage.HistoryEntry{
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

	for _, st := range serverData.Settings {
		result.Settings[st.Key] = st.Value
	}

	now := time.Now()
	s.mu.Lock()
	s.lastSynced = &now
	s.lastError = nil
	s.mu.Unlock()

	s.log.Info().
		Int("bundles", len(result.Bundles)).
		Int("history", len(result.History)).
		Int("settings", len(result.Settings)).
		Msg("Data pulled from server")

	return result, nil
}

// Push uploads all local data to the server
func (s *SyncService) Push() error {
	if !s.isConnected() {
		return fmt.Errorf("not connected")
	}

	s.mu.Lock()
	s.isSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isSyncing = false
		s.mu.Unlock()
	}()

	// Get local data
	bundleRepo := storage.NewBundleRepository(s.app.db)
	historyRepo := storage.NewHistoryRepository(s.app.db)
	settingsRepo := storage.NewSettingsRepository(s.app.db)

	bundles, _ := bundleRepo.List()
	history, _ := historyRepo.GetRecent(100)
	settings, _ := settingsRepo.GetAll()

	// Convert to sync format
	syncBundles := make([]BundleSync, len(bundles))
	for i, b := range bundles {
		syncBundles[i] = BundleSync{
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

	syncHistory := make([]HistorySync, len(history))
	for i, h := range history {
		syncHistory[i] = HistorySync{
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

	syncSettings := make([]SettingSync, 0, len(settings))
	for key, value := range settings {
		syncSettings = append(syncSettings, SettingSync{
			Key:       key,
			Value:     value,
			UpdatedAt: time.Now(),
		})
	}

	reqBody := map[string]interface{}{
		"bundles":  syncBundles,
		"history":  syncHistory,
		"settings": syncSettings,
	}

	jsonBody, _ := json.Marshal(reqBody)

	host := s.getAPIHost()
	url := fmt.Sprintf("https://%s/api/sync", host)

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.app.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		err := fmt.Errorf("%s", errResp.Error)
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()
		return err
	}

	now := time.Now()
	s.mu.Lock()
	s.lastSynced = &now
	s.lastError = nil
	s.mu.Unlock()

	s.log.Info().Msg("Data pushed to server")
	return nil
}

// SyncBundles synchronizes only bundles
func (s *SyncService) SyncBundles() error {
	if !s.isConnected() {
		return nil // Silent fail if not connected
	}

	bundleRepo := storage.NewBundleRepository(s.app.db)
	bundles, err := bundleRepo.List()
	if err != nil {
		return err
	}

	syncBundles := make([]BundleSync, len(bundles))
	for i, b := range bundles {
		syncBundles[i] = BundleSync{
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

	jsonBody, _ := json.Marshal(map[string]interface{}{
		"bundles": syncBundles,
	})

	host := s.getAPIHost()
	url := fmt.Sprintf("https://%s/api/sync/bundles", host)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.app.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.log.Debug().Err(err).Msg("Failed to sync bundles")
		return nil // Silent fail
	}
	defer resp.Body.Close()

	s.log.Debug().Int("status", resp.StatusCode).Msg("Bundles synced")
	return nil
}

// SyncSettings synchronizes only settings
func (s *SyncService) SyncSettings() error {
	if !s.isConnected() {
		return nil // Silent fail if not connected
	}

	settingsRepo := storage.NewSettingsRepository(s.app.db)
	settings, _ := settingsRepo.GetAll()

	syncSettings := make([]SettingSync, 0, len(settings))
	for key, value := range settings {
		syncSettings = append(syncSettings, SettingSync{
			Key:       key,
			Value:     value,
			UpdatedAt: time.Now(),
		})
	}

	jsonBody, _ := json.Marshal(map[string]interface{}{
		"settings": syncSettings,
	})

	host := s.getAPIHost()
	url := fmt.Sprintf("https://%s/api/sync/settings", host)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.app.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.log.Debug().Err(err).Msg("Failed to sync settings")
		return nil // Silent fail
	}
	defer resp.Body.Close()

	s.log.Debug().Int("status", resp.StatusCode).Msg("Settings synced")
	return nil
}

// PushHistoryEntry pushes a single history entry
func (s *SyncService) PushHistoryEntry(entry *storage.HistoryEntry) error {
	if !s.isConnected() {
		return nil // Silent fail if not connected
	}

	syncEntry := HistorySync{
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

	jsonBody, _ := json.Marshal(map[string]interface{}{
		"history": []HistorySync{syncEntry},
	})

	host := s.getAPIHost()
	url := fmt.Sprintf("https://%s/api/sync/history", host)

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.app.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.log.Debug().Err(err).Msg("Failed to push history entry")
		return nil // Silent fail
	}
	defer resp.Body.Close()

	s.log.Debug().Int("status", resp.StatusCode).Msg("History entry pushed")
	return nil
}

// ClearHistory clears history on the server
func (s *SyncService) ClearHistory() error {
	if !s.isConnected() {
		return nil // Silent fail if not connected
	}

	host := s.getAPIHost()
	url := fmt.Sprintf("https://%s/api/sync/history", host)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.app.authToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.log.Debug().Err(err).Msg("Failed to clear history on server")
		return nil // Silent fail
	}
	defer resp.Body.Close()

	s.log.Debug().Int("status", resp.StatusCode).Msg("History cleared on server")
	return nil
}

// ApplyServerData applies server data to local storage
func (s *SyncService) ApplyServerData(data *SyncData) error {
	if data == nil {
		return nil
	}

	bundleRepo := storage.NewBundleRepository(s.app.db)
	settingsRepo := storage.NewSettingsRepository(s.app.db)

	// Apply bundles (merge by updated_at)
	for _, serverBundle := range data.Bundles {
		localBundle, err := bundleRepo.GetByName(serverBundle.Name)
		if err != nil || localBundle == nil {
			// Bundle doesn't exist locally, create it
			bundleRepo.Create(serverBundle)
		} else {
			// Bundle exists, update if server is newer
			if serverBundle.UpdatedAt.After(localBundle.UpdatedAt) {
				serverBundle.ID = localBundle.ID
				bundleRepo.Update(serverBundle)
			}
		}
	}

	// Apply settings
	for key, value := range data.Settings {
		settingsRepo.Set(key, value)
	}

	s.log.Info().
		Int("bundles", len(data.Bundles)).
		Int("settings", len(data.Settings)).
		Msg("Server data applied to local storage")

	return nil
}

// Sync types for JSON serialization
type BundleSync struct {
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

type HistorySync struct {
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

type SettingSync struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
