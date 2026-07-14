-- +goose Up
-- Store the last 4 digits of the card bound for YooKassa autopayments so the
-- profile can display the saved payment method (e.g. "•••• 1234") and offer a
-- self-service unbind, as required to enable recurring payments in production.
ALTER TABLE subscriptions ADD COLUMN yookassa_card_last4 TEXT;

-- +goose Down
ALTER TABLE subscriptions DROP COLUMN IF EXISTS yookassa_card_last4;
