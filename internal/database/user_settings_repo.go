package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrSettingNotFound = errors.New("setting not found")

// UserSettingsRepository handles user settings database operations
type UserSettingsRepository struct {
	db *sql.DB
}

// NewUserSettingsRepository creates a new user settings repository
func NewUserSettingsRepository(db *sql.DB) *UserSettingsRepository {
	return &UserSettingsRepository{db: db}
}

// Get retrieves a setting value for a user
func (r *UserSettingsRepository) Get(userID int64, key string) (string, error) {
	query := `SELECT value FROM user_settings WHERE user_id = ? AND key = ?`

	var value string
	err := r.db.QueryRow(query, userID, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrSettingNotFound
		}
		return "", fmt.Errorf("get user setting: %w", err)
	}

	return value, nil
}

// GetWithDefault retrieves a setting value or returns the default if not found
func (r *UserSettingsRepository) GetWithDefault(userID int64, key, defaultValue string) string {
	value, err := r.Get(userID, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// Set sets a setting value for a user (upsert)
func (r *UserSettingsRepository) Set(userID int64, key, value string) error {
	query := `
		INSERT INTO user_settings (user_id, key, value, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, key) DO UPDATE SET value = ?, updated_at = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, userID, key, value, now, value, now)
	if err != nil {
		return fmt.Errorf("set user setting: %w", err)
	}

	return nil
}

// Delete deletes a setting for a user
func (r *UserSettingsRepository) Delete(userID int64, key string) error {
	query := `DELETE FROM user_settings WHERE user_id = ? AND key = ?`

	result, err := r.db.Exec(query, userID, key)
	if err != nil {
		return fmt.Errorf("delete user setting: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrSettingNotFound
	}

	return nil
}

// GetAll retrieves all settings for a user as a map
func (r *UserSettingsRepository) GetAll(userID int64) (map[string]string, error) {
	query := `SELECT key, value FROM user_settings WHERE user_id = ?`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("get all user settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan user setting: %w", err)
		}
		settings[key] = value
	}

	return settings, nil
}

// GetAllWithTimestamps retrieves all settings for a user with timestamps
func (r *UserSettingsRepository) GetAllWithTimestamps(userID int64) ([]*UserSetting, error) {
	query := `SELECT user_id, key, value, updated_at FROM user_settings WHERE user_id = ?`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("get all user settings: %w", err)
	}
	defer rows.Close()

	var settings []*UserSetting
	for rows.Next() {
		setting := &UserSetting{}
		if err := rows.Scan(&setting.UserID, &setting.Key, &setting.Value, &setting.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user setting: %w", err)
		}
		settings = append(settings, setting)
	}

	return settings, nil
}

// SetBulk sets multiple settings at once (upsert)
func (r *UserSettingsRepository) SetBulk(userID int64, settings map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO user_settings (user_id, key, value, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, key) DO UPDATE SET value = ?, updated_at = ?
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for key, value := range settings {
		_, err := stmt.Exec(userID, key, value, now, value, now)
		if err != nil {
			return fmt.Errorf("set setting %s: %w", key, err)
		}
	}

	return tx.Commit()
}

// SyncBulk synchronizes settings with conflict resolution by timestamp
func (r *UserSettingsRepository) SyncBulk(userID int64, incoming []*UserSetting) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current settings with timestamps
	query := `SELECT key, value, updated_at FROM user_settings WHERE user_id = ?`
	rows, err := tx.Query(query, userID)
	if err != nil {
		return fmt.Errorf("get current settings: %w", err)
	}

	existing := make(map[string]*UserSetting)
	for rows.Next() {
		setting := &UserSetting{UserID: userID}
		if err := rows.Scan(&setting.Key, &setting.Value, &setting.UpdatedAt); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan setting: %w", err)
		}
		existing[setting.Key] = setting
	}
	_ = rows.Close()

	// Upsert with timestamp comparison
	upsertQuery := `
		INSERT INTO user_settings (user_id, key, value, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, key) DO UPDATE SET value = ?, updated_at = ?
	`

	for _, inc := range incoming {
		if ex, ok := existing[inc.Key]; ok {
			// Only update if incoming is newer
			if !inc.UpdatedAt.After(ex.UpdatedAt) {
				continue
			}
		}

		_, err := tx.Exec(upsertQuery, userID, inc.Key, inc.Value, inc.UpdatedAt, inc.Value, inc.UpdatedAt)
		if err != nil {
			return fmt.Errorf("upsert setting %s: %w", inc.Key, err)
		}
	}

	return tx.Commit()
}

// Clear deletes all settings for a user
func (r *UserSettingsRepository) Clear(userID int64) error {
	query := `DELETE FROM user_settings WHERE user_id = ?`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("clear user settings: %w", err)
	}
	return nil
}

// Count returns the number of settings for a user
func (r *UserSettingsRepository) Count(userID int64) (int, error) {
	query := `SELECT COUNT(*) FROM user_settings WHERE user_id = ?`
	var count int
	if err := r.db.QueryRow(query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count user settings: %w", err)
	}
	return count, nil
}
