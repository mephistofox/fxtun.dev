package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mephistofox/fxtun.dev/internal/server/database/sqlc"
)

// TOTPRepository handles TOTP secret database operations using PostgreSQL via sqlc.
type TOTPRepository struct {
	q *sqlc.Queries
}

// sqlcTOTPToDomain converts a sqlc.TotpSecret to a domain TOTPSecret.
func sqlcTOTPToDomain(t sqlc.TotpSecret) *TOTPSecret {
	var codes []string
	if len(t.BackupCodes) > 0 {
		_ = json.Unmarshal(t.BackupCodes, &codes)
	}
	return &TOTPSecret{
		ID:              t.ID,
		UserID:          t.UserID,
		SecretEncrypted: t.SecretEncrypted,
		IsEnabled:       t.IsEnabled,
		BackupCodes:     codes,
		CreatedAt:       tsToTime(t.CreatedAt),
	}
}

// Create creates a new TOTP secret.
func (r *TOTPRepository) Create(totp *TOTPSecret) error {
	ctx := context.Background()
	backupCodes, err := json.Marshal(totp.BackupCodes)
	if err != nil {
		return fmt.Errorf("marshal backup codes: %w", err)
	}
	row, err := r.q.CreateTOTPSecret(ctx, sqlc.CreateTOTPSecretParams{
		UserID:          totp.UserID,
		SecretEncrypted: totp.SecretEncrypted,
		IsEnabled:       totp.IsEnabled,
		BackupCodes:     backupCodes,
	})
	if err != nil {
		return fmt.Errorf("create totp secret: %w", err)
	}
	totp.ID = row.ID
	totp.CreatedAt = tsToTime(row.CreatedAt)
	return nil
}

// GetByUserID retrieves a TOTP secret by user ID.
func (r *TOTPRepository) GetByUserID(userID int64) (*TOTPSecret, error) {
	ctx := context.Background()
	t, err := r.q.GetTOTPByUserID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrTOTPNotFound
		}
		return nil, fmt.Errorf("get totp by user id: %w", err)
	}
	return sqlcTOTPToDomain(t), nil
}

// Update updates an existing TOTP secret.
func (r *TOTPRepository) Update(totp *TOTPSecret) error {
	ctx := context.Background()
	backupCodes, err := json.Marshal(totp.BackupCodes)
	if err != nil {
		return fmt.Errorf("marshal backup codes: %w", err)
	}
	err = r.q.UpdateTOTPSecret(ctx, sqlc.UpdateTOTPSecretParams{
		ID:              totp.ID,
		SecretEncrypted: totp.SecretEncrypted,
		IsEnabled:       totp.IsEnabled,
		BackupCodes:     backupCodes,
	})
	if err != nil {
		return fmt.Errorf("update totp secret: %w", err)
	}
	return nil
}

// Enable enables TOTP for a user.
func (r *TOTPRepository) Enable(userID int64) error {
	ctx := context.Background()
	err := r.q.EnableTOTP(ctx, userID)
	if err != nil {
		return fmt.Errorf("enable totp: %w", err)
	}
	return nil
}

// Disable disables TOTP for a user.
func (r *TOTPRepository) Disable(userID int64) error {
	ctx := context.Background()
	err := r.q.DisableTOTP(ctx, userID)
	if err != nil {
		return fmt.Errorf("disable totp: %w", err)
	}
	return nil
}

// Delete deletes a TOTP secret by user ID.
func (r *TOTPRepository) Delete(userID int64) error {
	ctx := context.Background()
	err := r.q.DeleteTOTP(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete totp: %w", err)
	}
	return nil
}

// IsEnabled checks if TOTP is enabled for a user.
func (r *TOTPRepository) IsEnabled(userID int64) (bool, error) {
	ctx := context.Background()
	enabled, err := r.q.IsTOTPEnabled(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("check totp enabled: %w", err)
	}
	return enabled, nil
}

// UpdateBackupCodes updates backup codes for a user's TOTP secret.
func (r *TOTPRepository) UpdateBackupCodes(userID int64, codes []string) error {
	ctx := context.Background()
	backupCodes, err := json.Marshal(codes)
	if err != nil {
		return fmt.Errorf("marshal backup codes: %w", err)
	}
	err = r.q.UpdateTOTPBackupCodes(ctx, sqlc.UpdateTOTPBackupCodesParams{
		UserID:      userID,
		BackupCodes: backupCodes,
	})
	if err != nil {
		return fmt.Errorf("update backup codes: %w", err)
	}
	return nil
}
