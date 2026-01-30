package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrTLSCertNotFound is returned when a TLS certificate is not found.
var ErrTLSCertNotFound = errors.New("tls certificate not found")

// TLSCertRepository provides access to tls_certificates table.
type TLSCertRepository struct {
	db *sql.DB
}

// NewTLSCertRepository creates a new TLSCertRepository.
func NewTLSCertRepository(db *sql.DB) *TLSCertRepository {
	return &TLSCertRepository{db: db}
}

// Upsert inserts or updates a TLS certificate.
func (r *TLSCertRepository) Upsert(cert *TLSCertificate) error {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO tls_certificates (domain, cert_pem, key_pem, expires_at, issued_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(domain) DO UPDATE SET cert_pem=excluded.cert_pem, key_pem=excluded.key_pem, expires_at=excluded.expires_at, issued_at=excluded.issued_at`,
		cert.Domain, cert.CertPEM, cert.KeyPEM, cert.ExpiresAt, cert.IssuedAt, now,
	)
	if err != nil {
		return fmt.Errorf("upsert tls certificate: %w", err)
	}
	id, _ := result.LastInsertId()
	if id > 0 {
		cert.ID = id
	}
	cert.CreatedAt = now
	return nil
}

// GetByDomain retrieves a TLS certificate by domain.
func (r *TLSCertRepository) GetByDomain(domain string) (*TLSCertificate, error) {
	c := &TLSCertificate{}
	err := r.db.QueryRow(
		`SELECT id, domain, cert_pem, key_pem, expires_at, issued_at, created_at FROM tls_certificates WHERE domain = ?`, domain,
	).Scan(&c.ID, &c.Domain, &c.CertPEM, &c.KeyPEM, &c.ExpiresAt, &c.IssuedAt, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTLSCertNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tls certificate: %w", err)
	}
	return c, nil
}

// GetExpiring retrieves certificates expiring before the given time.
func (r *TLSCertRepository) GetExpiring(before time.Time) ([]*TLSCertificate, error) {
	rows, err := r.db.Query(
		`SELECT id, domain, cert_pem, key_pem, expires_at, issued_at, created_at FROM tls_certificates WHERE expires_at < ?`, before,
	)
	if err != nil {
		return nil, fmt.Errorf("get expiring certs: %w", err)
	}
	defer rows.Close()

	var certs []*TLSCertificate
	for rows.Next() {
		c := &TLSCertificate{}
		if err := rows.Scan(&c.ID, &c.Domain, &c.CertPEM, &c.KeyPEM, &c.ExpiresAt, &c.IssuedAt, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tls certificate: %w", err)
		}
		certs = append(certs, c)
	}
	return certs, nil
}

// DeleteByDomain removes a TLS certificate by domain.
func (r *TLSCertRepository) DeleteByDomain(domain string) error {
	_, err := r.db.Exec(`DELETE FROM tls_certificates WHERE domain = ?`, domain)
	if err != nil {
		return fmt.Errorf("delete tls certificate: %w", err)
	}
	return nil
}
