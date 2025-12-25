package storage

import (
	"database/sql"
	"fmt"
)

// SettingsRepository provides operations for application settings
type SettingsRepository struct {
	db *Database
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *Database) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves a setting value
func (r *SettingsRepository) Get(key string) (string, error) {
	var value string
	err := r.db.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("query setting: %w", err)
	}
	return value, nil
}

// Set stores a setting value
func (r *SettingsRepository) Set(key, value string) error {
	_, err := r.db.db.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("set setting: %w", err)
	}
	return nil
}

// Delete removes a setting
func (r *SettingsRepository) Delete(key string) error {
	_, err := r.db.db.Exec("DELETE FROM settings WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("delete setting: %w", err)
	}
	return nil
}

// GetAll returns all settings as a map
func (r *SettingsRepository) GetAll() (map[string]string, error) {
	rows, err := r.db.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		settings[key] = value
	}

	return settings, rows.Err()
}

// Common setting keys
const (
	SettingTheme          = "theme"
	SettingMinimizeToTray = "minimize_to_tray"
	SettingNotifications  = "notifications"
	SettingServerAddress  = "server_address"
	SettingAutoStart      = "auto_start"
)

// GetBool retrieves a boolean setting
func (r *SettingsRepository) GetBool(key string, defaultValue bool) bool {
	value, err := r.Get(key)
	if err != nil || value == "" {
		return defaultValue
	}
	return value == "true" || value == "1"
}

// SetBool stores a boolean setting
func (r *SettingsRepository) SetBool(key string, value bool) error {
	v := "false"
	if value {
		v = "true"
	}
	return r.Set(key, v)
}
