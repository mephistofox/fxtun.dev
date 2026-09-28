package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mephistofox/fxtun.dev/internal/server/database/sqlc"
)

// DomainRepository handles reserved domain database operations using PostgreSQL via sqlc.
type DomainRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

// sqlcDomainToDomain converts a sqlc.ReservedDomain to a domain ReservedDomain.
func sqlcDomainToDomain(d sqlc.ReservedDomain) *ReservedDomain {
	return &ReservedDomain{
		ID:        d.ID,
		UserID:    d.UserID,
		Subdomain: d.Subdomain,
		CreatedAt: tsToTime(d.CreatedAt),
	}
}

// CreateWithLimit reserves a subdomain only if the user is still under maxDomains.
// The count and the insert run in one transaction behind a row lock on the user,
// so concurrent requests cannot each read the same pre-insert count.
// A negative maxDomains means unlimited.
func (r *DomainRepository) CreateWithLimit(domain *ReservedDomain, maxDomains int) error {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reserve domain: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lockedID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, domain.UserID).Scan(&lockedID); err != nil {
		if isNotFound(err) {
			return ErrUserNotFound
		}
		return fmt.Errorf("lock user for domain reserve: %w", err)
	}

	if maxDomains >= 0 {
		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM reserved_domains WHERE user_id = $1`, domain.UserID).Scan(&count); err != nil {
			return fmt.Errorf("count reserved domains: %w", err)
		}
		if count >= maxDomains {
			return ErrMaxDomainsReached
		}
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO reserved_domains (user_id, subdomain, created_at) VALUES ($1, $2, NOW()) RETURNING id, created_at`,
		domain.UserID, domain.Subdomain).Scan(&domain.ID, &domain.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDomainAlreadyExists
		}
		return fmt.Errorf("create reserved domain: %w", err)
	}

	return tx.Commit(ctx)
}

// GetByID retrieves a reserved domain by ID.
func (r *DomainRepository) GetByID(id int64) (*ReservedDomain, error) {
	ctx := context.Background()
	d, err := r.q.GetReservedDomainByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrDomainNotFound
		}
		return nil, fmt.Errorf("get reserved domain by id: %w", err)
	}
	return sqlcDomainToDomain(d), nil
}

// GetBySubdomain retrieves a reserved domain by subdomain name.
func (r *DomainRepository) GetBySubdomain(subdomain string) (*ReservedDomain, error) {
	ctx := context.Background()
	d, err := r.q.GetReservedDomainBySubdomain(ctx, subdomain)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrDomainNotFound
		}
		return nil, fmt.Errorf("get reserved domain by subdomain: %w", err)
	}
	return sqlcDomainToDomain(d), nil
}

// GetByUserID retrieves all reserved domains for a user.
func (r *DomainRepository) GetByUserID(userID int64) ([]*ReservedDomain, error) {
	ctx := context.Background()
	rows, err := r.q.ListReservedDomainsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get reserved domains by user id: %w", err)
	}
	domains := make([]*ReservedDomain, 0, len(rows))
	for _, d := range rows {
		domains = append(domains, sqlcDomainToDomain(d))
	}
	return domains, nil
}

// Delete deletes a reserved domain by ID.
func (r *DomainRepository) Delete(id int64) error {
	ctx := context.Background()
	err := r.q.DeleteReservedDomain(ctx, id)
	if err != nil {
		return fmt.Errorf("delete reserved domain: %w", err)
	}
	return nil
}

// DeleteByUserID deletes all reserved domains for a user.
func (r *DomainRepository) DeleteByUserID(userID int64) error {
	ctx := context.Background()
	err := r.q.DeleteReservedDomainsByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete reserved domains by user id: %w", err)
	}
	return nil
}

// IsAvailable checks if a subdomain is available (not reserved).
func (r *DomainRepository) IsAvailable(subdomain string) (bool, error) {
	ctx := context.Background()
	available, err := r.q.IsSubdomainAvailable(ctx, subdomain)
	if err != nil {
		return false, fmt.Errorf("check subdomain availability: %w", err)
	}
	return available, nil
}

// IsOwnedByUser checks if a subdomain is reserved by a specific user.
func (r *DomainRepository) IsOwnedByUser(subdomain string, userID int64) (bool, error) {
	ctx := context.Background()
	owned, err := r.q.IsSubdomainOwnedByUser(ctx, sqlc.IsSubdomainOwnedByUserParams{
		Subdomain: subdomain,
		UserID:    userID,
	})
	if err != nil {
		return false, fmt.Errorf("check subdomain ownership: %w", err)
	}
	return owned, nil
}
