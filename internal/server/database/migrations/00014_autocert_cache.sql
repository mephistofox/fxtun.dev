-- +goose Up
-- Opaque blob store for golang.org/x/crypto/acme/autocert's Cache interface.
--
-- autocert keys are not domains: the ACME account key is stored under
-- "acme_account+key", HTTP-01 tokens under "…+token", etc. They never fitted
-- the tls_certificates table, so Put was a no-op and Get returned a cert
-- without its private key. The consequence was a brand new ACME account
-- registration on every process start (Let's Encrypt allows 10 per IP per 3
-- hours) and repeated re-issuance of certificates the cache could not return
-- (5 duplicates per week).
CREATE TABLE IF NOT EXISTS autocert_cache (
    key TEXT PRIMARY KEY,
    data BYTEA NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS autocert_cache;
