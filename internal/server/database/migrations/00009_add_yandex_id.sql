-- +goose Up
-- Add Yandex OAuth identity column (mirrors google_id).
ALTER TABLE users ADD COLUMN yandex_id VARCHAR(255);
CREATE UNIQUE INDEX idx_users_yandex_id ON users(yandex_id) WHERE yandex_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_users_yandex_id;
ALTER TABLE users DROP COLUMN IF EXISTS yandex_id;
