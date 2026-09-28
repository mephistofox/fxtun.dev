-- +goose Up
-- payments(subscription_id) is scanned once per subscription on every hourly
-- scheduler tick; subscriptions(creem_subscription_id) is scanned on every webhook.
CREATE INDEX IF NOT EXISTS idx_payments_subscription_id ON payments(subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_creem_subscription_id
    ON subscriptions(creem_subscription_id) WHERE creem_subscription_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_subscriptions_creem_subscription_id;
DROP INDEX IF EXISTS idx_payments_subscription_id;
