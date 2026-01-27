// Package gui provides the Wails backend for the GUI client.
package gui

import (
	"context"
	"encoding/json"

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
}

// NewApp creates a new App instance
func NewApp(log zerolog.Logger) *App {
	app := &App{
		log:     log.With().Str("component", "gui").Logger(),
		keyring: keyring.New(),
	}

	// Initialize services
	app.TunnelService = NewTunnelService(app)
	app.AuthService = NewAuthService(app)
	app.BundleService = NewBundleService(app)
	app.SettingsService = NewSettingsService(app)
	app.HistoryService = NewHistoryService(app)
	app.DomainService = NewDomainService(app)
	app.SyncService = NewSyncService(app)

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

	a.log.Info().Msg("GUI application started")
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	a.log.Info().Msg("GUI application shutting down")

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

// GetVersion returns the application version
func (a *App) GetVersion() string {
	return "1.0.0"
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
