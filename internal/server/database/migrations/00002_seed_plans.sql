-- +goose Up
INSERT INTO plans (slug, name, price, max_tunnels, max_domains, max_custom_domains, max_tokens, max_tunnels_per_token, inspector_enabled, is_public, bandwidth_mbps) VALUES
    ('free', 'Free', 0, 3, 1, 0, 1, 3, FALSE, TRUE, 10),
    ('base', 'Base', 5, 5, 5, 1, 5, 5, TRUE, TRUE, 50),
    ('pro', 'Pro', 10, 15, 15, 5, 10, 10, TRUE, TRUE, 100),
    ('business', 'Business', 20, 50, 50, 50, 50, 50, TRUE, TRUE, 0),
    ('admin', 'Admin', 0, -1, -1, -1, -1, -1, TRUE, FALSE, 0)
ON CONFLICT (slug) DO NOTHING;

-- +goose Down
DELETE FROM plans WHERE slug IN ('free', 'base', 'pro', 'business', 'admin');
