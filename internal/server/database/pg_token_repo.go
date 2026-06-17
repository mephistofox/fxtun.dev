package database

import (
	"context"
	"fmt"

	"github.com/mephistofox/fxtun.dev/internal/server/database/sqlc"
)

// APITokenRepository handles API token database operations using PostgreSQL via sqlc.
type APITokenRepository struct {
	q *sqlc.Queries
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
