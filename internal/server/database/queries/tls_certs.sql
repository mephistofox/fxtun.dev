-- name: UpsertTLSCertificate :one
INSERT INTO tls_certificates (domain, cert_pem, key_pem, expires_at, issued_at, created_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (domain) DO UPDATE SET
    cert_pem = EXCLUDED.cert_pem,
    key_pem = EXCLUDED.key_pem,
    expires_at = EXCLUDED.expires_at,
    issued_at = EXCLUDED.issued_at
RETURNING id;

-- name: GetTLSCertByDomain :one
SELECT id, domain, cert_pem, key_pem, expires_at, issued_at, created_at
FROM tls_certificates WHERE domain = $1;

-- name: ListExpiringTLSCerts :many
SELECT id, domain, cert_pem, key_pem, expires_at, issued_at, created_at
FROM tls_certificates WHERE expires_at < $1;

-- name: DeleteTLSCertByDomain :exec
DELETE FROM tls_certificates WHERE domain = $1;
