-- name: GetSetting :one
SELECT value FROM user_settings WHERE user_id = $1 AND key = $2;

-- name: GetAllSettings :many
SELECT key, value FROM user_settings WHERE user_id = $1;

-- name: GetAllSettingsWithTimestamps :many
SELECT user_id, key, value, updated_at FROM user_settings WHERE user_id = $1;

-- name: UpsertSetting :exec
INSERT INTO user_settings (user_id, key, value, updated_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at;

-- name: UpsertSettingIfNewer :exec
INSERT INTO user_settings (user_id, key, value, updated_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
WHERE EXCLUDED.updated_at > user_settings.updated_at;

-- name: DeleteSetting :exec
DELETE FROM user_settings WHERE user_id = $1 AND key = $2;

-- name: ClearSettings :exec
DELETE FROM user_settings WHERE user_id = $1;

-- name: CountSettingsByUserID :one
SELECT COUNT(*) FROM user_settings WHERE user_id = $1;
