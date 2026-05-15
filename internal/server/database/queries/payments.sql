-- name: CreatePayment :one
INSERT INTO payments (user_id, subscription_id, invoice_id, amount, status, is_recurring, yookassa_data, provider, provider_data, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
RETURNING id, created_at;

-- name: GetPaymentByID :one
SELECT id, user_id, subscription_id, invoice_id, amount, status, is_recurring, yookassa_data, provider, provider_data, created_at
FROM payments WHERE id = $1;

-- name: GetPaymentByInvoiceID :one
SELECT id, user_id, subscription_id, invoice_id, amount, status, is_recurring, yookassa_data, provider, provider_data, created_at
FROM payments WHERE invoice_id = $1;

-- name: UpdatePayment :exec
UPDATE payments SET subscription_id = $2, status = $3, yookassa_data = $4, provider = $5, provider_data = $6
WHERE id = $1;

-- name: ListPaymentsByUserID :many
SELECT id, user_id, subscription_id, invoice_id, amount, status, is_recurring, yookassa_data, provider, provider_data, created_at
FROM payments WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountPaymentsByUserID :one
SELECT COUNT(*) FROM payments WHERE user_id = $1;

-- name: GetPendingPaymentsBySubscriptionID :many
SELECT id, user_id, subscription_id, invoice_id, amount, status, is_recurring, yookassa_data, provider, provider_data, created_at
FROM payments WHERE subscription_id = $1 AND status = 'pending' ORDER BY created_at DESC;

-- name: ListAllPayments :many
SELECT id, user_id, subscription_id, invoice_id, amount, status, is_recurring, yookassa_data, provider, provider_data, created_at
FROM payments ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountAllPayments :one
SELECT COUNT(*) FROM payments;

-- name: GetNextInvoiceID :one
SELECT COALESCE(MAX(invoice_id), 100000) + 1 AS next_id FROM payments;

-- name: ExpireStalePendingSubscriptions :exec
UPDATE subscriptions SET status = 'expired', updated_at = NOW()
WHERE status = 'pending' AND id IN (
    SELECT subscription_id FROM payments p WHERE p.status = 'pending' AND p.created_at < $1
);

-- name: FailStalePendingPayments :execrows
UPDATE payments SET status = 'failed' WHERE status = 'pending' AND created_at < $1;
