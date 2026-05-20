-- +goose Up
ALTER TABLE plans ADD COLUMN udp_enabled BOOLEAN NOT NULL DEFAULT true;

-- Free tier: UDP disabled. Paid tiers keep the default (true).
UPDATE plans SET udp_enabled = false WHERE slug = 'free';

-- +goose Down
ALTER TABLE plans DROP COLUMN udp_enabled;
