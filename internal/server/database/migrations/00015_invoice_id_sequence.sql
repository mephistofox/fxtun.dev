-- +goose Up
-- Invoice numbers came from SELECT MAX(invoice_id) + 1, so two checkouts
-- running at the same time got the same number and the second one failed on
-- the UNIQUE constraint — the user saw a broken payment. A sequence hands out
-- distinct numbers without coordination.
CREATE SEQUENCE IF NOT EXISTS payments_invoice_id_seq AS bigint START WITH 100001;

-- Move the sequence past whatever is already stored, so existing rows never
-- collide with freshly issued numbers.
SELECT setval('payments_invoice_id_seq', GREATEST((SELECT COALESCE(MAX(invoice_id), 100000) FROM payments), 100000));

-- +goose Down
DROP SEQUENCE IF EXISTS payments_invoice_id_seq;
