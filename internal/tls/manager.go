package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/acme/autocert"

	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/database"
)

// CertManager manages TLS certificates with ACME and in-memory caching.
type CertManager struct {
	cfg     config.TLSSettings
	db      *database.Database
	log     zerolog.Logger
	cache   map[string]*tls.Certificate
	mu      sync.RWMutex
	acmeMgr *autocert.Manager
	stopCh  chan struct{}
}

// NewCertManager creates a new certificate manager.
func NewCertManager(cfg config.TLSSettings, db *database.Database, log zerolog.Logger) *CertManager {
	cm := &CertManager{
		cfg:    cfg,
		db:     db,
		log:    log.With().Str("component", "cert_manager").Logger(),
		cache:  make(map[string]*tls.Certificate),
		stopCh: make(chan struct{}),
	}

	cm.acmeMgr = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Email:      cfg.ACMEEmail,
		Cache:      cm,
		HostPolicy: cm.hostPolicy,
	}

	return cm
}

// LoadFromDB loads all certificates from the database into memory cache.
func (cm *CertManager) LoadFromDB() error {
	domains, err := cm.db.CustomDomains.GetAllVerified()
	if err != nil {
		return fmt.Errorf("load verified domains: %w", err)
	}

	loaded := 0
	for _, d := range domains {
		cert, err := cm.db.TLSCerts.GetByDomain(d.Domain)
		if err != nil {
			continue
		}
		tlsCert, err := tls.X509KeyPair(cert.CertPEM, cert.KeyPEM)
		if err != nil {
			cm.log.Warn().Str("domain", d.Domain).Err(err).Msg("Failed to parse certificate")
			continue
		}
		cm.mu.Lock()
		cm.cache[d.Domain] = &tlsCert
		cm.mu.Unlock()
		loaded++
	}

	cm.log.Info().Int("count", loaded).Msg("Loaded TLS certificates from database")
	return nil
}

// GetCertificate is the tls.Config.GetCertificate callback for SNI-based cert selection.
// It first checks the local cache/DB, then falls back to autocert for on-demand ACME issuance.
func (cm *CertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	name := hello.ServerName

	cm.mu.RLock()
	cert, ok := cm.cache[name]
	cm.mu.RUnlock()
	if ok {
		return cert, nil
	}

	dbCert, err := cm.db.TLSCerts.GetByDomain(name)
	if err == nil {
		tlsCert, err := tls.X509KeyPair(dbCert.CertPEM, dbCert.KeyPEM)
		if err == nil {
			cm.mu.Lock()
			cm.cache[name] = &tlsCert
			cm.mu.Unlock()
			return &tlsCert, nil
		}
		cm.log.Warn().Str("domain", name).Err(err).Msg("Failed to parse cached certificate, falling back to ACME")
	}

	// Fall back to autocert — will obtain cert via ACME if domain is in hostPolicy
	acmeCert, err := cm.acmeMgr.GetCertificate(hello)
	if err != nil {
		return nil, fmt.Errorf("no certificate for %s: %w", name, err)
	}

	// Store the obtained cert
	certPEM, keyPEM, expiresAt, extractErr := extractPEM(acmeCert)
	if extractErr == nil {
		_ = cm.db.TLSCerts.Upsert(&database.TLSCertificate{
			Domain:    name,
			CertPEM:   certPEM,
			KeyPEM:    keyPEM,
			ExpiresAt: expiresAt,
			IssuedAt:  time.Now(),
		})
		cm.mu.Lock()
		cm.cache[name] = acmeCert
		cm.mu.Unlock()
		cm.log.Info().Str("domain", name).Time("expires", expiresAt).Msg("TLS certificate obtained on-demand")
	}

	return acmeCert, nil
}

// ObtainCert obtains a certificate for a domain via ACME in background.
func (cm *CertManager) ObtainCert(domain string) {
	go func() {
		cm.log.Info().Str("domain", domain).Msg("Obtaining TLS certificate")

		hello := &tls.ClientHelloInfo{ServerName: domain}
		cert, err := cm.acmeMgr.GetCertificate(hello)
		if err != nil {
			cm.log.Error().Str("domain", domain).Err(err).Msg("Failed to obtain certificate")
			return
		}

		certPEM, keyPEM, expiresAt, err := extractPEM(cert)
		if err != nil {
			cm.log.Error().Str("domain", domain).Err(err).Msg("Failed to extract PEM")
			return
		}

		dbCert := &database.TLSCertificate{
			Domain:    domain,
			CertPEM:   certPEM,
			KeyPEM:    keyPEM,
			ExpiresAt: expiresAt,
			IssuedAt:  time.Now(),
		}
		if err := cm.db.TLSCerts.Upsert(dbCert); err != nil {
			cm.log.Error().Str("domain", domain).Err(err).Msg("Failed to store certificate")
			return
		}

		cm.mu.Lock()
		cm.cache[domain] = cert
		cm.mu.Unlock()

		cm.log.Info().Str("domain", domain).Time("expires", expiresAt).Msg("TLS certificate obtained")
	}()
}

// RemoveCert removes a certificate from cache and database.
func (cm *CertManager) RemoveCert(domain string) {
	cm.mu.Lock()
	delete(cm.cache, domain)
	cm.mu.Unlock()

	if err := cm.db.TLSCerts.DeleteByDomain(domain); err != nil {
		cm.log.Warn().Str("domain", domain).Err(err).Msg("Failed to delete certificate from DB")
	}
}

// HandleACMEChallenge serves ACME HTTP-01 challenge responses.
// Returns true if this was an ACME challenge request.
func (cm *CertManager) HandleACMEChallenge(w http.ResponseWriter, r *http.Request) bool {
	if cm.acmeMgr == nil {
		return false
	}
	const prefix = "/.well-known/acme-challenge/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		return false
	}
	cm.acmeMgr.HTTPHandler(nil).ServeHTTP(w, r)
	return true
}

// StartRenewal starts the background renewal goroutine.
func (cm *CertManager) StartRenewal() {
	go func() {
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cm.renewExpiring()
			case <-cm.stopCh:
				return
			}
		}
	}()
}

// Stop stops the renewal goroutine.
func (cm *CertManager) Stop() {
	close(cm.stopCh)
}

// TLSConfig returns a tls.Config using this manager's GetCertificate.
// Includes acme-tls/1 in NextProtos for tls-alpn-01 challenge support.
func (cm *CertManager) TLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: cm.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1", "acme-tls/1"},
		MinVersion:     tls.VersionTLS12,
	}
}

func (cm *CertManager) renewExpiring() {
	threshold := time.Now().Add(30 * 24 * time.Hour)
	certs, err := cm.db.TLSCerts.GetExpiring(threshold)
	if err != nil {
		cm.log.Error().Err(err).Msg("Failed to get expiring certificates")
		return
	}

	for _, cert := range certs {
		cm.log.Info().Str("domain", cert.Domain).Time("expires", cert.ExpiresAt).Msg("Renewing certificate")
		cm.ObtainCert(cert.Domain)
	}
}

func (cm *CertManager) hostPolicy(_ context.Context, host string) error {
	_, err := cm.db.CustomDomains.GetByDomain(host)
	if err != nil {
		return fmt.Errorf("unknown host: %s", host)
	}
	return nil
}

// autocert.Cache interface implementation

func (cm *CertManager) Get(_ context.Context, key string) ([]byte, error) {
	cert, err := cm.db.TLSCerts.GetByDomain(key)
	if err != nil {
		return nil, autocert.ErrCacheMiss
	}
	return cert.CertPEM, nil
}

func (cm *CertManager) Put(_ context.Context, _ string, _ []byte) error {
	return nil
}

func (cm *CertManager) Delete(_ context.Context, key string) error {
	return cm.db.TLSCerts.DeleteByDomain(key)
}

func extractPEM(cert *tls.Certificate) (certPEM, keyPEM []byte, expiresAt time.Time, err error) {
	for _, b := range cert.Certificate {
		certPEM = append(certPEM, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: b})...)
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("parse leaf: %w", err)
	}
	expiresAt = leaf.NotAfter

	keyBytes, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("marshal private key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})

	return certPEM, keyPEM, expiresAt, nil
}
