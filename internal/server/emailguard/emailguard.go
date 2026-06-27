// Package emailguard validates that an email address is real and not a
// disposable / throwaway address, to make automated abuse and scam-account
// registration expensive while staying frictionless for genuine users.
//
// Two independent layers are applied:
//
//  1. A curated, embedded blocklist of known disposable-mail domains (and their
//     subdomains). This is fast and offline.
//  2. A DNS check that the domain can actually receive mail — an MX record, or
//     (per RFC 5321 §5.1) a fallback A/AAAA record. This catches typos and the
//     long tail of brand-new throwaway domains that no static list can track.
//
// The DNS layer caches both positive and negative results for a short TTL so a
// burst of sign-ups from the same domain does not hammer the resolver and does
// not sit on the request hot-path repeatedly.
package emailguard

import (
	"context"
	_ "embed"
	"errors"
	"net"
	"strings"
	"sync"
	"time"
)

//go:embed disposable_domains.txt
var disposableList string

var (
	// ErrInvalidEmail is returned when the address is not syntactically valid.
	ErrInvalidEmail = errors.New("invalid email address")
	// ErrDisposableEmail is returned for known throwaway / temp-mail domains.
	ErrDisposableEmail = errors.New("disposable email addresses are not allowed")
	// ErrUndeliverable is returned when the domain has no MX/A record and so
	// cannot receive mail (non-existent or misconfigured domain).
	ErrUndeliverable = errors.New("email domain cannot receive mail")
)

// resolver is the DNS interface used for MX/host lookups; overridable in tests.
type resolver interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupHost(ctx context.Context, host string) ([]string, error)
}

type mxCacheEntry struct {
	ok        bool
	expiresAt time.Time
}

// Validator checks email addresses against the disposable blocklist and DNS.
// The zero value is not usable; construct with New.
type Validator struct {
	disposable map[string]struct{}
	resolver   resolver
	dnsTimeout time.Duration
	cacheTTL   time.Duration

	mu    sync.Mutex
	cache map[string]mxCacheEntry
}

// New builds a Validator from the embedded blocklist with sensible defaults
// (1.5s DNS timeout, 1h result cache).
func New() *Validator {
	return &Validator{
		disposable: parseList(disposableList),
		resolver:   net.DefaultResolver,
		dnsTimeout: 1500 * time.Millisecond,
		cacheTTL:   time.Hour,
		cache:      make(map[string]mxCacheEntry),
	}
}

// parseList turns the embedded file into a domain set (lowercased, no comments).
func parseList(raw string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		set[strings.ToLower(line)] = struct{}{}
	}
	return set
}

// Normalize trims and lowercases an address and returns its local and domain
// parts. It performs a lightweight syntactic check (exactly one "@", non-empty
// local part, a dotted domain) — full RFC 5322 parsing is intentionally avoided
// as it accepts many addresses that bounce in practice.
func Normalize(email string) (normalized, domain string, err error) {
	email = strings.TrimSpace(email)
	if len(email) == 0 || len(email) > 254 {
		return "", "", ErrInvalidEmail
	}
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "", "", ErrInvalidEmail
	}
	local := email[:at]
	domain = strings.ToLower(email[at+1:])
	if len(local) > 64 || strings.ContainsAny(email, " \t\r\n") {
		return "", "", ErrInvalidEmail
	}
	// Domain must look like a hostname: at least one dot, no empty labels.
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") || strings.Contains(domain, "..") {
		return "", "", ErrInvalidEmail
	}
	for _, r := range domain {
		if !(r == '.' || r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return "", "", ErrInvalidEmail
		}
	}
	return local + "@" + domain, domain, nil
}

// IsDisposable reports whether domain (or any parent domain) is on the
// blocklist, so "foo.mailinator.com" matches a "mailinator.com" entry.
func (v *Validator) IsDisposable(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	labels := strings.Split(domain, ".")
	for i := 0; i < len(labels)-1; i++ {
		candidate := strings.Join(labels[i:], ".")
		if _, ok := v.disposable[candidate]; ok {
			return true
		}
	}
	return false
}

// Validate normalizes the address, rejects disposable domains, and verifies the
// domain can receive mail. It returns the normalized address on success.
func (v *Validator) Validate(ctx context.Context, email string) (string, error) {
	normalized, domain, err := Normalize(email)
	if err != nil {
		return "", err
	}
	if v.IsDisposable(domain) {
		return "", ErrDisposableEmail
	}
	if !v.domainAcceptsMail(ctx, domain) {
		return "", ErrUndeliverable
	}
	return normalized, nil
}

// domainAcceptsMail checks (with caching) whether domain has an MX record or,
// failing that, an A/AAAA record it could deliver mail to.
func (v *Validator) domainAcceptsMail(ctx context.Context, domain string) bool {
	now := time.Now()

	v.mu.Lock()
	if entry, ok := v.cache[domain]; ok && now.Before(entry.expiresAt) {
		v.mu.Unlock()
		return entry.ok
	}
	v.mu.Unlock()

	ok := v.lookup(ctx, domain)

	v.mu.Lock()
	v.cache[domain] = mxCacheEntry{ok: ok, expiresAt: now.Add(v.cacheTTL)}
	v.mu.Unlock()

	return ok
}

func (v *Validator) lookup(ctx context.Context, domain string) bool {
	ctx, cancel := context.WithTimeout(ctx, v.dnsTimeout)
	defer cancel()

	if mx, err := v.resolver.LookupMX(ctx, domain); err == nil {
		for _, rec := range mx {
			if strings.TrimSuffix(rec.Host, ".") != "" {
				return true
			}
		}
	}
	// RFC 5321 §5.1: with no MX, an implicit A/AAAA record is used.
	if hosts, err := v.resolver.LookupHost(ctx, domain); err == nil && len(hosts) > 0 {
		return true
	}
	return false
}
