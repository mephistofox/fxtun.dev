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
