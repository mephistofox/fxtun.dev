// Package gui provides the Wails backend for the GUI client.
package gui

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mephistofox/fxtunnel/internal/client"
	"github.com/mephistofox/fxtunnel/internal/keyring"
	"github.com/mephistofox/fxtunnel/internal/storage"
)

// App is the main application struct for Wails
type App struct {
	ctx context.Context
	log zerolog.Logger

	db      *storage.Database
	keyring *keyring.Keyring
	client  *client.Client

	// Centralized API client
	api *apiClient

	// Build info
	version   string
	buildTime string

	// Tray
	trayIcon []byte

	// History tracking: tunnelID → historyEntryID
	historyEntries   map[string]int64
	historyEntriesMu sync.RWMutex

	// Auth state
	serverAddress string
	authToken     string
	refreshToken  string

	// Services exposed to frontend
	TunnelService   *TunnelService
	AuthService     *AuthService
	BundleService   *BundleService
	SettingsService *SettingsService
	HistoryService  *HistoryService
	DomainService   *DomainService
	SyncService     *SyncService
	InspectService  *InspectService
}

// LogHook returns a zerolog Hook that forwards log events to the GUI frontend.
// Attach this to the root logger so all log messages appear in the Logs view.
func (a *App) LogHook() zerolog.Hook {
	return &logHook{app: a}
}

// NewApp creates a new App instance
func NewApp(log zerolog.Logger) *App {
	app := &App{
		log:            log.With().Str("component", "gui").Logger(),
		keyring:        keyring.New(),
		historyEntries: make(map[string]int64),
	}

	app.api = &apiClient{app: app, log: app.log.With().Str("component", "api-client").Logger()}

	// Initialize services
	app.TunnelService = NewTunnelService(app)
	app.AuthService = NewAuthService(app)
	app.BundleService = NewBundleService(app)
	app.SettingsService = NewSettingsService(app)
	app.HistoryService = NewHistoryService(app)
	app.DomainService = NewDomainService(app)
	app.SyncService = NewSyncService(app)
	app.InspectService = NewInspectService(app)

	return app
}

// Startup is called when the app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.log.Info().Msg("GUI application starting")

	// Initialize database
	db, err := storage.NewDefault()
	if err != nil {
		a.log.Error().Err(err).Msg("Failed to initialize database")
		return
	}
	a.db = db

	// Initialize system tray
	if len(a.trayIcon) > 0 {
		a.initTray(a.trayIcon)
	}

	a.log.Info().Msg("GUI application started")
}

// SetIcon sets the application icon for the system tray.
func (a *App) SetIcon(icon []byte) {
	a.trayIcon = icon
}

// UpdateLogger replaces the app's logger and propagates to all services.
func (a *App) UpdateLogger(log zerolog.Logger) {
	a.log = log.With().Str("component", "gui").Logger()
	a.TunnelService.log = a.log.With().Str("service", "tunnel").Logger()
	a.AuthService.log = a.log.With().Str("service", "auth").Logger()
	a.BundleService.log = a.log.With().Str("service", "bundle").Logger()
	a.SettingsService.log = a.log.With().Str("service", "settings").Logger()
	a.HistoryService.log = a.log.With().Str("service", "history").Logger()
	a.DomainService.log = a.log.With().Str("service", "domain").Logger()
	a.SyncService.log = a.log.With().Str("service", "sync").Logger()
	a.InspectService.log = a.log.With().Str("service", "inspect").Logger()
	a.api.log = a.log.With().Str("component", "api-client").Logger()
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	a.log.Info().Msg("GUI application shutting down")

	a.cleanupTray()

	if a.client != nil {
		a.client.Close()
	}

	if a.db != nil {
		a.db.Close()
	}
}

// emitEvent sends an event to the frontend
func (a *App) emitEvent(eventName string, data interface{}) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, eventName, data)
}

// subscribeToClientEvents subscribes to client events and forwards them to the frontend
func (a *App) subscribeToClientEvents() {
	if a.client == nil {
		return
	}

	a.client.Events().Subscribe(func(event client.Event) {
		// Handle tunnel closed: record disconnect with traffic stats
		if event.Type == client.EventTunnelClosed {
			if tunnelID, ok := event.Payload["tunnel_id"].(string); ok {
				var bytesSent, bytesReceived int64
				if v, ok := event.Payload["bytes_sent"]; ok {
					bytesSent, _ = v.(int64)
				}
				if v, ok := event.Payload["bytes_received"]; ok {
					bytesReceived, _ = v.(int64)
				}
				a.recordTunnelDisconnect(tunnelID, bytesSent, bytesReceived)
			}
		}

		// Convert to JSON-friendly format
		data := map[string]interface{}{
			"type": string(event.Type),
		}
		if event.Payload != nil {
			data["payload"] = event.Payload
		}

		// Emit to frontend
		a.emitEvent(string(event.Type), data)

		// Log event
		a.log.Debug().
			Str("event", string(event.Type)).
			Interface("payload", event.Payload).
			Msg("Client event")
	})
}

// TrackTunnelHistory associates a tunnel ID with a history entry ID.
func (a *App) TrackTunnelHistory(tunnelID string, historyEntryID int64) {
	a.historyEntriesMu.Lock()
	defer a.historyEntriesMu.Unlock()
	a.historyEntries[tunnelID] = historyEntryID
}

// recordTunnelDisconnect records disconnect stats for a tunnel.
func (a *App) recordTunnelDisconnect(tunnelID string, bytesSent, bytesReceived int64) {
	a.historyEntriesMu.Lock()
	entryID, ok := a.historyEntries[tunnelID]
	if ok {
		delete(a.historyEntries, tunnelID)
	}
	a.historyEntriesMu.Unlock()

	if !ok {
		return
	}

	if err := a.HistoryService.RecordDisconnect(entryID, bytesSent, bytesReceived); err != nil {
		a.log.Error().Err(err).Int64("entry_id", entryID).Msg("Failed to record disconnect")
	}
}

// SetBuildInfo sets version and build time from ldflags.
func (a *App) SetBuildInfo(version, buildTime string) {
	a.version = version
	a.buildTime = buildTime
}

// GetVersion returns the application version
func (a *App) GetVersion() string {
	if a.version != "" {
		return a.version
	}
	return "dev"
}

// GetBuildDate returns the build date string
func (a *App) GetBuildDate() string {
	if a.buildTime != "" {
		return a.buildTime
	}
	return "unknown"
}

// GetPlatformInfo returns platform information
func (a *App) GetPlatformInfo() map[string]string {
	env := runtime.Environment(a.ctx)
	return map[string]string{
		"platform":    env.Platform,
		"arch":        env.Arch,
		"buildType":   env.BuildType,
	}
}

// ExportData exports all user data as JSON
func (a *App) ExportData() (string, error) {
	bundles, err := a.BundleService.List()
	if err != nil {
		return "", err
	}

	settings, err := a.SettingsService.GetAll()
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"bundles":  bundles,
		"settings": settings,
		"version":  "1.0",
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// ImportData imports user data from JSON
func (a *App) ImportData(jsonData string) error {
	var data struct {
		Bundles []storage.Bundle  `json:"bundles"`
		Version string            `json:"version"`
	}

	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return err
	}

	// Import bundles
	for _, bundle := range data.Bundles {
		bundle.ID = 0 // Reset ID for new creation
		if _, err := a.BundleService.Create(&bundle); err != nil {
			a.log.Error().Err(err).Str("bundle", bundle.Name).Msg("Failed to import bundle")
		}
	}

	return nil
}
