-- name: CreateSession :one
INSERT INTO sessions (user_id, refresh_token_hash, user_agent, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at;

-- name: GetSessionByTokenHash :one
SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at
FROM sessions WHERE refresh_token_hash = $1;

-- name: GetSessionsByUserID :many
SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at
FROM sessions WHERE user_id = $1 ORDER BY created_at DESC;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions WHERE refresh_token_hash = $1;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at < NOW();
