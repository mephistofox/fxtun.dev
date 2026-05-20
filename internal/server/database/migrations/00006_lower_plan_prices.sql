-- +goose Up
-- Align plan prices with landing page pricing.
-- base/Starter: $5 → $2.50, pro: $10 → $5, business: $20 → $7.50.
-- Existing subscriptions in production retain their original billed prices
-- (this only affects new checkouts and the public /api/plans/public list).
UPDATE plans SET price = 2.50 WHERE slug = 'base';
UPDATE plans SET price = 5.00 WHERE slug = 'pro';
UPDATE plans SET price = 7.50 WHERE slug = 'business';

-- +goose Down
UPDATE plans SET price = 5.00 WHERE slug = 'base';
UPDATE plans SET price = 10.00 WHERE slug = 'pro';
UPDATE plans SET price = 20.00 WHERE slug = 'business';
