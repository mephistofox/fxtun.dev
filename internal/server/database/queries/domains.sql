-- name: CreateReservedDomain :one
INSERT INTO reserved_domains (user_id, subdomain, created_at)
VALUES ($1, $2, NOW())
RETURNING id, created_at;

-- name: GetReservedDomainByID :one
SELECT id, user_id, subdomain, created_at FROM reserved_domains WHERE id = $1;

-- name: GetReservedDomainBySubdomain :one
SELECT id, user_id, subdomain, created_at FROM reserved_domains WHERE subdomain = $1;

-- name: ListReservedDomainsByUserID :many
SELECT id, user_id, subdomain, created_at FROM reserved_domains WHERE user_id = $1 ORDER BY created_at DESC;

-- name: DeleteReservedDomain :exec
DELETE FROM reserved_domains WHERE id = $1;

-- name: DeleteReservedDomainsByUserID :exec
DELETE FROM reserved_domains WHERE user_id = $1;

-- name: CountReservedDomainsByUserID :one
SELECT COUNT(*) FROM reserved_domains WHERE user_id = $1;

-- name: IsSubdomainAvailable :one
SELECT COUNT(*) = 0 AS available FROM reserved_domains WHERE subdomain = $1;

-- name: IsSubdomainOwnedByUser :one
SELECT COUNT(*) > 0 AS owned FROM reserved_domains WHERE subdomain = $1 AND user_id = $2;
