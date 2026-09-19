package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mephistofox/fxtunnel/internal/server/database/sqlc"
)

// APITokenRepository handles API token database operations using PostgreSQL via sqlc.
type APITokenRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

// sqlcTokenToDomain converts a sqlc.ApiToken to a domain APIToken.
func sqlcTokenToDomain(t sqlc.ApiToken) *APIToken {
	return &APIToken{
		ID:                t.ID,
		UserID:            t.UserID,
		TokenHash:         t.TokenHash,
		Name:              t.Name,
		AllowedSubdomains: jsonToStringSlice(t.AllowedSubdomains),
		MaxTunnels:        int(t.MaxTunnels),
		AllowedIPs:        jsonToStringSlice(t.AllowedIps),
		LastUsedAt:        tsToTimePtr(t.LastUsedAt),
		CreatedAt:         tsToTime(t.CreatedAt),
	}
}

// Create creates a new API token.
func (r *APITokenRepository) Create(token *APIToken) error {
	ctx := context.Background()
	row, err := r.q.CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
		UserID:            token.UserID,
		TokenHash:         token.TokenHash,
		Name:              token.Name,
		AllowedSubdomains: stringSliceToJSON(token.AllowedSubdomains),
		MaxTunnels:        int32(token.MaxTunnels),
		AllowedIps:        stringSliceToJSON(token.AllowedIPs),
	})
	if err != nil {
		return fmt.Errorf("create api token: %w", err)
	}
	token.ID = row.ID
	token.CreatedAt = tsToTime(row.CreatedAt)
	return nil
}

// CreateWithLimit creates a token only if the user is still under maxTokens.
// The count and the insert run in one transaction behind a row lock on the user,
// so concurrent requests cannot each read the same pre-insert count.
// A negative maxTokens means unlimited.
func (r *APITokenRepository) CreateWithLimit(token *APIToken, maxTokens int) error {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create api token: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lockedID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, token.UserID).Scan(&lockedID); err != nil {
		if isNotFound(err) {
			return ErrUserNotFound
		}
		return fmt.Errorf("lock user for token create: %w", err)
	}

	if maxTokens >= 0 {
		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM api_tokens WHERE user_id = $1`, token.UserID).Scan(&count); err != nil {
			return fmt.Errorf("count api tokens: %w", err)
		}
		if count >= maxTokens {
			return ErrMaxTokensReached
		}
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO api_tokens (user_id, token_hash, name, allowed_subdomains, max_tunnels, allowed_ips, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING id, created_at`,
		token.UserID, token.TokenHash, token.Name,
		stringSliceToJSON(token.AllowedSubdomains), int32(token.MaxTunnels), stringSliceToJSON(token.AllowedIPs),
	).Scan(&token.ID, &token.CreatedAt)
	if err != nil {
		return fmt.Errorf("create api token: %w", err)
	}

	return tx.Commit(ctx)
}

// GetByID retrieves an API token by ID.
func (r *APITokenRepository) GetByID(id int64) (*APIToken, error) {
	ctx := context.Background()
	t, err := r.q.GetAPITokenByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("get api token by id: %w", err)
	}
	return sqlcTokenToDomain(t), nil
}

// GetByTokenHash retrieves an API token by token hash.
func (r *APITokenRepository) GetByTokenHash(tokenHash string) (*APIToken, error) {
	ctx := context.Background()
	t, err := r.q.GetAPITokenByHash(ctx, tokenHash)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("get api token by hash: %w", err)
	}
	return sqlcTokenToDomain(t), nil
}

// GetByUserID retrieves all API tokens for a user.
func (r *APITokenRepository) GetByUserID(userID int64) ([]*APIToken, error) {
	ctx := context.Background()
	rows, err := r.q.ListAPITokensByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get api tokens by user id: %w", err)
	}
	tokens := make([]*APIToken, 0, len(rows))
	for _, t := range rows {
		tokens = append(tokens, sqlcTokenToDomain(t))
	}
	return tokens, nil
}

// Delete deletes an API token by ID.
func (r *APITokenRepository) Delete(id int64) error {
	ctx := context.Background()
	err := r.q.DeleteAPIToken(ctx, id)
	if err != nil {
		return fmt.Errorf("delete api token: %w", err)
	}
	return nil
}

// DeleteByUserID deletes all API tokens for a user.
func (r *APITokenRepository) DeleteByUserID(userID int64) error {
	ctx := context.Background()
	err := r.q.DeleteAPITokensByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete api tokens by user id: %w", err)
	}
	return nil
}

// UpdateLastUsed updates the last used timestamp.
func (r *APITokenRepository) UpdateLastUsed(id int64) error {
	ctx := context.Background()
	err := r.q.UpdateAPITokenLastUsed(ctx, id)
	if err != nil {
		return fmt.Errorf("update last used: %w", err)
	}
	return nil
}

// Count returns the total number of tokens for a user.
func (r *APITokenRepository) Count(userID int64) (int, error) {
	ctx := context.Background()
	count, err := r.q.CountAPITokensByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count api tokens: %w", err)
	}
	return int(count), nil
}
