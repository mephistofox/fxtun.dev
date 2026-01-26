package client

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

// dialLocalWithFallback attempts to connect to a local service with IPv4/IPv6 fallback.
// If localAddr is specified, it uses that address directly.
// If localAddr is empty, it tries 127.0.0.1 first, then [::1] (IPv6 localhost).
// This handles cases where dev servers may listen on IPv6 only (e.g., on Windows
// where localhost often resolves to ::1).
func dialLocalWithFallback(log zerolog.Logger, localAddr string, localPort int, timeout time.Duration) (net.Conn, error) {
	portStr := strconv.Itoa(localPort)

	// If explicit address is specified, use it directly without fallback
	if localAddr != "" {
		addr := net.JoinHostPort(localAddr, portStr)
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
		}
		return conn, nil
	}

	// Try IPv4 localhost first
	ipv4Addr := net.JoinHostPort("127.0.0.1", portStr)
	conn, err := net.DialTimeout("tcp", ipv4Addr, timeout)
	if err == nil {
		log.Debug().Str("addr", ipv4Addr).Msg("Connected to local service via IPv4")
		return conn, nil
	}
	ipv4Err := err

	// Fallback to IPv6 localhost
	ipv6Addr := net.JoinHostPort("::1", portStr)
	conn, err = net.DialTimeout("tcp", ipv6Addr, timeout)
	if err == nil {
		log.Debug().Str("addr", ipv6Addr).Msg("Connected to local service via IPv6")
		return conn, nil
	}

	// Both failed, return combined error
	return nil, fmt.Errorf("failed to connect to local service on port %d: IPv4 (%s): %v, IPv6 (%s): %v",
		localPort, ipv4Addr, ipv4Err, ipv6Addr, err)
}
