package api

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"sort"
	"time"
)

// certificateInfo is the JSON shape returned to admin UI for each TLS cert.
type certificateInfo struct {
	Hostname  string    `json:"hostname"`
	Subject   string    `json:"subject,omitempty"`
	SANs      []string  `json:"sans,omitempty"`
	Issuer    string    `json:"issuer,omitempty"`
	NotBefore time.Time `json:"not_before,omitempty"`
	NotAfter  time.Time `json:"not_after,omitempty"`
	DaysLeft  int       `json:"days_left"`
	Status    string    `json:"status"`           // ok | expiring | critical | expired | error
	Error     string    `json:"error,omitempty"`
	Source    string    `json:"source"`           // tls | database
	Wildcard  bool      `json:"wildcard"`
}

// handleAdminListCertificates returns the TLS posture for every known prod
// hostname (apex + aliases + standard admin/mon subdomains) plus every TLS
// certificate stored in the database for custom domains.
//
// Hostnames are probed via TLS to 127.0.0.1:443 with the right SNI, so this
// reflects what nginx actually serves — without needing root access to
// /etc/letsencrypt/.
func (s *Server) handleAdminListCertificates(w http.ResponseWriter, r *http.Request) {
	hostnames := collectKnownHostnames(s.cfg.Domain.Base, s.cfg.Domain.Aliases)

	results := make([]certificateInfo, 0, len(hostnames)+16)
	for _, host := range hostnames {
		results = append(results, probeCertificate(host))
	}

	// Custom-domain certs from the DB. We iterate via the custom_domains list
	// (TLSCerts repo only exposes GetExpiring/GetByDomain), then look up each
	// domain's cert.
	if s.db != nil && s.db.TLSCerts != nil && s.db.CustomDomains != nil {
		domains, _, err := s.db.CustomDomains.GetAll(10_000, 0)
		if err == nil {
			for _, d := range domains {
				cert, err := s.db.TLSCerts.GetByDomain(d.Domain)
				if err != nil || cert == nil {
					continue
				}
				results = append(results, certInfoFromDB(d.Domain, cert.IssuedAt, cert.ExpiresAt))
			}
		}
	}

	// Stable sort: expired/critical first, then by days_left ascending so
	// the admin sees what to act on at the top.
	sort.SliceStable(results, func(i, j int) bool {
		return statusPriority(results[i].Status) < statusPriority(results[j].Status) ||
			(results[i].Status == results[j].Status && results[i].DaysLeft < results[j].DaysLeft)
	})

	s.respondJSON(w, http.StatusOK, map[string]any{
		"certificates": results,
		"total":        len(results),
	})
}

// collectKnownHostnames builds the list of hostnames whose TLS we expect to
// manage: base + aliases, plus standard admin/mon/www subdomains for each.
func collectKnownHostnames(base string, aliases []string) []string {
	domains := []string{}
	if base != "" {
		domains = append(domains, base)
	}
	domains = append(domains, aliases...)

	seen := map[string]bool{}
	out := []string{}
	for _, d := range domains {
		if d == "" {
			continue
		}
		for _, host := range []string{d, "www." + d, "admin." + d, "mon." + d} {
			if seen[host] {
				continue
			}
			seen[host] = true
			out = append(out, host)
		}
	}
	return out
}

// probeCertificate connects via TLS to 127.0.0.1:443 with the given SNI and
// extracts the peer certificate. InsecureSkipVerify is set so we still parse
// even when the cert doesn't match the SNI (we want to surface that case).
func probeCertificate(hostname string) certificateInfo {
	info := certificateInfo{Hostname: hostname, Source: "tls", Status: "error"}

	cert, err := fetchPeerCert(hostname)
	if err != nil {
		info.Error = err.Error()
		return info
	}

	info.Subject = cert.Subject.CommonName
	info.SANs = cert.DNSNames
	info.Issuer = cert.Issuer.CommonName
	info.NotBefore = cert.NotBefore.UTC()
	info.NotAfter = cert.NotAfter.UTC()
	info.DaysLeft = int(time.Until(cert.NotAfter).Hours() / 24)
	info.Status = computeCertStatus(info.DaysLeft)
	info.Wildcard = false
	for _, san := range cert.DNSNames {
		if len(san) > 0 && san[0] == '*' {
			info.Wildcard = true
			break
		}
	}
	return info
}

func fetchPeerCert(hostname string) (*x509.Certificate, error) {
	d := &net.Dialer{Timeout: 5 * time.Second}
	conf := &tls.Config{
		ServerName:         hostname,
		InsecureSkipVerify: true, //nolint:gosec // we extract the cert; we don't trust it for routing
		MinVersion:         tls.VersionTLS12,
	}
	conn, err := tls.DialWithDialer(d, "tcp", "127.0.0.1:443", conf)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("no peer certificates")
	}
	return certs[0], nil
}

func certInfoFromDB(domain string, issued, expires time.Time) certificateInfo {
	days := int(time.Until(expires).Hours() / 24)
	return certificateInfo{
		Hostname:  domain,
		NotBefore: issued.UTC(),
		NotAfter:  expires.UTC(),
		DaysLeft:  days,
		Status:    computeCertStatus(days),
		Source:    "database",
	}
}

func computeCertStatus(daysLeft int) string {
	switch {
	case daysLeft < 0:
		return "expired"
	case daysLeft <= 7:
		return "critical"
	case daysLeft <= 30:
		return "expiring"
	default:
		return "ok"
	}
}

func statusPriority(s string) int {
	switch s {
	case "expired":
		return 0
	case "critical":
		return 1
	case "expiring":
		return 2
	case "error":
		return 3
	case "ok":
		return 4
	default:
		return 5
	}
}
