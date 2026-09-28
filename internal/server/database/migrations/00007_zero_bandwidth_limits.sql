-- +goose Up
-- Zero bandwidth_mbps on all plans. The column is not enforced by code
-- (no throttling implemented) and the landing page advertises "no bandwidth
-- limits on any plan". Keeping non-zero numbers here is a footgun: any future
-- throttling implementation would silently break the marketing promise.
UPDATE plans SET bandwidth_mbps = 0;

-- +goose Down
-- Restoring per-plan defaults is intentionally not provided — the previous
-- numbers were vestigial and never enforced.
SELECT 1;
