package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InviteCodeRepository handles invite code database operations using raw SQL (no sqlc queries exist yet).
type InviteCodeRepository struct {
	pool *pgxpool.Pool
}

// List returns invite codes with pagination, ordered by creation date descending.
// Returns codes, total count, and error.
func (r *InviteCodeRepository) List(limit, offset int) ([]*InviteCode, int, error) {
	ctx := context.Background()

	if limit <= 0 {
		limit = 100
	}

	// Get total count
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM invite_codes`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count invite codes: %w", err)
	}

	query := `SELECT id, code, created_by_user_id, used_by_user_id, used_at, expires_at, created_at
		FROM invite_codes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list invite codes: %w", err)
	}
	defer rows.Close()

	var codes []*InviteCode
	for rows.Next() {
		c := &InviteCode{}
		if err := rows.Scan(&c.ID, &c.Code, &c.CreatedByUserID, &c.UsedByUserID, &c.UsedAt, &c.ExpiresAt, &c.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan invite code: %w", err)
		}
		codes = append(codes, c)
	}
	return codes, total, rows.Err()
}

// Create creates a new invite code with the given code string and creator user ID.
func (r *InviteCodeRepository) Create(code string, createdByUserID int64) (*InviteCode, error) {
	ctx := context.Background()
	query := `INSERT INTO invite_codes (code, created_by_user_id) VALUES ($1, $2) RETURNING id, created_at`

	c := &InviteCode{
		Code:            code,
		CreatedByUserID: &createdByUserID,
	}
	err := r.pool.QueryRow(ctx, query, code, createdByUserID).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("invite code already exists")
		}
		return nil, fmt.Errorf("create invite code: %w", err)
	}
	return c, nil
}

// Delete removes an invite code by ID.
func (r *InviteCodeRepository) Delete(id int64) error {
	ctx := context.Background()
	query := `DELETE FROM invite_codes WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete invite code: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInviteCodeNotFound
	}
	return nil
}
