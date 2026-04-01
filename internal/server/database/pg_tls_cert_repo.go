package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/database/sqlc"
)

// TLSCertRepository handles TLS certificate database operations using PostgreSQL via sqlc.
// Private keys are encrypted at rest with AES-256-GCM when an encryption key is configured.
type TLSCertRepository struct {
	q          *sqlc.Queries
	encryptKey []byte // 32 bytes for AES-256; nil means no encryption
}

// SetEncryptionKey configures AES-256-GCM encryption for TLS private keys at rest.
func (r *TLSCertRepository) SetEncryptionKey(key []byte) {
	r.encryptKey = key
}

// encryptKeyPEM encrypts a private key using AES-256-GCM.
func (r *TLSCertRepository) encryptKeyPEM(plaintext []byte) ([]byte, error) {
	if r.encryptKey == nil {
		return plaintext, nil
	}

	block, err := aes.NewCipher(r.encryptKey)
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

// decryptKeyPEM decrypts a private key. Falls back to plaintext for unencrypted legacy keys.
func (r *TLSCertRepository) decryptKeyPEM(data []byte) ([]byte, error) {
	// Check for encryption marker
	marker := []byte("enc:")
	if len(data) < len(marker) || string(data[:len(marker)]) != "enc:" {
		// Legacy unencrypted key — return as-is
		return data, nil
	}

	if r.encryptKey == nil {
		return nil, errors.New("encrypted TLS key found but no encryption key configured")
	}

	encoded := data[len(marker):]
	ciphertext, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(r.encryptKey)
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

// sqlcTLSCertToDomain converts a sqlc.TlsCertificate to a domain TLSCertificate,
// decrypting the private key in the process.
func (r *TLSCertRepository) sqlcTLSCertToDomain(c sqlc.TlsCertificate) (*TLSCertificate, error) {
	keyPEM, err := r.decryptKeyPEM(c.KeyPem)
	if err != nil {
		return nil, fmt.Errorf("decrypt TLS key for %s: %w", c.Domain, err)
	}
	return &TLSCertificate{
		ID:        c.ID,
		Domain:    c.Domain,
		CertPEM:   c.CertPem,
		KeyPEM:    keyPEM,
		ExpiresAt: tsToTime(c.ExpiresAt),
		IssuedAt:  tsToTime(c.IssuedAt),
		CreatedAt: tsToTime(c.CreatedAt),
	}, nil
}

// Upsert inserts or updates a TLS certificate. Private keys are encrypted at rest.
func (r *TLSCertRepository) Upsert(cert *TLSCertificate) error {
	keyData, err := r.encryptKeyPEM(cert.KeyPEM)
	if err != nil {
		return fmt.Errorf("encrypt TLS key: %w", err)
	}

	ctx := context.Background()
	id, err := r.q.UpsertTLSCertificate(ctx, sqlc.UpsertTLSCertificateParams{
		Domain:    cert.Domain,
		CertPem:   cert.CertPEM,
		KeyPem:    keyData,
		ExpiresAt: timeToPgtz(cert.ExpiresAt),
		IssuedAt:  timeToPgtz(cert.IssuedAt),
	})
	if err != nil {
		return fmt.Errorf("upsert tls certificate: %w", err)
	}
	cert.ID = id
	cert.CreatedAt = time.Now()
	return nil
}

// GetByDomain retrieves a TLS certificate by domain. Private keys are decrypted transparently.
func (r *TLSCertRepository) GetByDomain(domain string) (*TLSCertificate, error) {
	ctx := context.Background()
	c, err := r.q.GetTLSCertByDomain(ctx, domain)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrTLSCertNotFound
		}
		return nil, fmt.Errorf("get tls certificate: %w", err)
	}
	return r.sqlcTLSCertToDomain(c)
}

// GetExpiring retrieves certificates expiring before the given time.
func (r *TLSCertRepository) GetExpiring(before time.Time) ([]*TLSCertificate, error) {
	ctx := context.Background()
	rows, err := r.q.ListExpiringTLSCerts(ctx, timeToPgtz(before))
	if err != nil {
		return nil, fmt.Errorf("get expiring certs: %w", err)
	}
	certs := make([]*TLSCertificate, 0, len(rows))
	for _, c := range rows {
		cert, err := r.sqlcTLSCertToDomain(c)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

// DeleteByDomain removes a TLS certificate by domain.
func (r *TLSCertRepository) DeleteByDomain(domain string) error {
	ctx := context.Background()
	err := r.q.DeleteTLSCertByDomain(ctx, domain)
	if err != nil {
		return fmt.Errorf("delete tls certificate: %w", err)
	}
	return nil
}
