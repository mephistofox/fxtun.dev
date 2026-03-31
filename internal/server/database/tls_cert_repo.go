package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"
)

// ErrTLSCertNotFound is returned when a TLS certificate is not found.
var ErrTLSCertNotFound = errors.New("tls certificate not found")

// TLSCertRepository provides access to tls_certificates table.
type TLSCertRepository struct {
	db            *sql.DB
	encryptionKey []byte // 32 bytes for AES-256; nil means no encryption
}

// NewTLSCertRepository creates a new TLSCertRepository.
func NewTLSCertRepository(db *sql.DB) *TLSCertRepository {
	return &TLSCertRepository{db: db}
}

// SetEncryptionKey configures AES-256-GCM encryption for TLS private keys at rest.
func (r *TLSCertRepository) SetEncryptionKey(key []byte) {
	r.encryptionKey = key
}

// encryptKey encrypts a private key using AES-256-GCM.
func (r *TLSCertRepository) encryptKey(plaintext []byte) ([]byte, error) {
	if r.encryptionKey == nil {
		return plaintext, nil
	}

	block, err := aes.NewCipher(r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	// Prefix with "enc:" marker so we can distinguish from plaintext PEM
	marker := []byte("enc:")
	result := make([]byte, len(marker)+base64.StdEncoding.EncodedLen(len(ciphertext)))
	copy(result, marker)
	base64.StdEncoding.Encode(result[len(marker):], ciphertext)
	return result, nil
}

// decryptKey decrypts a private key. Falls back to plaintext for unencrypted legacy keys.
func (r *TLSCertRepository) decryptKey(data []byte) ([]byte, error) {
	// Check for encryption marker
	marker := []byte("enc:")
	if len(data) < len(marker) || string(data[:len(marker)]) != "enc:" {
		// Legacy unencrypted key — return as-is
		return data, nil
	}

	if r.encryptionKey == nil {
		return nil, errors.New("encrypted TLS key found but no encryption key configured")
	}

	encoded := data[len(marker):]
	ciphertext, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// Upsert inserts or updates a TLS certificate. Private keys are encrypted at rest.
func (r *TLSCertRepository) Upsert(cert *TLSCertificate) error {
	keyData, err := r.encryptKey(cert.KeyPEM)
	if err != nil {
		return fmt.Errorf("encrypt TLS key: %w", err)
	}

	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO tls_certificates (domain, cert_pem, key_pem, expires_at, issued_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(domain) DO UPDATE SET cert_pem=excluded.cert_pem, key_pem=excluded.key_pem, expires_at=excluded.expires_at, issued_at=excluded.issued_at`,
		cert.Domain, cert.CertPEM, keyData, cert.ExpiresAt, cert.IssuedAt, now,
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

// GetByDomain retrieves a TLS certificate by domain. Private keys are decrypted transparently.
func (r *TLSCertRepository) GetByDomain(domain string) (*TLSCertificate, error) {
	c := &TLSCertificate{}
	var keyData []byte
	err := r.db.QueryRow(
		`SELECT id, domain, cert_pem, key_pem, expires_at, issued_at, created_at FROM tls_certificates WHERE domain = ?`, domain,
	).Scan(&c.ID, &c.Domain, &c.CertPEM, &keyData, &c.ExpiresAt, &c.IssuedAt, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTLSCertNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tls certificate: %w", err)
	}

	c.KeyPEM, err = r.decryptKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("decrypt TLS key for %s: %w", domain, err)
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
		var keyData []byte
		if err := rows.Scan(&c.ID, &c.Domain, &c.CertPEM, &keyData, &c.ExpiresAt, &c.IssuedAt, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tls certificate: %w", err)
		}
		c.KeyPEM, err = r.decryptKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("decrypt TLS key for %s: %w", c.Domain, err)
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
