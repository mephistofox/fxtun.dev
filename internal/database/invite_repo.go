package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInviteNotFound      = errors.New("invite code not found")
	ErrInviteAlreadyUsed   = errors.New("invite code already used")
	ErrInviteExpired       = errors.New("invite code expired")
	ErrInviteAlreadyExists = errors.New("invite code already exists")
)

// InviteRepository handles invite code database operations
type InviteRepository struct {
	db *sql.DB
}

// NewInviteRepository creates a new invite repository
func NewInviteRepository(db *sql.DB) *InviteRepository {
	return &InviteRepository{db: db}
}

// Create creates a new invite code
func (r *InviteRepository) Create(invite *InviteCode) error {
	query := `
		INSERT INTO invite_codes (code, created_by_user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		invite.Code,
		invite.CreatedByUserID,
		invite.ExpiresAt,
		now,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrInviteAlreadyExists
		}
		return fmt.Errorf("create invite code: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	invite.ID = id
	invite.CreatedAt = now
	return nil
}

// GetByID retrieves an invite code by ID
func (r *InviteRepository) GetByID(id int64) (*InviteCode, error) {
	query := `
		SELECT id, code, created_by_user_id, used_by_user_id, used_at, expires_at, created_at
		FROM invite_codes WHERE id = ?
	`

	return r.scanInvite(r.db.QueryRow(query, id))
}

// GetByCode retrieves an invite code by code string
func (r *InviteRepository) GetByCode(code string) (*InviteCode, error) {
	query := `
		SELECT id, code, created_by_user_id, used_by_user_id, used_at, expires_at, created_at
		FROM invite_codes WHERE code = ?
	`

	return r.scanInvite(r.db.QueryRow(query, code))
}

// scanInvite scans a single invite code row
func (r *InviteRepository) scanInvite(row *sql.Row) (*InviteCode, error) {
	invite := &InviteCode{}
	var createdByUserID, usedByUserID sql.NullInt64
	var usedAt, expiresAt sql.NullTime

	err := row.Scan(
		&invite.ID,
		&invite.Code,
		&createdByUserID,
		&usedByUserID,
		&usedAt,
		&expiresAt,
		&invite.CreatedAt,
	)
	if err != nil {
		return nil, notFoundOrError(err, ErrInviteNotFound, "scan invite code")
	}

	if createdByUserID.Valid {
		invite.CreatedByUserID = &createdByUserID.Int64
	}
	if usedByUserID.Valid {
		invite.UsedByUserID = &usedByUserID.Int64
	}
	if usedAt.Valid {
		invite.UsedAt = &usedAt.Time
	}
	if expiresAt.Valid {
		invite.ExpiresAt = &expiresAt.Time
	}

	return invite, nil
}

// UseTx marks an invite code as used within a transaction
func (r *InviteRepository) UseTx(tx *sql.Tx, code string, userID int64) error {
	query := `UPDATE invite_codes SET used_by_user_id = ?, used_at = ? WHERE code = ? AND used_by_user_id IS NULL`
	now := time.Now()
	result, err := tx.Exec(query, userID, now, code)
	if err != nil {
		return fmt.Errorf("use invite code: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrInviteAlreadyUsed
	}
	return nil
}

// Use marks an invite code as used
func (r *InviteRepository) Use(code string, userID int64) error {
	// First, get the invite to check validity
	invite, err := r.GetByCode(code)
	if err != nil {
		return err
	}

	if invite.IsUsed() {
		return ErrInviteAlreadyUsed
	}

	if invite.IsExpired() {
		return ErrInviteExpired
	}

	// Mark as used
	query := `
		UPDATE invite_codes
		SET used_by_user_id = ?, used_at = ?
		WHERE id = ? AND used_by_user_id IS NULL
	`

	now := time.Now()
	result, err := r.db.Exec(query, userID, now, invite.ID)
	if err != nil {
		return fmt.Errorf("use invite code: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrInviteAlreadyUsed
	}

	return nil
}

// Delete deletes an invite code by ID
func (r *InviteRepository) Delete(id int64) error {
	query := `DELETE FROM invite_codes WHERE id = ?`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete invite code: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrInviteNotFound
	}

	return nil
}

// List returns all invite codes with pagination
func (r *InviteRepository) List(limit, offset int) ([]*InviteCode, int, error) {
	countQuery := `SELECT COUNT(*) FROM invite_codes`
	var total int
	if err := r.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invite codes: %w", err)
	}

	query := `
		SELECT id, code, created_by_user_id, used_by_user_id, used_at, expires_at, created_at
		FROM invite_codes ORDER BY created_at DESC LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list invite codes: %w", err)
	}
	defer rows.Close()

	var invites []*InviteCode
	for rows.Next() {
		invite := &InviteCode{}
		var createdByUserID, usedByUserID sql.NullInt64
		var usedAt, expiresAt sql.NullTime

		if err := rows.Scan(
			&invite.ID,
			&invite.Code,
			&createdByUserID,
			&usedByUserID,
			&usedAt,
			&expiresAt,
			&invite.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan invite code: %w", err)
		}

		if createdByUserID.Valid {
			invite.CreatedByUserID = &createdByUserID.Int64
		}
		if usedByUserID.Valid {
			invite.UsedByUserID = &usedByUserID.Int64
		}
		if usedAt.Valid {
			invite.UsedAt = &usedAt.Time
		}
		if expiresAt.Valid {
			invite.ExpiresAt = &expiresAt.Time
		}

		invites = append(invites, invite)
	}

	return invites, total, nil
}

// ListUnused returns all unused invite codes
func (r *InviteRepository) ListUnused(limit, offset int) ([]*InviteCode, int, error) {
	countQuery := `SELECT COUNT(*) FROM invite_codes WHERE used_by_user_id IS NULL`
	var total int
	if err := r.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count unused invite codes: %w", err)
	}

	query := `
		SELECT id, code, created_by_user_id, used_by_user_id, used_at, expires_at, created_at
		FROM invite_codes WHERE used_by_user_id IS NULL
		ORDER BY created_at DESC LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list unused invite codes: %w", err)
	}
	defer rows.Close()

	var invites []*InviteCode
	for rows.Next() {
		invite := &InviteCode{}
		var createdByUserID, usedByUserID sql.NullInt64
		var usedAt, expiresAt sql.NullTime

		if err := rows.Scan(
			&invite.ID,
			&invite.Code,
			&createdByUserID,
			&usedByUserID,
			&usedAt,
			&expiresAt,
			&invite.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan invite code: %w", err)
		}

		if createdByUserID.Valid {
			invite.CreatedByUserID = &createdByUserID.Int64
		}
		if expiresAt.Valid {
			invite.ExpiresAt = &expiresAt.Time
		}

		invites = append(invites, invite)
	}

	return invites, total, nil
}

// DeleteExpired deletes all expired unused invite codes
func (r *InviteRepository) DeleteExpired() (int64, error) {
	query := `DELETE FROM invite_codes WHERE expires_at IS NOT NULL AND expires_at < ? AND used_by_user_id IS NULL`

	result, err := r.db.Exec(query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("delete expired invite codes: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return rows, nil
}

// IsValid checks if an invite code is valid (exists, not used, not expired)
func (r *InviteRepository) IsValid(code string) (bool, error) {
	invite, err := r.GetByCode(code)
	if err != nil {
		if errors.Is(err, ErrInviteNotFound) {
			return false, nil
		}
		return false, err
	}

	return !invite.IsUsed() && !invite.IsExpired(), nil
}
