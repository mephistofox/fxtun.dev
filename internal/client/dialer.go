package client

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// resolvedAddrCache caches the resolved address (IPv4 or IPv6) per port
// so that subsequent connections skip the fallback probe.
var (
	resolvedAddrs   = make(map[int]string)
	resolvedAddrsMu sync.RWMutex
)

// dialLocalWithFallback attempts to connect to a local service with IPv4/IPv6 fallback.
// After the first successful connection, it caches the working address for the port
// so subsequent connections are instant.
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

	// Check cache first
	resolvedAddrsMu.RLock()
	cached, hasCached := resolvedAddrs[localPort]
	resolvedAddrsMu.RUnlock()

	if hasCached {
		conn, err := net.DialTimeout("tcp", cached, timeout)
		if err == nil {
			return conn, nil
		}
		// Cache miss (service restarted on different interface), clear and re-probe
		resolvedAddrsMu.Lock()
		delete(resolvedAddrs, localPort)
		resolvedAddrsMu.Unlock()
		log.Debug().Str("addr", cached).Msg("Cached address failed, re-probing")
	}

	// Probe: try IPv4 first, then IPv6
	ipv4Addr := net.JoinHostPort("127.0.0.1", portStr)
	conn, err := net.DialTimeout("tcp", ipv4Addr, timeout)
	if err == nil {
		log.Debug().Str("addr", ipv4Addr).Msg("Connected to local service via IPv4")
		resolvedAddrsMu.Lock()
		resolvedAddrs[localPort] = ipv4Addr
		resolvedAddrsMu.Unlock()
		return conn, nil
	}
	ipv4Err := err

	// Fallback to IPv6 localhost
	ipv6Addr := net.JoinHostPort("::1", portStr)
	conn, err = net.DialTimeout("tcp", ipv6Addr, timeout)
	if err == nil {
		log.Debug().Str("addr", ipv6Addr).Msg("Connected to local service via IPv6")
		resolvedAddrsMu.Lock()
		resolvedAddrs[localPort] = ipv6Addr
		resolvedAddrsMu.Unlock()
		return conn, nil
	}

	return nil, fmt.Errorf("failed to connect to local service on port %d: IPv4 (%s): %v, IPv6 (%s): %v",
		localPort, ipv4Addr, ipv4Err, ipv6Addr, err)
}
