-- +goose Up
ALTER TABLE plans ADD COLUMN max_data_sessions INTEGER NOT NULL DEFAULT 8;

UPDATE plans SET max_data_sessions = 8 WHERE slug = 'free';
UPDATE plans SET max_data_sessions = 16 WHERE slug = 'base';
UPDATE plans SET max_data_sessions = 24 WHERE slug = 'pro';
UPDATE plans SET max_data_sessions = 48 WHERE slug = 'business';
UPDATE plans SET max_data_sessions = -1 WHERE slug = 'admin';

-- +goose Down
ALTER TABLE plans DROP COLUMN max_data_sessions;
