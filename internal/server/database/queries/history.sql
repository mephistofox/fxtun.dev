-- name: CreateHistoryEntry :one
INSERT INTO user_history (user_id, bundle_name, tunnel_type, local_port, remote_addr, url, connected_at, disconnected_at, bytes_sent, bytes_received)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: UpdateHistoryEntry :exec
UPDATE user_history SET disconnected_at = $3, bytes_sent = $4, bytes_received = $5
WHERE id = $1 AND user_id = $2;

-- name: GetHistoryEntryByID :one
SELECT id, user_id, bundle_name, tunnel_type, local_port, remote_addr, url, connected_at, disconnected_at, bytes_sent, bytes_received
FROM user_history WHERE id = $1 AND user_id = $2;

-- name: ListHistoryByUserID :many
SELECT id, user_id, bundle_name, tunnel_type, local_port, remote_addr, url, connected_at, disconnected_at, bytes_sent, bytes_received
FROM user_history WHERE user_id = $1 ORDER BY connected_at DESC LIMIT $2 OFFSET $3;

-- name: ClearHistory :exec
DELETE FROM user_history WHERE user_id = $1;

-- name: CountHistoryByUserID :one
SELECT COUNT(*) FROM user_history WHERE user_id = $1;

-- name: GetHistoryStats :one
SELECT
    COUNT(*) AS total_connections,
    COALESCE(SUM(bytes_sent), 0) AS total_bytes_sent,
    COALESCE(SUM(bytes_received), 0) AS total_bytes_received
FROM user_history WHERE user_id = $1;

-- name: DeleteHistoryOlderThan :execrows
DELETE FROM user_history WHERE user_id = $1 AND connected_at < $2;
