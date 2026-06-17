package core

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// isIPAllowed returns true if ip is permitted by the tunnel's allowlist.
// If the allowlist is empty, all IPs are permitted.
func isIPAllowed(ip net.IP, tunnel *Tunnel) bool {
	if len(tunnel.AllowedNets) == 0 && len(tunnel.AllowedIPs) == 0 {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	for _, allowed := range tunnel.AllowedIPs {
		a := allowed
		if v4 := a.To4(); v4 != nil {
			a = v4
		}
		if ip.Equal(a) {
			return true
		}
	}
	for _, cidr := range tunnel.AllowedNets {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// checkIPAllowlist validates the client IP against the tunnel's IP allowlist.
// Returns true if the request is allowed (either no restriction or IP matches).
// Returns false and writes a 403 response if the IP is not in the allowlist.
//
// trusted is the set of reverse-proxy IPs whose forwarded headers may be
// believed; pass the server's auth.trusted_proxies set.
func checkIPAllowlist(w http.ResponseWriter, r *http.Request, tunnel *Tunnel, trusted map[string]struct{}) bool {
	if len(tunnel.AllowedNets) == 0 && len(tunnel.AllowedIPs) == 0 {
		return true
	}
	clientIP := extractClientIP(r, trusted)
	if clientIP == nil || !isIPAllowed(clientIP, tunnel) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}
	return true
}

// extractClientIP extracts the real client IP from the request.
//
// Forwarded headers (X-Real-IP / X-Forwarded-For) are honoured ONLY when the
// immediate TCP peer is a trusted reverse proxy. A direct, untrusted
// connection cannot spoof its source IP through these headers — otherwise an
// attacker could bypass a tunnel's IP allowlist by sending X-Forwarded-For of
// an allowed address. When trusted, X-Real-IP wins (nginx sets it to the real
// client); X-Forwarded-For's first entry is a fallback only.
func extractClientIP(r *http.Request, trusted map[string]struct{}) net.IP {
	peer := r.RemoteAddr
	if h, _, err := net.SplitHostPort(peer); err == nil {
		peer = h
	}

	if _, ok := trusted[normalizeIP(peer)]; ok {
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			if ip := net.ParseIP(strings.TrimSpace(realIP)); ip != nil {
				return ip
			}
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			first := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
			if ip := net.ParseIP(first); ip != nil {
				return ip
			}
		}
	}

	// Untrusted source (or no usable header): use the TCP peer address.
	return net.ParseIP(peer)
}

// buildTrustedProxySet normalises a list of trusted-proxy IPs into a lookup
// set keyed by canonical IP string.
func buildTrustedProxySet(proxies []string) map[string]struct{} {
	set := make(map[string]struct{}, len(proxies))
	for _, p := range proxies {
		if ip := normalizeIP(strings.TrimSpace(p)); ip != "" {
			set[ip] = struct{}{}
		}
	}
	return set
}

// normalizeIP canonicalises an IP literal (strips IPv6 brackets) so that
// "[::1]" and "::1" compare equal against the trusted-proxy set.
func normalizeIP(host string) string {
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return host
}

// parseAllowIPs parses a list of raw IP/CIDR strings into separate IP and CIDR slices.
// Entries containing "/" are parsed as CIDR; others are parsed as exact IPs.
// Returns an error if any entry is invalid.
func parseAllowIPs(raw []string) (ips []net.IP, nets []*net.IPNet, err error) {
	for _, entry := range raw {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			_, cidr, parseErr := net.ParseCIDR(entry)
			if parseErr != nil {
				return nil, nil, fmt.Errorf("invalid CIDR %q: %w", entry, parseErr)
			}
			nets = append(nets, cidr)
		} else {
			ip := net.ParseIP(entry)
			if ip == nil {
				return nil, nil, fmt.Errorf("invalid IP address %q", entry)
			}
			ips = append(ips, ip)
		}
	}
	return ips, nets, nil
}
