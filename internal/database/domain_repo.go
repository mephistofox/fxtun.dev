package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrDomainNotFound      = errors.New("domain not found")
	ErrDomainAlreadyExists = errors.New("domain already reserved")
	ErrMaxDomainsReached   = errors.New("maximum domains reached")
)

// DomainRepository handles reserved domain database operations
type DomainRepository struct {
	db *sql.DB
}

// NewDomainRepository creates a new domain repository
func NewDomainRepository(db *sql.DB) *DomainRepository {
	return &DomainRepository{db: db}
}

// Create creates a new reserved domain
func (r *DomainRepository) Create(domain *ReservedDomain) error {
	query := `
		INSERT INTO reserved_domains (user_id, subdomain, created_at)
		VALUES (?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		domain.UserID,
		domain.Subdomain,
		now,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDomainAlreadyExists
		}
		return fmt.Errorf("create reserved domain: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	domain.ID = id
	domain.CreatedAt = now
	return nil
}

// GetByID retrieves a reserved domain by ID
func (r *DomainRepository) GetByID(id int64) (*ReservedDomain, error) {
	query := `
		SELECT id, user_id, subdomain, created_at
		FROM reserved_domains WHERE id = ?
	`

	domain := &ReservedDomain{}
	err := r.db.QueryRow(query, id).Scan(
		&domain.ID,
		&domain.UserID,
		&domain.Subdomain,
		&domain.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDomainNotFound
		}
		return nil, fmt.Errorf("get reserved domain by id: %w", err)
	}

	return domain, nil
}

// GetBySubdomain retrieves a reserved domain by subdomain name
func (r *DomainRepository) GetBySubdomain(subdomain string) (*ReservedDomain, error) {
	query := `
		SELECT id, user_id, subdomain, created_at
		FROM reserved_domains WHERE subdomain = ?
	`

	domain := &ReservedDomain{}
	err := r.db.QueryRow(query, subdomain).Scan(
		&domain.ID,
		&domain.UserID,
		&domain.Subdomain,
		&domain.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDomainNotFound
		}
		return nil, fmt.Errorf("get reserved domain by subdomain: %w", err)
	}

	return domain, nil
}

// GetByUserID retrieves all reserved domains for a user
func (r *DomainRepository) GetByUserID(userID int64) ([]*ReservedDomain, error) {
	query := `
		SELECT id, user_id, subdomain, created_at
		FROM reserved_domains WHERE user_id = ? ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("get reserved domains by user id: %w", err)
	}
	defer rows.Close()

	var domains []*ReservedDomain
	for rows.Next() {
		domain := &ReservedDomain{}
		if err := rows.Scan(
			&domain.ID,
			&domain.UserID,
			&domain.Subdomain,
			&domain.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reserved domain: %w", err)
		}
		domains = append(domains, domain)
	}

	return domains, nil
}

// Delete deletes a reserved domain by ID
func (r *DomainRepository) Delete(id int64) error {
	query := `DELETE FROM reserved_domains WHERE id = ?`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete reserved domain: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrDomainNotFound
	}

	return nil
}

// DeleteByUserID deletes all reserved domains for a user
func (r *DomainRepository) DeleteByUserID(userID int64) error {
	query := `DELETE FROM reserved_domains WHERE user_id = ?`

	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("delete reserved domains by user id: %w", err)
	}

	return nil
}

// Count returns the number of reserved domains for a user
func (r *DomainRepository) Count(userID int64) (int, error) {
	query := `SELECT COUNT(*) FROM reserved_domains WHERE user_id = ?`
	var count int
	if err := r.db.QueryRow(query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count reserved domains: %w", err)
	}
	return count, nil
}

// IsAvailable checks if a subdomain is available (not reserved)
func (r *DomainRepository) IsAvailable(subdomain string) (bool, error) {
	query := `SELECT COUNT(*) FROM reserved_domains WHERE subdomain = ?`
	var count int
	if err := r.db.QueryRow(query, subdomain).Scan(&count); err != nil {
		return false, fmt.Errorf("check subdomain availability: %w", err)
	}
	return count == 0, nil
}

// IsOwnedByUser checks if a subdomain is reserved by a specific user
func (r *DomainRepository) IsOwnedByUser(subdomain string, userID int64) (bool, error) {
	query := `SELECT COUNT(*) FROM reserved_domains WHERE subdomain = ? AND user_id = ?`
	var count int
	if err := r.db.QueryRow(query, subdomain, userID).Scan(&count); err != nil {
		return false, fmt.Errorf("check subdomain ownership: %w", err)
	}
	return count > 0, nil
}
