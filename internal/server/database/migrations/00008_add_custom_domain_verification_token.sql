-- +goose Up
-- Per-domain ownership-proof token. The user must publish it as a TXT record
-- at _fxtunnel-challenge.<domain> before the domain is verified — proving
-- control of the domain rather than merely pointing an A-record at the shared
-- server IP (which any tenant could do, enabling cross-tenant takeover).
ALTER TABLE custom_domains ADD COLUMN verification_token TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE custom_domains DROP COLUMN verification_token;
