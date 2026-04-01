package database

import (
	"context"
	"fmt"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/database/sqlc"
)

// UserSettingsRepository handles user settings database operations using PostgreSQL via sqlc.
type UserSettingsRepository struct {
	q *sqlc.Queries
}

// Get retrieves a single setting value by user ID and key.
func (r *UserSettingsRepository) Get(userID int64, key string) (string, error) {
	ctx := context.Background()
	value, err := r.q.GetSetting(ctx, sqlc.GetSettingParams{
		UserID: userID,
		Key:    key,
	})
	if err != nil {
		if isNotFound(err) {
			return "", fmt.Errorf("setting %q not found", key)
		}
		return "", fmt.Errorf("get setting: %w", err)
	}
	return value, nil
}

// GetWithDefault retrieves a setting value, returning defaultValue if not found.
func (r *UserSettingsRepository) GetWithDefault(userID int64, key, defaultValue string) string {
	value, err := r.Get(userID, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetAll retrieves all settings for a user as a key-value map.
func (r *UserSettingsRepository) GetAll(userID int64) (map[string]string, error) {
	ctx := context.Background()
	rows, err := r.q.GetAllSettings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[row.Key] = row.Value
	}
	return result, nil
}

// GetAllWithTimestamps retrieves all settings for a user with their timestamps.
func (r *UserSettingsRepository) GetAllWithTimestamps(userID int64) ([]*UserSetting, error) {
	ctx := context.Background()
	rows, err := r.q.GetAllSettingsWithTimestamps(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get all settings with timestamps: %w", err)
	}
	settings := make([]*UserSetting, 0, len(rows))
	for _, row := range rows {
		settings = append(settings, &UserSetting{
			UserID:    row.UserID,
			Key:       row.Key,
			Value:     row.Value,
			UpdatedAt: tsToTime(row.UpdatedAt),
		})
	}
	return settings, nil
}

// Set creates or updates a single setting.
func (r *UserSettingsRepository) Set(userID int64, key, value string) error {
	ctx := context.Background()
	err := r.q.UpsertSetting(ctx, sqlc.UpsertSettingParams{
		UserID:    userID,
		Key:       key,
		Value:     value,
		UpdatedAt: timeToPgtz(time.Now()),
	})
	if err != nil {
		return fmt.Errorf("set setting: %w", err)
	}
	return nil
}

// SetBulk creates or updates multiple settings at once.
func (r *UserSettingsRepository) SetBulk(userID int64, settings map[string]string) error {
	ctx := context.Background()
	now := timeToPgtz(time.Now())
	for key, value := range settings {
		err := r.q.UpsertSetting(ctx, sqlc.UpsertSettingParams{
			UserID:    userID,
			Key:       key,
			Value:     value,
			UpdatedAt: now,
		})
		if err != nil {
			return fmt.Errorf("set setting %q: %w", key, err)
		}
	}
	return nil
}

// SyncBulk upserts settings only if the incoming timestamp is newer.
func (r *UserSettingsRepository) SyncBulk(userID int64, incoming []*UserSetting) error {
	ctx := context.Background()
	for _, s := range incoming {
		err := r.q.UpsertSettingIfNewer(ctx, sqlc.UpsertSettingIfNewerParams{
			UserID:    userID,
			Key:       s.Key,
			Value:     s.Value,
			UpdatedAt: timeToPgtz(s.UpdatedAt),
		})
		if err != nil {
			return fmt.Errorf("sync setting %q: %w", s.Key, err)
		}
	}
	return nil
}

// Delete removes a single setting by key.
func (r *UserSettingsRepository) Delete(userID int64, key string) error {
	ctx := context.Background()
	err := r.q.DeleteSetting(ctx, sqlc.DeleteSettingParams{
		UserID: userID,
		Key:    key,
	})
	if err != nil {
		return fmt.Errorf("delete setting: %w", err)
	}
	return nil
}

// Clear removes all settings for a user.
func (r *UserSettingsRepository) Clear(userID int64) error {
	ctx := context.Background()
	err := r.q.ClearSettings(ctx, userID)
	if err != nil {
		return fmt.Errorf("clear settings: %w", err)
	}
	return nil
}

// Count returns the number of settings for a user.
func (r *UserSettingsRepository) Count(userID int64) (int, error) {
	ctx := context.Background()
	count, err := r.q.CountSettingsByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count settings: %w", err)
	}
	return int(count), nil
}
