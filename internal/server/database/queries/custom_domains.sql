-- name: CreateCustomDomain :one
INSERT INTO custom_domains (user_id, domain, target_subdomain, verification_token, verified, created_at)
VALUES ($1, $2, $3, $4, FALSE, NOW())
RETURNING id, created_at;

-- name: GetCustomDomainByID :one
SELECT id, user_id, domain, target_subdomain, verified, verified_at, created_at, verification_token
FROM custom_domains WHERE id = $1;

-- name: GetCustomDomainByDomain :one
SELECT id, user_id, domain, target_subdomain, verified, verified_at, created_at, verification_token
FROM custom_domains WHERE domain = $1;

-- name: ListCustomDomainsByUserID :many
SELECT id, user_id, domain, target_subdomain, verified, verified_at, created_at, verification_token
FROM custom_domains WHERE user_id = $1 ORDER BY created_at DESC;

-- name: ListAllCustomDomains :many
SELECT id, user_id, domain, target_subdomain, verified, verified_at, created_at, verification_token
FROM custom_domains ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountAllCustomDomains :one
SELECT COUNT(*) FROM custom_domains;

-- name: ListVerifiedCustomDomains :many
SELECT id, user_id, domain, target_subdomain, verified, verified_at, created_at, verification_token
FROM custom_domains WHERE verified = TRUE;

-- name: CountCustomDomainsByUserID :one
SELECT COUNT(*) FROM custom_domains WHERE user_id = $1;

-- name: SetCustomDomainVerified :exec
UPDATE custom_domains SET verified = $2, verified_at = $3 WHERE id = $1;

-- name: SetCustomDomainVerificationToken :exec
UPDATE custom_domains SET verification_token = $2 WHERE id = $1;

-- name: DeleteCustomDomain :exec
DELETE FROM custom_domains WHERE id = $1;
