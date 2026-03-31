package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrCustomDomainNotFound      = errors.New("custom domain not found")
	ErrCustomDomainAlreadyExists = errors.New("custom domain already exists")
)

// CustomDomainRepository provides access to custom_domains table.
type CustomDomainRepository struct {
	db *sql.DB
}

// NewCustomDomainRepository creates a new CustomDomainRepository.
func NewCustomDomainRepository(db *sql.DB) *CustomDomainRepository {
	return &CustomDomainRepository{db: db}
}

// Create inserts a new custom domain.
func (r *CustomDomainRepository) Create(d *CustomDomain) error {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO custom_domains (user_id, domain, target_subdomain, verified, created_at) VALUES (?, ?, ?, ?, ?)`,
		d.UserID, d.Domain, d.TargetSubdomain, d.Verified, now,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrCustomDomainAlreadyExists
		}
		return fmt.Errorf("create custom domain: %w", err)
	}
	id, _ := result.LastInsertId()
	d.ID = id
	d.CreatedAt = now
	return nil
}

// GetByID retrieves a custom domain by ID.
func (r *CustomDomainRepository) GetByID(id int64) (*CustomDomain, error) {
	d := &CustomDomain{}
	err := r.db.QueryRow(
		`SELECT id, user_id, domain, target_subdomain, verified, verified_at, created_at FROM custom_domains WHERE id = ?`, id,
	).Scan(&d.ID, &d.UserID, &d.Domain, &d.TargetSubdomain, &d.Verified, &d.VerifiedAt, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCustomDomainNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get custom domain by id: %w", err)
	}
	return d, nil
}

// GetByDomain retrieves a custom domain by domain name.
func (r *CustomDomainRepository) GetByDomain(domain string) (*CustomDomain, error) {
	d := &CustomDomain{}
	err := r.db.QueryRow(
		`SELECT id, user_id, domain, target_subdomain, verified, verified_at, created_at FROM custom_domains WHERE domain = ?`, domain,
	).Scan(&d.ID, &d.UserID, &d.Domain, &d.TargetSubdomain, &d.Verified, &d.VerifiedAt, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCustomDomainNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get custom domain by domain: %w", err)
	}
	return d, nil
}

// GetByUserID retrieves all custom domains for a user.
func (r *CustomDomainRepository) GetByUserID(userID int64) ([]*CustomDomain, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, domain, target_subdomain, verified, verified_at, created_at FROM custom_domains WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get custom domains by user id: %w", err)
	}
	defer rows.Close()

	var domains []*CustomDomain
	for rows.Next() {
		d := &CustomDomain{}
		if err := rows.Scan(&d.ID, &d.UserID, &d.Domain, &d.TargetSubdomain, &d.Verified, &d.VerifiedAt, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan custom domain: %w", err)
		}
		domains = append(domains, d)
	}
	return domains, nil
}

// GetAll retrieves all custom domains with pagination.
func (r *CustomDomainRepository) GetAll(limit, offset int) ([]*CustomDomain, int, error) {
	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM custom_domains`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count custom domains: %w", err)
	}

	rows, err := r.db.Query(
		`SELECT id, user_id, domain, target_subdomain, verified, verified_at, created_at FROM custom_domains ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list custom domains: %w", err)
	}
	defer rows.Close()

	var domains []*CustomDomain
	for rows.Next() {
		d := &CustomDomain{}
		if err := rows.Scan(&d.ID, &d.UserID, &d.Domain, &d.TargetSubdomain, &d.Verified, &d.VerifiedAt, &d.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan custom domain: %w", err)
		}
		domains = append(domains, d)
	}
	return domains, total, nil
}

// GetAllVerified retrieves all verified custom domains.
func (r *CustomDomainRepository) GetAllVerified() ([]*CustomDomain, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, domain, target_subdomain, verified, verified_at, created_at FROM custom_domains WHERE verified = TRUE`,
	)
	if err != nil {
		return nil, fmt.Errorf("get verified custom domains: %w", err)
	}
	defer rows.Close()

	var domains []*CustomDomain
	for rows.Next() {
		d := &CustomDomain{}
		if err := rows.Scan(&d.ID, &d.UserID, &d.Domain, &d.TargetSubdomain, &d.Verified, &d.VerifiedAt, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan custom domain: %w", err)
		}
		domains = append(domains, d)
	}
	return domains, nil
}

// CountByUserID returns the number of custom domains for a user.
func (r *CustomDomainRepository) CountByUserID(userID int64) (int, error) {
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM custom_domains WHERE user_id = ?`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count custom domains: %w", err)
	}
	return count, nil
}

// SetVerified updates the verified status of a custom domain.
func (r *CustomDomainRepository) SetVerified(id int64, verified bool) error {
	var verifiedAt interface{}
	if verified {
		now := time.Now()
		verifiedAt = now
	}
	_, err := r.db.Exec(`UPDATE custom_domains SET verified = ?, verified_at = ? WHERE id = ?`, verified, verifiedAt, id)
	if err != nil {
		return fmt.Errorf("set custom domain verified: %w", err)
	}
	return nil
}

// Delete removes a custom domain by ID.
func (r *CustomDomainRepository) Delete(id int64) error {
	result, err := r.db.Exec(`DELETE FROM custom_domains WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete custom domain: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrCustomDomainNotFound
	}
	return nil
}
