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
func checkIPAllowlist(w http.ResponseWriter, r *http.Request, tunnel *Tunnel) bool {
	if len(tunnel.AllowedNets) == 0 && len(tunnel.AllowedIPs) == 0 {
		return true
	}
	clientIP := extractClientIP(r)
	if clientIP == nil || !isIPAllowed(clientIP, tunnel) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}
	return true
}

// extractClientIP extracts the client IP from the request.
// Priority: X-Real-IP -> X-Forwarded-For (first entry) -> RemoteAddr.
func extractClientIP(r *http.Request) net.IP {
	// Try X-Real-IP first
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		if ip := net.ParseIP(strings.TrimSpace(realIP)); ip != nil {
			return ip
		}
	}

	// Try X-Forwarded-For (first entry)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
		if ip := net.ParseIP(first); ip != nil {
			return ip
		}
	}

	// Fall back to RemoteAddr
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return net.ParseIP(host)
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
