package tls

import (
	"fmt"
	"net"
	"strings"
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

// ValidateCustomDomain validates domain format for custom domain usage.
func ValidateCustomDomain(domain, baseDomain string) error {
	if domain == "" {
		return fmt.Errorf("domain is required")
	}

	if !strings.Contains(domain, ".") {
		return fmt.Errorf("invalid domain format")
	}

	if net.ParseIP(domain) != nil {
		return fmt.Errorf("IP addresses are not allowed")
	}

	if strings.EqualFold(domain, baseDomain) || strings.HasSuffix(strings.ToLower(domain), "."+strings.ToLower(baseDomain)) {
		return fmt.Errorf("cannot use base domain or its subdomains")
	}

	if strings.EqualFold(domain, "localhost") || strings.HasSuffix(strings.ToLower(domain), ".localhost") {
		return fmt.Errorf("localhost is not allowed")
	}

	return nil
}
