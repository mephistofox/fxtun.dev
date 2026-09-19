package tls

import (
	"fmt"
	"net"
	"strings"
	"unicode"
)

// VerifyCNAME checks that domain has a CNAME pointing to expectedTarget.
func VerifyCNAME(domain, expectedTarget string) error {
	cname, err := net.LookupCNAME(domain)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %s: %w", domain, err)
	}

	cname = strings.TrimSuffix(cname, ".")
	expectedTarget = strings.TrimSuffix(expectedTarget, ".")

	if !strings.EqualFold(cname, expectedTarget) {
		return fmt.Errorf("CNAME mismatch: %s points to %s, expected %s", domain, cname, expectedTarget)
	}

	return nil
}

// IsApexDomain returns true if the domain is a second-level domain (e.g. example.com).
func IsApexDomain(domain string) bool {
	parts := strings.Split(domain, ".")
	return len(parts) == 2
}

// VerifyARecord checks that domain resolves to the same IP addresses as expectedTarget.
func VerifyARecord(domain, expectedTarget string) error {
	targetIPs, err := net.LookupHost(expectedTarget)
	if err != nil {
		return fmt.Errorf("failed to resolve target %s: %w", expectedTarget, err)
	}

	domainIPs, err := net.LookupHost(domain)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %s: %w", domain, err)
	}

	targetSet := make(map[string]bool, len(targetIPs))
	for _, ip := range targetIPs {
		targetSet[ip] = true
	}

	for _, ip := range domainIPs {
		if targetSet[ip] {
			return nil
		}
	}

	return fmt.Errorf("A/AAAA mismatch: %s resolves to %v, expected one of %v (from %s)", domain, domainIPs, targetIPs, expectedTarget)
}

// VerifyDNS checks domain ownership. For apex domains (2nd level) it verifies
// A/AAAA records point to the same IP as expectedTarget. For subdomains (3rd+ level)
// it first tries CNAME, then falls back to A/AAAA verification.
func VerifyDNS(domain, expectedTarget string) error {
	if IsApexDomain(domain) {
		return VerifyARecord(domain, expectedTarget)
	}

	if err := VerifyCNAME(domain, expectedTarget); err == nil {
		return nil
	}

	return VerifyARecord(domain, expectedTarget)
}

// ChallengeRecordName returns the DNS name where the ownership-proof TXT
// record must be published for the given custom domain.
func ChallengeRecordName(domain string) string {
	return "_fxtunnel-challenge." + strings.TrimSuffix(domain, ".")
}

// VerifyTXT proves domain ownership: it looks up the TXT records at
// _fxtunnel-challenge.<domain> and requires one to equal the per-domain token.
// Unlike A/CNAME checks (which only prove the domain points at the shared
// server IP — something any tenant can do), a unique secret TXT token can only
// be set by whoever controls the domain's DNS, preventing cross-tenant
// custom-domain takeover.
func VerifyTXT(domain, token string) error {
	if token == "" {
		return fmt.Errorf("no verification token issued for %s", domain)
	}
	name := ChallengeRecordName(domain)
	records, err := net.LookupTXT(name)
	if err != nil {
		return fmt.Errorf("TXT lookup failed for %s: %w", name, err)
	}
	for _, rec := range records {
		if strings.TrimSpace(rec) == token {
			return nil
		}
	}
	return fmt.Errorf("ownership TXT record not found at %s (expected token value)", name)
}

// NormalizeDomain returns the canonical form of a DNS name: trimmed,
// lowercased, without the trailing root dot. Custom domains MUST be stored in
// this form — the runtime routing map lowercases on lookup and removal, so a
// row stored as "EXAMPLE.com" would evict the entry belonging to the owner of
// "example.com".
// It must be idempotent: the API normalizes the submitted domain and
// ValidateCustomDomain normalizes it again, and a value that changes between
// the two passes is validated in one form and stored in another. Trimming the
// trailing dots and spaces together in a single pass is what guarantees that —
// "a.com. ." and ".." both settle on the first try.
func NormalizeDomain(domain string) string {
	trimmed := strings.TrimRightFunc(strings.ToLower(domain), func(r rune) bool {
		return r == '.' || unicode.IsSpace(r)
	})
	return strings.TrimLeftFunc(trimmed, unicode.IsSpace)
}

// validHostname reports whether name (already normalized) is a syntactically
// valid multi-label DNS hostname. Everything that passes ValidateCustomDomain
// is stored, keyed in the runtime SNI routing map and later handed to ACME, so
// "has a dot in it" is not a sufficient gate: spaces, newlines, wildcards and
// non-ASCII all have to be rejected here.
func validHostname(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}
	labels := strings.Split(name, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if c != '-' && !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') {
				return false
			}
		}
	}
	return true
}

// ValidateCustomDomain validates domain format for custom domain usage.
// reservedDomains lists names owned by this installation (base domain, its
// aliases and every authoritative DNS zone); the domain may be neither equal
// to nor a subdomain of any of them.
func ValidateCustomDomain(domain string, reservedDomains ...string) error {
	if domain == "" {
		return fmt.Errorf("domain is required")
	}

	name := NormalizeDomain(domain)

	if !validHostname(name) {
		return fmt.Errorf("invalid domain format")
	}

	if net.ParseIP(name) != nil {
		return fmt.Errorf("IP addresses are not allowed")
	}

	for _, reserved := range reservedDomains {
		reserved = NormalizeDomain(reserved)
		if reserved == "" {
			continue
		}
		if name == reserved || strings.HasSuffix(name, "."+reserved) {
			return fmt.Errorf("cannot use base domain or its subdomains")
		}
	}

	if name == "localhost" || strings.HasSuffix(name, ".localhost") {
		return fmt.Errorf("localhost is not allowed")
	}

	return nil
}
