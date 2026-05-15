-- name: CreateSubscription :one
INSERT INTO subscriptions (user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
RETURNING id, created_at, updated_at;

-- name: GetSubscriptionByID :one
SELECT id, user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at
FROM subscriptions WHERE id = $1;

-- name: GetActiveSubscriptionByUserID :one
SELECT id, user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at
FROM subscriptions WHERE user_id = $1 AND status IN ('active', 'cancelled') ORDER BY created_at DESC LIMIT 1;

-- name: GetPendingSubscriptionByUserID :one
SELECT id, user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at
FROM subscriptions WHERE user_id = $1 AND status = 'pending' ORDER BY created_at DESC LIMIT 1;

-- name: GetSubscriptionByCreemID :one
SELECT id, user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at
FROM subscriptions WHERE creem_subscription_id = $1;

-- name: ListSubscriptionsByUserID :many
SELECT id, user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at
FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC;

-- name: ListAllSubscriptions :many
SELECT id, user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at
FROM subscriptions ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountAllSubscriptions :one
SELECT COUNT(*) FROM subscriptions;

-- name: UpdateSubscription :exec
UPDATE subscriptions SET plan_id = $2, next_plan_id = $3, status = $4, recurring = $5, current_period_start = $6, current_period_end = $7, yookassa_payment_method_id = $8, creem_customer_id = $9, creem_subscription_id = $10, updated_at = NOW()
WHERE id = $1;

-- name: DeleteSubscription :exec
DELETE FROM subscriptions WHERE id = $1;

-- name: GetExpiringSubscriptions :many
SELECT id, user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at
FROM subscriptions WHERE status = 'active' AND recurring = TRUE AND current_period_end <= $1;

-- name: GetExpiredSubscriptions :many
SELECT id, user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at
FROM subscriptions WHERE status IN ('active', 'cancelled') AND current_period_end < NOW();

-- name: GetSubscriptionsWithPendingPlanChange :many
SELECT id, user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at
FROM subscriptions WHERE next_plan_id IS NOT NULL AND current_period_end < NOW();

-- name: GetSubscriptionsForRenewalReminder :many
SELECT id, user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id, creem_customer_id, creem_subscription_id, created_at, updated_at
FROM subscriptions WHERE status = 'active' AND recurring = TRUE AND current_period_end >= $1 AND current_period_end < $2;
