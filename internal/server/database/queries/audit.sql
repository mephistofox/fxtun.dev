-- name: CreateAuditLog :exec
INSERT INTO audit_logs (user_id, action, details, ip_address, created_at)
VALUES ($1, $2, $3, $4, NOW());

-- name: ListAuditLogsByUserID :many
SELECT id, user_id, action, details, ip_address, created_at
FROM audit_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountAuditLogsByUserID :one
SELECT COUNT(*) FROM audit_logs WHERE user_id = $1;

-- name: ListAuditLogs :many
SELECT id, user_id, action, details, ip_address, created_at
FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs;

-- name: ListAuditLogsByAction :many
SELECT id, user_id, action, details, ip_address, created_at
FROM audit_logs WHERE action = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountAuditLogsByAction :one
SELECT COUNT(*) FROM audit_logs WHERE action = $1;

-- name: DeleteAuditLogsOlderThan :execrows
DELETE FROM audit_logs WHERE created_at < $1;

-- name: GetLatestAuditLogByUserAndAction :one
SELECT id, user_id, action, details, ip_address, created_at
FROM audit_logs WHERE user_id = $1 AND action = $2
ORDER BY created_at DESC LIMIT 1;
