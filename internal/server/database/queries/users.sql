-- name: CreateUser :one
INSERT INTO users (phone, password_hash, display_name, is_admin, is_active, plan_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
RETURNING id, created_at;

-- name: CreateOAuthUser :one
INSERT INTO users (phone, password_hash, display_name, is_admin, is_active, github_id, google_id, email, avatar_url, plan_id, created_at)
VALUES ($1, '', $2, $3, $4, $5, $6, $7, $8, $9, NOW())
RETURNING id, created_at;

-- name: GetUserByID :one
SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id, plan_id, first_tunnel_at
FROM users WHERE id = $1;

-- name: GetUserByPhone :one
SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id, plan_id, first_tunnel_at
FROM users WHERE phone = $1;

-- name: GetUserByEmail :one
SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id, plan_id, first_tunnel_at
FROM users WHERE email = $1;

-- name: GetUserByGitHubID :one
SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id, plan_id, first_tunnel_at
FROM users WHERE github_id = $1;

-- name: GetUserByGoogleID :one
SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id, plan_id, first_tunnel_at
FROM users WHERE google_id = $1;

-- name: GetUsersByIDs :many
SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id, plan_id, first_tunnel_at
FROM users WHERE id = ANY($1::bigint[]);

-- name: UpdateUser :exec
UPDATE users SET display_name = $2, is_admin = $3, is_active = $4, last_login_at = $5, plan_id = $6
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: UpdateUserEmail :exec
UPDATE users SET email = $2 WHERE id = $1;

-- name: UpdateUserPhone :exec
UPDATE users SET phone = $2 WHERE id = $1;

-- name: UpdateUserLastLogin :exec
UPDATE users SET last_login_at = NOW() WHERE id = $1;

-- name: UpdateUserPlan :exec
UPDATE users SET plan_id = $2 WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: LinkGitHub :exec
UPDATE users SET github_id = $2,
    email = COALESCE(NULLIF(email, ''), $3),
    avatar_url = COALESCE(NULLIF(avatar_url, ''), $4)
WHERE id = $1;

-- name: LinkGoogle :exec
UPDATE users SET google_id = $2,
    email = COALESCE(NULLIF(email, ''), $3),
    avatar_url = COALESCE(NULLIF(avatar_url, ''), $4)
WHERE id = $1;

-- name: SetFirstTunnelAt :execrows
UPDATE users SET first_tunnel_at = $2 WHERE id = $1 AND first_tunnel_at IS NULL;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: ListUsersFiltered :many
SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id, plan_id, first_tunnel_at
FROM users
WHERE (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
  AND (sqlc.narg('is_admin')::boolean IS NULL OR is_admin = sqlc.narg('is_admin'))
  AND (sqlc.narg('search')::text IS NULL OR
       LOWER(email) LIKE sqlc.narg('search') ESCAPE '\' OR
       LOWER(phone) LIKE sqlc.narg('search') ESCAPE '\' OR
       LOWER(display_name) LIKE sqlc.narg('search') ESCAPE '\')
ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountUsersFiltered :one
SELECT COUNT(*)
FROM users
WHERE (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
  AND (sqlc.narg('is_admin')::boolean IS NULL OR is_admin = sqlc.narg('is_admin'))
  AND (sqlc.narg('search')::text IS NULL OR
       LOWER(email) LIKE sqlc.narg('search') ESCAPE '\' OR
       LOWER(phone) LIKE sqlc.narg('search') ESCAPE '\' OR
       LOWER(display_name) LIKE sqlc.narg('search') ESCAPE '\');

-- name: GetUserStats :one
SELECT
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE is_active = TRUE) AS active,
    COUNT(*) FILTER (WHERE is_active = FALSE) AS blocked,
    COUNT(*) FILTER (WHERE is_admin = TRUE) AS admins
FROM users
WHERE (sqlc.narg('search')::text IS NULL OR
       LOWER(email) LIKE sqlc.narg('search') ESCAPE '\' OR
       LOWER(phone) LIKE sqlc.narg('search') ESCAPE '\' OR
       LOWER(display_name) LIKE sqlc.narg('search') ESCAPE '\');
