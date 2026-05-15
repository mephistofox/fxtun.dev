-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, token_hash, name, allowed_subdomains, max_tunnels, allowed_ips, created_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
RETURNING id, created_at;

-- name: GetAPITokenByID :one
SELECT id, user_id, token_hash, name, allowed_subdomains, max_tunnels, allowed_ips, last_used_at, created_at
FROM api_tokens WHERE id = $1;

-- name: GetAPITokenByHash :one
SELECT id, user_id, token_hash, name, allowed_subdomains, max_tunnels, allowed_ips, last_used_at, created_at
FROM api_tokens WHERE token_hash = $1;

-- name: ListAPITokensByUserID :many
SELECT id, user_id, token_hash, name, allowed_subdomains, max_tunnels, allowed_ips, last_used_at, created_at
FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC;

-- name: DeleteAPIToken :exec
DELETE FROM api_tokens WHERE id = $1;

-- name: DeleteAPITokensByUserID :exec
DELETE FROM api_tokens WHERE user_id = $1;

-- name: UpdateAPITokenLastUsed :exec
UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1;

-- name: CountAPITokensByUserID :one
SELECT COUNT(*) FROM api_tokens WHERE user_id = $1;
