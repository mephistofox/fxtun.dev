-- name: CreateBundle :one
INSERT INTO user_bundles (user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
RETURNING id, created_at, updated_at;

-- name: UpdateBundle :exec
UPDATE user_bundles SET name = $3, type = $4, local_port = $5, subdomain = $6, remote_port = $7, auto_connect = $8, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: DeleteBundle :exec
DELETE FROM user_bundles WHERE id = $1 AND user_id = $2;

-- name: DeleteBundleByName :exec
DELETE FROM user_bundles WHERE user_id = $1 AND name = $2;

-- name: DeleteAllBundles :exec
DELETE FROM user_bundles WHERE user_id = $1;

-- name: GetBundleByID :one
SELECT id, user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
FROM user_bundles WHERE id = $1 AND user_id = $2;

-- name: GetBundleByName :one
SELECT id, user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
FROM user_bundles WHERE user_id = $1 AND name = $2;

-- name: ListBundlesByUserID :many
SELECT id, user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
FROM user_bundles WHERE user_id = $1 ORDER BY name;

-- name: CountBundlesByUserID :one
SELECT COUNT(*) FROM user_bundles WHERE user_id = $1;

-- name: UpsertBundle :one
INSERT INTO user_bundles (user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, name) DO UPDATE SET
    type = EXCLUDED.type,
    local_port = EXCLUDED.local_port,
    subdomain = EXCLUDED.subdomain,
    remote_port = EXCLUDED.remote_port,
    auto_connect = EXCLUDED.auto_connect,
    updated_at = EXCLUDED.updated_at
WHERE EXCLUDED.updated_at > user_bundles.updated_at
RETURNING id, created_at, updated_at;
