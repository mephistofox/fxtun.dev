-- +goose Up
CREATE TABLE edge_nodes (
    id                BIGSERIAL PRIMARY KEY,
    node_id           TEXT UNIQUE NOT NULL,
    name              TEXT NOT NULL,
    region            TEXT NOT NULL DEFAULT '',
    public_addr       TEXT NOT NULL,
    http_addr         TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'pending',
    approved_at       TIMESTAMPTZ,
    approved_by       BIGINT REFERENCES users(id) ON DELETE SET NULL,
    last_heartbeat_at TIMESTAMPTZ,
    version           TEXT NOT NULL DEFAULT '',
    metadata          JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_edge_nodes_status ON edge_nodes(status);

-- +goose Down
DROP TABLE IF EXISTS edge_nodes;
