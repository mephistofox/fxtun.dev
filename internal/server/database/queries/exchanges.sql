-- name: SaveExchange :exec
INSERT INTO inspect_exchanges (id, tunnel_id, user_id, trace_id, replay_ref, timestamp, duration_ns, method, path, host, request_headers, request_body, request_body_size, response_headers, response_body, response_body_size, status_code, remote_addr)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18);

-- name: GetExchangeByID :one
SELECT id, tunnel_id, trace_id, replay_ref, timestamp, duration_ns, method, path, host, request_headers, request_body, request_body_size, response_headers, response_body, response_body_size, status_code, remote_addr
FROM inspect_exchanges WHERE id = $1;

-- name: ListExchangesByTunnelID :many
SELECT id, tunnel_id, trace_id, replay_ref, timestamp, duration_ns, method, path, host, request_headers, request_body, request_body_size, response_headers, response_body, response_body_size, status_code, remote_addr
FROM inspect_exchanges WHERE tunnel_id = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3;

-- name: CountExchangesByTunnelID :one
SELECT COUNT(*) FROM inspect_exchanges WHERE tunnel_id = $1;

-- name: ListExchangesByHostAndUser :many
SELECT id, tunnel_id, trace_id, replay_ref, timestamp, duration_ns, method, path, host, request_headers, request_body, request_body_size, response_headers, response_body, response_body_size, status_code, remote_addr
FROM inspect_exchanges WHERE host = $1 AND user_id = $2 ORDER BY timestamp DESC LIMIT $3 OFFSET $4;

-- name: CountExchangesByHostAndUser :one
SELECT COUNT(*) FROM inspect_exchanges WHERE host = $1 AND user_id = $2;

-- name: DeleteExchangesOlderThan :execrows
DELETE FROM inspect_exchanges WHERE created_at < $1;

-- name: DeleteExchangesByTunnelID :execrows
DELETE FROM inspect_exchanges WHERE tunnel_id = $1;
