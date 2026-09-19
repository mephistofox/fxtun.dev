-- +goose Up
-- Custom domains must be unique as DNS names, i.e. case-insensitively. The
-- plain UNIQUE constraint on custom_domains.domain is case-sensitive, so a
-- tenant could register "EXAMPLE.com" next to another tenant's verified
-- "example.com". Deleting that squatter row then removed the victim's entry
-- from the runtime routing map (custom-domain lookup/removal lowercases), so
-- the victim's domain answered "Tunnel not found" until the next restart.
--
-- Existing data is made index-safe first, without deleting anything:
--   1. Rows that collide with an older row under the canonical form are parked
--      under a "+dup<id>" name and un-verified. "+" cannot appear in a
--      hostname, so a parked row can never match a request again, but the row
--      itself (and its owner) is preserved for manual review.
--   2. Every remaining row is rewritten to its canonical form.
-- Both steps are no-ops on clean data.

-- +goose StatementBegin
UPDATE custom_domains c
SET domain = c.domain || '+dup' || c.id,
    verified = FALSE,
    verified_at = NULL
WHERE EXISTS (
    SELECT 1 FROM custom_domains o
    WHERE o.id < c.id
      AND lower(rtrim(o.domain, '.')) = lower(rtrim(c.domain, '.'))
);
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE custom_domains
SET domain = lower(rtrim(domain, '.'))
WHERE domain <> lower(rtrim(domain, '.'));
-- +goose StatementEnd

-- The index covers the canonical form, not the stored spelling: "example.com",
-- "EXAMPLE.com" and "example.com." are one DNS name and must collide even if
-- something bypasses the handler's normalization.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_custom_domains_domain_lower
    ON custom_domains (lower(rtrim(domain, '.')));

-- +goose Down
DROP INDEX IF EXISTS uniq_custom_domains_domain_lower;
