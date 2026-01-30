package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrTOTPNotFound = errors.New("totp secret not found")

// TOTPRepository handles TOTP secret database operations
type TOTPRepository struct {
	db *sql.DB
}

// NewTOTPRepository creates a new TOTP repository
func NewTOTPRepository(db *sql.DB) *TOTPRepository {
	return &TOTPRepository{db: db}
}

// Create creates a new TOTP secret
func (r *TOTPRepository) Create(totp *TOTPSecret) error {
	backupCodes, err := json.Marshal(totp.BackupCodes)
	if err != nil {
		return fmt.Errorf("marshal backup codes: %w", err)
	}

	query := `
		INSERT INTO totp_secrets (user_id, secret_encrypted, is_enabled, backup_codes, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		totp.UserID,
		totp.SecretEncrypted,
		totp.IsEnabled,
		string(backupCodes),
		now,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("totp secret already exists for user")
		}
		return fmt.Errorf("create totp secret: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	totp.ID = id
	totp.CreatedAt = now
	return nil
}

// GetByUserID retrieves a TOTP secret by user ID
func (r *TOTPRepository) GetByUserID(userID int64) (*TOTPSecret, error) {
	query := `
		SELECT id, user_id, secret_encrypted, is_enabled, backup_codes, created_at
		FROM totp_secrets WHERE user_id = ?
	`

	totp := &TOTPSecret{}
	var backupCodes sql.NullString

	err := r.db.QueryRow(query, userID).Scan(
		&totp.ID,
		&totp.UserID,
		&totp.SecretEncrypted,
		&totp.IsEnabled,
		&backupCodes,
		&totp.CreatedAt,
	)
	if err != nil {
		return nil, notFoundOrError(err, ErrTOTPNotFound, "get totp secret by user id")
	}

	if backupCodes.Valid && backupCodes.String != "" {
		if err := json.Unmarshal([]byte(backupCodes.String), &totp.BackupCodes); err != nil {
			totp.BackupCodes = []string{}
		}
	}

	return totp, nil
}

// Update updates a TOTP secret
func (r *TOTPRepository) Update(totp *TOTPSecret) error {
	backupCodes, err := json.Marshal(totp.BackupCodes)
	if err != nil {
		return fmt.Errorf("marshal backup codes: %w", err)
	}

	query := `
		UPDATE totp_secrets
		SET secret_encrypted = ?, is_enabled = ?, backup_codes = ?
		WHERE id = ?
	`

	result, err := r.db.Exec(query,
		totp.SecretEncrypted,
		totp.IsEnabled,
		string(backupCodes),
		totp.ID,
	)
	if err != nil {
		return fmt.Errorf("update totp secret: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrTOTPNotFound
	}

	return nil
}

// Enable enables TOTP for a user
func (r *TOTPRepository) Enable(userID int64) error {
	query := `UPDATE totp_secrets SET is_enabled = TRUE WHERE user_id = ?`

	result, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("enable totp: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrTOTPNotFound
	}

	return nil
}

// Disable disables TOTP for a user
func (r *TOTPRepository) Disable(userID int64) error {
	query := `UPDATE totp_secrets SET is_enabled = FALSE WHERE user_id = ?`

	result, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("disable totp: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrTOTPNotFound
	}

	return nil
}

// Delete deletes a TOTP secret by user ID
func (r *TOTPRepository) Delete(userID int64) error {
	query := `DELETE FROM totp_secrets WHERE user_id = ?`

	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("delete totp secret: %w", err)
	}

	return nil
}

// IsEnabled checks if TOTP is enabled for a user
func (r *TOTPRepository) IsEnabled(userID int64) (bool, error) {
	query := `SELECT is_enabled FROM totp_secrets WHERE user_id = ?`

	var isEnabled bool
	err := r.db.QueryRow(query, userID).Scan(&isEnabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check totp enabled: %w", err)
	}

	return isEnabled, nil
}

// UpdateBackupCodes updates backup codes for a user
func (r *TOTPRepository) UpdateBackupCodes(userID int64, codes []string) error {
	backupCodes, err := json.Marshal(codes)
	if err != nil {
		return fmt.Errorf("marshal backup codes: %w", err)
	}

	query := `UPDATE totp_secrets SET backup_codes = ? WHERE user_id = ?`

	_, err = r.db.Exec(query, string(backupCodes), userID)
	if err != nil {
		return fmt.Errorf("update backup codes: %w", err)
	}

	return nil
}
