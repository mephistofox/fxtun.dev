-- name: CreateTOTPSecret :one
INSERT INTO totp_secrets (user_id, secret_encrypted, is_enabled, backup_codes, created_at)
VALUES ($1, $2, $3, $4, NOW())
RETURNING id, created_at;

-- name: GetTOTPByUserID :one
SELECT id, user_id, secret_encrypted, is_enabled, backup_codes, created_at
FROM totp_secrets WHERE user_id = $1;

-- name: UpdateTOTPSecret :exec
UPDATE totp_secrets SET secret_encrypted = $2, is_enabled = $3, backup_codes = $4 WHERE id = $1;

-- name: EnableTOTP :exec
UPDATE totp_secrets SET is_enabled = TRUE WHERE user_id = $1;

-- name: DisableTOTP :exec
UPDATE totp_secrets SET is_enabled = FALSE WHERE user_id = $1;

-- name: DeleteTOTP :exec
DELETE FROM totp_secrets WHERE user_id = $1;

-- name: IsTOTPEnabled :one
SELECT is_enabled FROM totp_secrets WHERE user_id = $1;

-- name: UpdateTOTPBackupCodes :exec
UPDATE totp_secrets SET backup_codes = $2 WHERE user_id = $1;
