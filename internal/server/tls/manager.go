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

	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/server/database"
	"github.com/mephistofox/fxtunnel/internal/server/store"
)

// CertManager manages TLS certificates with ACME and in-memory caching.
type CertManager struct {
	cfg        config.TLSSettings
	db         *database.Database
	log        zerolog.Logger
	cache      map[string]*tls.Certificate
	negCache   map[string]time.Time // SNI name -> when the "no such cert" answer expires
	mu         sync.RWMutex
	acmeMgr    *autocert.Manager
	redisCache store.TLSCache
	stopCh     chan struct{}
	stopOnce   sync.Once

	// revalFails counts consecutive ownership re-check failures per domain.
	// Only touched by the single renewal goroutine, so it needs no lock. It is
	// in-memory on purpose: a restart resets the counter, which can only delay
	// an un-verify, never cause a spurious one.
	revalFails map[string]int

	// onUnverify, when set, is called after a domain loses verification so the
	// owner of the routing table can stop serving it immediately.
	onUnverify func(domain string)
}

const (
	// negCacheTTL is how long an unknown SNI name is remembered as unknown.
	// Without it every TLS handshake with a random SNI costs two database
	// round-trips (certificate lookup plus hostPolicy), which an
	// unauthenticated client can repeat as fast as it can open sockets.
	negCacheTTL = 5 * time.Minute
	// negCacheMax bounds the negative cache so a flood of random SNI names
	// cannot grow it without limit; overflowing simply drops the whole table.
	negCacheMax = 10000
	// revalMaxFailures is how many consecutive ownership re-checks must fail
	// before a verified custom domain is un-verified (~36h at the 12h tick).
	revalMaxFailures = 3
)

// SetRedisCache sets an optional L2 Redis cache between memory and DB.
func (cm *CertManager) SetRedisCache(c store.TLSCache) {
	cm.redisCache = c
}

// NewCertManager creates a new certificate manager.
func NewCertManager(cfg config.TLSSettings, db *database.Database, log zerolog.Logger) *CertManager {
	cm := &CertManager{
		cfg:        cfg,
		db:         db,
		log:        log.With().Str("component", "cert_manager").Logger(),
		cache:      make(map[string]*tls.Certificate),
		negCache:   make(map[string]time.Time),
		revalFails: make(map[string]int),
		stopCh:     make(chan struct{}),
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

	// L1: local memory cache. The same lock also answers from the negative
	// cache, so a handshake for an unknown SNI name costs no database work.
	cm.mu.RLock()
	cert, ok := cm.cache[name]
	negUntil, negative := cm.negCache[name]
	cm.mu.RUnlock()
	if ok {
		return cert, nil
	}
	if negative && time.Now().Before(negUntil) {
		return nil, fmt.Errorf("no certificate for %s", name)
	}

	// L2: Redis shared cache
	if cm.redisCache != nil {
		certPEM, keyPEM, err := cm.redisCache.Get(name)
		if err == nil {
			tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
			if err == nil {
				cm.mu.Lock()
				cm.cache[name] = &tlsCert
				cm.mu.Unlock()
				return &tlsCert, nil
			}
		}
	}

	// L3: database
	dbCert, err := cm.db.TLSCerts.GetByDomain(name)
	if err == nil {
		tlsCert, err := tls.X509KeyPair(dbCert.CertPEM, dbCert.KeyPEM)
		if err == nil {
			cm.mu.Lock()
			cm.cache[name] = &tlsCert
			cm.mu.Unlock()
			if cm.redisCache != nil {
				_ = cm.redisCache.Put(name, dbCert.CertPEM, dbCert.KeyPEM, dbCert.ExpiresAt)
			}
			return &tlsCert, nil
		}
		cm.log.Warn().Str("domain", name).Err(err).Msg("Failed to parse cached certificate, falling back to ACME")
	}

	// Fall back to autocert — will obtain cert via ACME if domain is in hostPolicy
	acmeCert, err := cm.acmeMgr.GetCertificate(hello)
	if err != nil {
		cm.rememberUnknown(name)
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
		if cm.redisCache != nil {
			_ = cm.redisCache.Put(name, certPEM, keyPEM, expiresAt)
		}
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
		delete(cm.negCache, domain)
		cm.mu.Unlock()

		cm.log.Info().Str("domain", domain).Time("expires", expiresAt).Msg("TLS certificate obtained")
	}()
}

// rememberUnknown records that we have no certificate for this SNI name, so
// the next handshake for it is answered without touching the database.
func (cm *CertManager) rememberUnknown(name string) {
	cm.mu.Lock()
	if len(cm.negCache) >= negCacheMax {
		cm.negCache = make(map[string]time.Time, negCacheMax)
	}
	cm.negCache[name] = time.Now().Add(negCacheTTL)
	cm.mu.Unlock()
}

// RemoveCert removes a certificate from cache and database.
func (cm *CertManager) RemoveCert(domain string) {
	cm.mu.Lock()
	delete(cm.cache, domain)
	cm.mu.Unlock()

	if err := cm.db.TLSCerts.DeleteByDomain(domain); err != nil {
		cm.log.Warn().Str("domain", domain).Err(err).Msg("Failed to delete certificate from DB")
	}

	// autocert keeps its own copy of the cert and its private key under the
	// domain key; drop it too so we do not retain key material for a domain we
	// no longer serve.
	if err := cm.Delete(context.Background(), domain); err != nil {
		cm.log.Warn().Str("domain", domain).Err(err).Msg("Failed to delete autocert cache entry")
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
				cm.revalidateVerified()
				cm.renewExpiring()
			case <-cm.stopCh:
				return
			}
		}
	}()
}

// Stop stops the renewal goroutine. Safe to call multiple times.
func (cm *CertManager) Stop() {
	cm.stopOnce.Do(func() {
		close(cm.stopCh)
	})
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

// revalidateVerified re-checks the ownership TXT record of every verified
// custom domain. Verification used to be a one-way door: SetVerified(id, true)
// was never rolled back, so a domain stayed routable and re-issuable long
// after its owner lost control of it (expired registration, transferred zone).
// Only a run of consecutive failures un-verifies a domain, so a DNS outage
// does not take a customer's domain down.
func (cm *CertManager) revalidateVerified() {
	domains, err := cm.db.CustomDomains.GetAllVerified()
	if err != nil {
		cm.log.Error().Err(err).Msg("Failed to list verified custom domains for re-validation")
		return
	}

	for _, d := range domains {
		// Legacy rows predate TXT verification and carry no token. They cannot
		// be re-checked; un-verifying them would break live customers.
		if d.VerificationToken == "" {
			cm.log.Warn().Str("domain", d.Domain).Msg("Verified custom domain has no ownership token, skipping re-validation")
			continue
		}

		err := VerifyTXT(d.Domain, d.VerificationToken)
		if err == nil {
			delete(cm.revalFails, d.Domain)
			continue
		}
		cm.revalFails[d.Domain]++
		cm.log.Warn().
			Str("domain", d.Domain).
			Int("consecutive_failures", cm.revalFails[d.Domain]).
			Err(err).
			Msg("Custom domain ownership re-check failed")

		if cm.revalFails[d.Domain] < revalMaxFailures {
			continue
		}

		if err := cm.db.CustomDomains.SetVerified(d.ID, false); err != nil {
			cm.log.Error().Str("domain", d.Domain).Err(err).Msg("Failed to un-verify custom domain")
			continue
		}
		delete(cm.revalFails, d.Domain)
		cm.RemoveCert(d.Domain)
		// The routing table lives in the core server's memory; without this the
		// domain keeps being served until the next restart even though it is no
		// longer verified.
		if cm.onUnverify != nil {
			cm.onUnverify(d.Domain)
		}
		cm.log.Warn().Str("domain", d.Domain).Msg("Custom domain un-verified: ownership TXT record lost")
	}
}

// SetOnUnverify registers a callback invoked when a domain loses verification,
// so the caller can drop it from its routing table.
func (cm *CertManager) SetOnUnverify(fn func(domain string)) {
	cm.onUnverify = fn
}

func (cm *CertManager) hostPolicy(_ context.Context, host string) error {
	d, err := cm.db.CustomDomains.GetByDomain(host)
	if err != nil {
		return fmt.Errorf("unknown host: %s", host)
	}
	if !d.Verified {
		return fmt.Errorf("domain not verified: %s", host)
	}
	return nil
}

// autocert.Cache interface implementation.
//
// This is an opaque blob store, NOT a view over tls_certificates: autocert
// keeps its ACME account key under "acme_account+key" and challenge material
// under other synthetic keys, and it stores cert+key concatenated under a
// domain key. Mapping it onto the certificate table meant Put dropped
// everything and Get handed back a certificate without its private key, so
// every restart registered a fresh ACME account and every handshake could
// trigger a duplicate issuance.

func (cm *CertManager) Get(ctx context.Context, key string) ([]byte, error) {
	var data []byte
	err := cm.db.Pool().QueryRow(ctx, `SELECT data FROM autocert_cache WHERE key = $1`, key).Scan(&data)
	if err != nil {
		return nil, autocert.ErrCacheMiss
	}
	return data, nil
}

func (cm *CertManager) Put(ctx context.Context, key string, data []byte) error {
	_, err := cm.db.Pool().Exec(ctx,
		`INSERT INTO autocert_cache (key, data, updated_at) VALUES ($1, $2, NOW())
		 ON CONFLICT (key) DO UPDATE SET data = EXCLUDED.data, updated_at = NOW()`,
		key, data)
	if err != nil {
		return fmt.Errorf("autocert cache put %s: %w", key, err)
	}
	return nil
}

func (cm *CertManager) Delete(ctx context.Context, key string) error {
	_, err := cm.db.Pool().Exec(ctx, `DELETE FROM autocert_cache WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("autocert cache delete %s: %w", key, err)
	}
	return nil
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
