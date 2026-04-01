-- +goose Up

-- Plans (created first because users references it)
CREATE TABLE plans (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    price DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_tunnels INTEGER NOT NULL DEFAULT 3,
    max_domains INTEGER NOT NULL DEFAULT 1,
    max_custom_domains INTEGER NOT NULL DEFAULT 0,
    max_tokens INTEGER NOT NULL DEFAULT 1,
    max_tunnels_per_token INTEGER NOT NULL DEFAULT 3,
    inspector_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    is_recommended BOOLEAN NOT NULL DEFAULT FALSE,
    bandwidth_mbps INTEGER NOT NULL DEFAULT 0,
    rate_limit_tcp INTEGER NOT NULL DEFAULT 0,
    rate_limit_udp INTEGER NOT NULL DEFAULT 0,
    rate_limit_http INTEGER NOT NULL DEFAULT 0,
    creem_product_id TEXT NOT NULL DEFAULT ''
);

-- Users
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    phone VARCHAR(20),
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    display_name VARCHAR(100),
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,
    github_id BIGINT,
    email VARCHAR(255),
    avatar_url TEXT,
    google_id VARCHAR(255),
    plan_id BIGINT REFERENCES plans(id),
    first_tunnel_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_users_phone ON users(phone) WHERE phone IS NOT NULL;
CREATE UNIQUE INDEX idx_users_github_id ON users(github_id) WHERE github_id IS NOT NULL;
CREATE UNIQUE INDEX idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL;
CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL AND email != '';

-- Invite codes
CREATE TABLE invite_codes (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(32) UNIQUE NOT NULL,
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    used_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invite_codes_code ON invite_codes(code);

-- Reserved domains
CREATE TABLE reserved_domains (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subdomain VARCHAR(63) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reserved_domains_user ON reserved_domains(user_id);

-- Sessions
CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    user_agent VARCHAR(255),
    ip_address VARCHAR(45),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
CREATE INDEX idx_sessions_refresh_token_hash ON sessions(refresh_token_hash);

-- API tokens
CREATE TABLE api_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    allowed_subdomains JSONB NOT NULL DEFAULT '[]',
    max_tunnels INTEGER NOT NULL DEFAULT 10,
    allowed_ips JSONB NOT NULL DEFAULT '[]',
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_tokens_user ON api_tokens(user_id);
CREATE INDEX idx_api_tokens_token_hash ON api_tokens(token_hash);

-- TOTP secrets
CREATE TABLE totp_secrets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret_encrypted VARCHAR(255) NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    backup_codes JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit logs
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB,
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);

-- User bundles
CREATE TABLE user_bundles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('http', 'tcp', 'udp')),
    local_port INTEGER NOT NULL,
    subdomain TEXT,
    remote_port INTEGER,
    auto_connect BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_user_bundles_user ON user_bundles(user_id);

-- User history
CREATE TABLE user_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bundle_name TEXT,
    tunnel_type TEXT NOT NULL,
    local_port INTEGER NOT NULL,
    remote_addr TEXT,
    url TEXT,
    connected_at TIMESTAMPTZ NOT NULL,
    disconnected_at TIMESTAMPTZ,
    bytes_sent BIGINT NOT NULL DEFAULT 0,
    bytes_received BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_user_history_user ON user_history(user_id);
CREATE INDEX idx_user_history_connected ON user_history(connected_at);

-- User settings
CREATE TABLE user_settings (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(user_id, key)
);

-- Custom domains
CREATE TABLE custom_domains (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain TEXT UNIQUE NOT NULL,
    target_subdomain TEXT NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_custom_domains_user ON custom_domains(user_id);
CREATE INDEX idx_custom_domains_domain ON custom_domains(domain);

-- TLS certificates
CREATE TABLE tls_certificates (
    id BIGSERIAL PRIMARY KEY,
    domain TEXT UNIQUE NOT NULL,
    cert_pem BYTEA NOT NULL,
    key_pem BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Subscriptions
CREATE TABLE subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id BIGINT NOT NULL REFERENCES plans(id),
    next_plan_id BIGINT REFERENCES plans(id),
    status TEXT NOT NULL DEFAULT 'pending',
    recurring BOOLEAN NOT NULL DEFAULT TRUE,
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    yookassa_payment_method_id TEXT,
    creem_customer_id TEXT,
    creem_subscription_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_period_end ON subscriptions(current_period_end);

-- Payments
CREATE TABLE payments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id BIGINT REFERENCES subscriptions(id) ON DELETE SET NULL,
    invoice_id BIGINT NOT NULL UNIQUE,
    amount DOUBLE PRECISION NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
    yookassa_data TEXT,
    provider TEXT NOT NULL DEFAULT 'yookassa',
    provider_data TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_invoice_id ON payments(invoice_id);
CREATE INDEX idx_payments_status ON payments(status);

-- Inspect exchanges
CREATE TABLE inspect_exchanges (
    id TEXT PRIMARY KEY,
    tunnel_id TEXT NOT NULL,
    user_id BIGINT NOT NULL,
    trace_id TEXT,
    replay_ref TEXT,
    timestamp TIMESTAMPTZ NOT NULL,
    duration_ns BIGINT NOT NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    host TEXT NOT NULL,
    request_headers JSONB,
    request_body BYTEA,
    request_body_size INTEGER NOT NULL DEFAULT 0,
    response_headers JSONB,
    response_body BYTEA,
    response_body_size INTEGER NOT NULL DEFAULT 0,
    status_code INTEGER NOT NULL,
    remote_addr TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inspect_exch_tunnel ON inspect_exchanges(tunnel_id, timestamp DESC);
CREATE INDEX idx_inspect_exch_created ON inspect_exchanges(created_at);
CREATE INDEX idx_inspect_exch_host_user ON inspect_exchanges(host, user_id, timestamp DESC);

-- +goose Down
DROP TABLE IF EXISTS inspect_exchanges;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS tls_certificates;
DROP TABLE IF EXISTS custom_domains;
DROP TABLE IF EXISTS user_settings;
DROP TABLE IF EXISTS user_history;
DROP TABLE IF EXISTS user_bundles;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS totp_secrets;
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS reserved_domains;
DROP TABLE IF EXISTS invite_codes;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS plans;
