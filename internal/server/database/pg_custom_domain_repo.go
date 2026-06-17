package database

import (
	"context"
	"fmt"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/database/sqlc"
)

// CustomDomainRepository handles custom domain database operations using PostgreSQL via sqlc.
type CustomDomainRepository struct {
	q *sqlc.Queries
}

// sqlcCustomDomainToDomain converts a sqlc.CustomDomain to a domain CustomDomain.
func sqlcCustomDomainToDomain(d sqlc.CustomDomain) *CustomDomain {
	return &CustomDomain{
		ID:                d.ID,
		UserID:            d.UserID,
		Domain:            d.Domain,
		TargetSubdomain:   d.TargetSubdomain,
		VerificationToken: d.VerificationToken,
		Verified:          d.Verified,
		VerifiedAt:        tsToTimePtr(d.VerifiedAt),
		CreatedAt:         tsToTime(d.CreatedAt),
	}
}

// Create creates a new custom domain.
func (r *CustomDomainRepository) Create(d *CustomDomain) error {
	ctx := context.Background()
	row, err := r.q.CreateCustomDomain(ctx, sqlc.CreateCustomDomainParams{
		UserID:            d.UserID,
		Domain:            d.Domain,
		TargetSubdomain:   d.TargetSubdomain,
		VerificationToken: d.VerificationToken,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrCustomDomainAlreadyExists
		}
		return fmt.Errorf("create custom domain: %w", err)
	}
	d.ID = row.ID
	d.CreatedAt = tsToTime(row.CreatedAt)
	return nil
}

// GetByID retrieves a custom domain by ID.
func (r *CustomDomainRepository) GetByID(id int64) (*CustomDomain, error) {
	ctx := context.Background()
	d, err := r.q.GetCustomDomainByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrCustomDomainNotFound
		}
		return nil, fmt.Errorf("get custom domain by id: %w", err)
	}
	return sqlcCustomDomainToDomain(d), nil
}

// GetByDomain retrieves a custom domain by domain name.
func (r *CustomDomainRepository) GetByDomain(domain string) (*CustomDomain, error) {
	ctx := context.Background()
	d, err := r.q.GetCustomDomainByDomain(ctx, domain)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrCustomDomainNotFound
		}
		return nil, fmt.Errorf("get custom domain by domain: %w", err)
	}
	return sqlcCustomDomainToDomain(d), nil
}

// GetByUserID retrieves all custom domains for a user.
func (r *CustomDomainRepository) GetByUserID(userID int64) ([]*CustomDomain, error) {
	ctx := context.Background()
	rows, err := r.q.ListCustomDomainsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get custom domains by user id: %w", err)
	}
	domains := make([]*CustomDomain, 0, len(rows))
	for _, d := range rows {
		domains = append(domains, sqlcCustomDomainToDomain(d))
	}
	return domains, nil
}

// GetAll retrieves all custom domains with pagination.
func (r *CustomDomainRepository) GetAll(limit, offset int) ([]*CustomDomain, int, error) {
	ctx := context.Background()

	count, err := r.q.CountAllCustomDomains(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count custom domains: %w", err)
	}

	rows, err := r.q.ListAllCustomDomains(ctx, sqlc.ListAllCustomDomainsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list all custom domains: %w", err)
	}

	domains := make([]*CustomDomain, 0, len(rows))
	for _, d := range rows {
		domains = append(domains, sqlcCustomDomainToDomain(d))
	}
	return domains, int(count), nil
}

// GetAllVerified retrieves all verified custom domains.
func (r *CustomDomainRepository) GetAllVerified() ([]*CustomDomain, error) {
	ctx := context.Background()
	rows, err := r.q.ListVerifiedCustomDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("list verified custom domains: %w", err)
	}
	domains := make([]*CustomDomain, 0, len(rows))
	for _, d := range rows {
		domains = append(domains, sqlcCustomDomainToDomain(d))
	}
	return domains, nil
}

// CountByUserID returns the number of custom domains for a user.
func (r *CustomDomainRepository) CountByUserID(userID int64) (int, error) {
	ctx := context.Background()
	count, err := r.q.CountCustomDomainsByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count custom domains by user id: %w", err)
	}
	return int(count), nil
}

// SetVerified sets the verified status of a custom domain.
func (r *CustomDomainRepository) SetVerified(id int64, verified bool) error {
	ctx := context.Background()
	var verifiedAt time.Time
	if verified {
		verifiedAt = time.Now()
	}
	err := r.q.SetCustomDomainVerified(ctx, sqlc.SetCustomDomainVerifiedParams{
		ID:         id,
		Verified:   verified,
		VerifiedAt: timeToPgtz(verifiedAt),
	})
	if err != nil {
		return fmt.Errorf("set custom domain verified: %w", err)
	}
	return nil
}

// SetVerificationToken stores the ownership-proof token for a custom domain.
func (r *CustomDomainRepository) SetVerificationToken(id int64, token string) error {
	ctx := context.Background()
	err := r.q.SetCustomDomainVerificationToken(ctx, sqlc.SetCustomDomainVerificationTokenParams{
		ID:                id,
		VerificationToken: token,
	})
	if err != nil {
		return fmt.Errorf("set custom domain verification token: %w", err)
	}
	return nil
}

// Delete deletes a custom domain by ID.
func (r *CustomDomainRepository) Delete(id int64) error {
	ctx := context.Background()
	err := r.q.DeleteCustomDomain(ctx, id)
	if err != nil {
		return fmt.Errorf("delete custom domain: %w", err)
	}
	return nil
}
