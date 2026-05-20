-- name: GetPlanByID :one
SELECT id, slug, name, price, max_tunnels, max_domains, max_custom_domains,
       max_tokens, max_tunnels_per_token, inspector_enabled, is_public,
       is_recommended, bandwidth_mbps, rate_limit_tcp, rate_limit_udp,
       rate_limit_http, creem_product_id, max_data_sessions, udp_enabled
FROM plans WHERE id = $1;

-- name: GetPlanBySlug :one
SELECT id, slug, name, price, max_tunnels, max_domains, max_custom_domains,
       max_tokens, max_tunnels_per_token, inspector_enabled, is_public,
       is_recommended, bandwidth_mbps, rate_limit_tcp, rate_limit_udp,
       rate_limit_http, creem_product_id, max_data_sessions, udp_enabled
FROM plans WHERE slug = $1;

-- name: GetDefaultPlan :one
SELECT id, slug, name, price, max_tunnels, max_domains, max_custom_domains,
       max_tokens, max_tunnels_per_token, inspector_enabled, is_public,
       is_recommended, bandwidth_mbps, rate_limit_tcp, rate_limit_udp,
       rate_limit_http, creem_product_id, max_data_sessions, udp_enabled
FROM plans WHERE slug = 'free' LIMIT 1;

-- name: ListPlans :many
SELECT id, slug, name, price, max_tunnels, max_domains, max_custom_domains,
       max_tokens, max_tunnels_per_token, inspector_enabled, is_public,
       is_recommended, bandwidth_mbps, rate_limit_tcp, rate_limit_udp,
       rate_limit_http, creem_product_id, max_data_sessions, udp_enabled
FROM plans ORDER BY price ASC;

-- name: ListPublicPlans :many
SELECT id, slug, name, price, max_tunnels, max_domains, max_custom_domains,
       max_tokens, max_tunnels_per_token, inspector_enabled, is_public,
       is_recommended, bandwidth_mbps, rate_limit_tcp, rate_limit_udp,
       rate_limit_http, creem_product_id, max_data_sessions, udp_enabled
FROM plans WHERE is_public = TRUE ORDER BY price ASC;

-- name: ListAllPlans :many
SELECT id, slug, name, price, max_tunnels, max_domains, max_custom_domains,
       max_tokens, max_tunnels_per_token, inspector_enabled, is_public,
       is_recommended, bandwidth_mbps, rate_limit_tcp, rate_limit_udp,
       rate_limit_http, creem_product_id, max_data_sessions, udp_enabled
FROM plans ORDER BY price ASC LIMIT $1 OFFSET $2;

-- name: CountAllPlans :one
SELECT COUNT(*) FROM plans;

-- name: CreatePlan :one
INSERT INTO plans (slug, name, price, max_tunnels, max_domains, max_custom_domains,
                   max_tokens, max_tunnels_per_token, inspector_enabled, is_public,
                   is_recommended, bandwidth_mbps, rate_limit_tcp, rate_limit_udp,
                   rate_limit_http, creem_product_id, max_data_sessions, udp_enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
RETURNING id;

-- name: UpdatePlan :exec
UPDATE plans SET
    name = $2, price = $3, max_tunnels = $4, max_domains = $5,
    max_custom_domains = $6, max_tokens = $7, max_tunnels_per_token = $8,
    inspector_enabled = $9, is_public = $10, is_recommended = $11,
    bandwidth_mbps = $12, rate_limit_tcp = $13, rate_limit_udp = $14,
    rate_limit_http = $15, creem_product_id = $16, max_data_sessions = $17,
    udp_enabled = $18
WHERE id = $1;

-- name: DeletePlan :exec
DELETE FROM plans WHERE id = $1;

-- name: CountPlanUsers :one
SELECT COUNT(*) FROM users WHERE plan_id = $1;
