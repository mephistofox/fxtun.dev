-- +goose Up
-- Enforce at most one pending subscription per user. This closes a checkout race
-- where two concurrent requests from the same user could each pass the "no pending
-- subscription" check and then each create a pending subscription + a live provider
-- payment, letting the user pay both and be charged twice.
--
-- Existing data is deduplicated first so the unique index can be built on
-- production: for each user keep only the most recent pending subscription, expire
-- the rest, and fail their still-pending payments (the scheduler would do the same
-- within an hour anyway).

-- +goose StatementBegin
WITH dupes AS (
    SELECT id
    FROM subscriptions
    WHERE status = 'pending'
      AND id NOT IN (
          SELECT DISTINCT ON (user_id) id
          FROM subscriptions
          WHERE status = 'pending'
          ORDER BY user_id, created_at DESC
      )
),
expired AS (
    UPDATE subscriptions
    SET status = 'expired', updated_at = NOW()
    WHERE id IN (SELECT id FROM dupes)
    RETURNING id
)
UPDATE payments
SET status = 'failed'
WHERE status = 'pending'
  AND subscription_id IN (SELECT id FROM expired);
-- +goose StatementEnd

CREATE UNIQUE INDEX uniq_pending_subscription_per_user
    ON subscriptions (user_id)
    WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS uniq_pending_subscription_per_user;
