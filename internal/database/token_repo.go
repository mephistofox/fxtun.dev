package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrTokenNotFound = errors.New("token not found")

// APITokenRepository handles API token database operations
type APITokenRepository struct {
	db *sql.DB
}

// NewAPITokenRepository creates a new API token repository
func NewAPITokenRepository(db *sql.DB) *APITokenRepository {
	return &APITokenRepository{db: db}
}

// Create creates a new API token
func (r *APITokenRepository) Create(token *APIToken) error {
	allowedSubdomains, err := json.Marshal(token.AllowedSubdomains)
	if err != nil {
		return fmt.Errorf("marshal allowed subdomains: %w", err)
	}

	query := `
		INSERT INTO api_tokens (user_id, token_hash, name, allowed_subdomains, max_tunnels, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		token.UserID,
		token.TokenHash,
		token.Name,
		string(allowedSubdomains),
		token.MaxTunnels,
		now,
	)
	if err != nil {
		return fmt.Errorf("create api token: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	token.ID = id
	token.CreatedAt = now
	return nil
}

// GetByID retrieves an API token by ID
func (r *APITokenRepository) GetByID(id int64) (*APIToken, error) {
	query := `
		SELECT id, user_id, token_hash, name, allowed_subdomains, max_tunnels, last_used_at, created_at
		FROM api_tokens WHERE id = ?
	`

	token := &APIToken{}
	var allowedSubdomains string
	var lastUsedAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.Name,
		&allowedSubdomains,
		&token.MaxTunnels,
		&lastUsedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("get api token by id: %w", err)
	}

	if err := json.Unmarshal([]byte(allowedSubdomains), &token.AllowedSubdomains); err != nil {
		token.AllowedSubdomains = []string{}
	}

	if lastUsedAt.Valid {
		token.LastUsedAt = &lastUsedAt.Time
	}

	return token, nil
}

// GetByTokenHash retrieves an API token by token hash
func (r *APITokenRepository) GetByTokenHash(tokenHash string) (*APIToken, error) {
	query := `
		SELECT id, user_id, token_hash, name, allowed_subdomains, max_tunnels, last_used_at, created_at
		FROM api_tokens WHERE token_hash = ?
	`

	token := &APIToken{}
	var allowedSubdomains string
	var lastUsedAt sql.NullTime

	err := r.db.QueryRow(query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.Name,
		&allowedSubdomains,
		&token.MaxTunnels,
		&lastUsedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("get api token by hash: %w", err)
	}

	if err := json.Unmarshal([]byte(allowedSubdomains), &token.AllowedSubdomains); err != nil {
		token.AllowedSubdomains = []string{}
	}

	if lastUsedAt.Valid {
		token.LastUsedAt = &lastUsedAt.Time
	}

	return token, nil
}

// GetByUserID retrieves all API tokens for a user
func (r *APITokenRepository) GetByUserID(userID int64) ([]*APIToken, error) {
	query := `
		SELECT id, user_id, token_hash, name, allowed_subdomains, max_tunnels, last_used_at, created_at
		FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("get api tokens by user id: %w", err)
	}
	defer rows.Close()

	var tokens []*APIToken
	for rows.Next() {
		token := &APIToken{}
		var allowedSubdomains string
		var lastUsedAt sql.NullTime

		if err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.TokenHash,
			&token.Name,
			&allowedSubdomains,
			&token.MaxTunnels,
			&lastUsedAt,
			&token.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan api token: %w", err)
		}

		if err := json.Unmarshal([]byte(allowedSubdomains), &token.AllowedSubdomains); err != nil {
			token.AllowedSubdomains = []string{}
		}

		if lastUsedAt.Valid {
			token.LastUsedAt = &lastUsedAt.Time
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
}

// Delete deletes an API token by ID
func (r *APITokenRepository) Delete(id int64) error {
	query := `DELETE FROM api_tokens WHERE id = ?`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete api token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrTokenNotFound
	}

	return nil
}

// DeleteByUserID deletes all API tokens for a user
func (r *APITokenRepository) DeleteByUserID(userID int64) error {
	query := `DELETE FROM api_tokens WHERE user_id = ?`

	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("delete api tokens by user id: %w", err)
	}

	return nil
}

// UpdateLastUsed updates the last used timestamp
func (r *APITokenRepository) UpdateLastUsed(id int64) error {
	query := `UPDATE api_tokens SET last_used_at = ? WHERE id = ?`

	_, err := r.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update last used: %w", err)
	}

	return nil
}

// Count returns the total number of tokens for a user
func (r *APITokenRepository) Count(userID int64) (int, error) {
	query := `SELECT COUNT(*) FROM api_tokens WHERE user_id = ?`
	var count int
	if err := r.db.QueryRow(query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count api tokens: %w", err)
	}
	return count, nil
}
