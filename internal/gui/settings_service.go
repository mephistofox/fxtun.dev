package gui

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/storage"
)

// SettingsService handles application settings
type SettingsService struct {
	app *App
	log zerolog.Logger
}

// NewSettingsService creates a new settings service
func NewSettingsService(app *App) *SettingsService {
	return &SettingsService{
		app: app,
		log: app.log.With().Str("service", "settings").Logger(),
	}
}

// Get retrieves a setting value
func (s *SettingsService) Get(key string) (string, error) {
	if s.app.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	repo := storage.NewSettingsRepository(s.app.db)
	return repo.Get(key)
}

// Set stores a setting value
func (s *SettingsService) Set(key, value string) error {
	if s.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	repo := storage.NewSettingsRepository(s.app.db)
	if err := repo.Set(key, value); err != nil {
		return err
	}

	s.log.Debug().Str("key", key).Msg("Setting saved")
	return nil
}

// GetAll returns all settings as a map
func (s *SettingsService) GetAll() (map[string]string, error) {
	if s.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	repo := storage.NewSettingsRepository(s.app.db)
	return repo.GetAll()
}

// GetBool retrieves a boolean setting
func (s *SettingsService) GetBool(key string, defaultValue bool) bool {
	if s.app.db == nil {
		return defaultValue
	}

	repo := storage.NewSettingsRepository(s.app.db)
	return repo.GetBool(key, defaultValue)
}

// SetBool stores a boolean setting
func (s *SettingsService) SetBool(key string, value bool) error {
	if s.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	repo := storage.NewSettingsRepository(s.app.db)
	return repo.SetBool(key, value)
}

// Settings keys
const (
	KeyTheme          = storage.SettingTheme
	KeyMinimizeToTray = storage.SettingMinimizeToTray
	KeyNotifications  = storage.SettingNotifications
	KeyServerAddress  = storage.SettingServerAddress
)

// GetTheme returns the current theme setting
func (s *SettingsService) GetTheme() string {
	theme, _ := s.Get(KeyTheme)
	if theme == "" {
		return "system"
	}
	return theme
}

// SetTheme sets the theme setting
func (s *SettingsService) SetTheme(theme string) error {
	return s.Set(KeyTheme, theme)
}

// GetMinimizeToTray returns the minimize to tray setting
func (s *SettingsService) GetMinimizeToTray() bool {
	return s.GetBool(KeyMinimizeToTray, true)
}

// SetMinimizeToTray sets the minimize to tray setting
func (s *SettingsService) SetMinimizeToTray(value bool) error {
	return s.SetBool(KeyMinimizeToTray, value)
}

// GetNotifications returns the notifications setting
func (s *SettingsService) GetNotifications() bool {
	return s.GetBool(KeyNotifications, true)
}

// SetNotifications sets the notifications setting
func (s *SettingsService) SetNotifications(value bool) error {
	return s.SetBool(KeyNotifications, value)
}

// GetDefaultServerAddress returns the default server address
func (s *SettingsService) GetDefaultServerAddress() string {
	addr, _ := s.Get(KeyServerAddress)
	return addr
}

// SetDefaultServerAddress sets the default server address
func (s *SettingsService) SetDefaultServerAddress(addr string) error {
	return s.Set(KeyServerAddress, addr)
}
