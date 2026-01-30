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
// so that subsequent connections skip the probe entirely.
var (
	resolvedAddrs   = make(map[int]string)
	resolvedAddrsMu sync.RWMutex
)

// dialLocalWithFallback connects to a local service with IPv4/IPv6 support.
// On first call for a port, it tries IPv4 first (most common), then falls back
// to IPv6 with a short delay, caching the winner for instant subsequent connections.
func dialLocalWithFallback(log zerolog.Logger, localAddr string, localPort int, timeout time.Duration) (net.Conn, error) {
	portStr := strconv.Itoa(localPort)

	// If explicit address is specified, use it directly
	if localAddr != "" {
		addr := net.JoinHostPort(localAddr, portStr)
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
		}
		tuneTCPConn(conn)
		return conn, nil
	}

	// Check cache first
	resolvedAddrsMu.RLock()
	cached, hasCached := resolvedAddrs[localPort]
	resolvedAddrsMu.RUnlock()

	if hasCached {
		conn, err := net.DialTimeout("tcp", cached, timeout)
		if err == nil {
			tuneTCPConn(conn)
			return conn, nil
		}
		// Cache stale — clear and re-probe
		resolvedAddrsMu.Lock()
		delete(resolvedAddrs, localPort)
		resolvedAddrsMu.Unlock()
		log.Debug().Str("addr", cached).Msg("Cached address failed, re-probing")
	}

	// Happy Eyeballs style: try IPv4 first, start IPv6 after short delay.
	// Most local services listen on 127.0.0.1, so IPv4 wins almost always.
	ipv4Addr := net.JoinHostPort("127.0.0.1", portStr)
	ipv6Addr := net.JoinHostPort("::1", portStr)

	type dialResult struct {
		conn net.Conn
		addr string
		err  error
	}

	results := make(chan dialResult, 2)

	// Start IPv4 immediately
	go func() {
		conn, err := net.DialTimeout("tcp", ipv4Addr, timeout)
		results <- dialResult{conn, ipv4Addr, err}
	}()

	// Start IPv6 after 50ms delay (Happy Eyeballs), unless IPv4 already won
	go func() {
		time.Sleep(50 * time.Millisecond)
		conn, err := net.DialTimeout("tcp", ipv6Addr, timeout)
		results <- dialResult{conn, ipv6Addr, err}
	}()

	var firstErr error
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err == nil {
			// Winner — cache and return, close the other when it arrives
			resolvedAddrsMu.Lock()
			resolvedAddrs[localPort] = r.addr
			resolvedAddrsMu.Unlock()

			proto := "IPv4"
			if r.addr == ipv6Addr {
				proto = "IPv6"
			}
			log.Debug().Str("addr", r.addr).Msgf("Connected to local service via %s", proto)

			// Drain the other result and close if it connected
			if i == 0 {
				go func() {
					other := <-results
					if other.conn != nil {
						other.conn.Close()
					}
				}()
			}
			tuneTCPConn(r.conn)
			return r.conn, nil
		}
		if firstErr == nil {
			firstErr = r.err
		}
	}

	return nil, fmt.Errorf("failed to connect to local service on port %d: %v", localPort, firstErr)
}


// ProbeLocalAddress probes a local port to determine the correct address
// (IPv4 or IPv6) and caches it. Call this when a tunnel is created
// so the first real connection is instant.
func ProbeLocalAddress(log zerolog.Logger, localAddr string, localPort int) {
	if localAddr != "" {
		return // Explicit address, no need to probe
	}

	resolvedAddrsMu.RLock()
	_, hasCached := resolvedAddrs[localPort]
	resolvedAddrsMu.RUnlock()
	if hasCached {
		return // Already cached
	}

	conn, err := dialLocalWithFallback(log, "", localPort, 2*time.Second)
	if err != nil {
		log.Debug().Int("port", localPort).Msg("Pre-probe failed, will retry on first connection")
		return
	}
	conn.Close()
	log.Info().Int("port", localPort).Msg("Local address pre-probed successfully")
}
