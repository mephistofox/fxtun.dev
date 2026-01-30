package client

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestDialLocalWithFallback_ExplicitAddr(t *testing.T) {
	// Start a local TCP listener
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	log := zerolog.Nop()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	conn, err := dialLocalWithFallback(log, "127.0.0.1", port, 2*time.Second)
	if err != nil {
		t.Fatalf("expected successful dial, got: %v", err)
	}
	conn.Close()
}

func TestDialLocalWithFallback_AutoDetect(t *testing.T) {
	// Clear cache to force probing
	resolvedAddrsMu.Lock()
	resolvedAddrs = make(map[int]string)
	resolvedAddrsMu.Unlock()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	log := zerolog.Nop()

	// Accept two connections (probe may try both IPv4 and IPv6)
	go func() {
		for i := 0; i < 3; i++ {
			conn, _ := ln.Accept()
			if conn != nil {
				conn.Close()
			}
		}
	}()

	conn, err := dialLocalWithFallback(log, "", port, 2*time.Second)
	if err != nil {
		t.Fatalf("expected successful dial, got: %v", err)
	}
	conn.Close()

	// Verify address was cached
	resolvedAddrsMu.RLock()
	cached, ok := resolvedAddrs[port]
	resolvedAddrsMu.RUnlock()
	if !ok {
		t.Fatal("expected address to be cached")
	}
	if cached == "" {
		t.Fatal("cached address should not be empty")
	}
}

func TestDialLocalWithFallback_CachedAddr(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	// Pre-populate cache
	resolvedAddrsMu.Lock()
	resolvedAddrs[port] = ln.Addr().String()
	resolvedAddrsMu.Unlock()

	defer func() {
		resolvedAddrsMu.Lock()
		delete(resolvedAddrs, port)
		resolvedAddrsMu.Unlock()
	}()

	log := zerolog.Nop()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	conn, err := dialLocalWithFallback(log, "", port, 2*time.Second)
	if err != nil {
		t.Fatalf("expected successful dial from cache, got: %v", err)
	}
	conn.Close()
}

func TestDialLocalWithFallback_StaleCache(t *testing.T) {
	// Clear cache
	resolvedAddrsMu.Lock()
	resolvedAddrs = make(map[int]string)
	resolvedAddrsMu.Unlock()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	// Set stale cache pointing to wrong port
	resolvedAddrsMu.Lock()
	resolvedAddrs[port] = "127.0.0.1:1"
	resolvedAddrsMu.Unlock()

	log := zerolog.Nop()

	go func() {
		for i := 0; i < 3; i++ {
			conn, _ := ln.Accept()
			if conn != nil {
				conn.Close()
			}
		}
	}()

	conn, err := dialLocalWithFallback(log, "", port, 2*time.Second)
	if err != nil {
		t.Fatalf("expected fallback after stale cache, got: %v", err)
	}
	conn.Close()
}

func TestDialLocalWithFallback_NoListener(t *testing.T) {
	resolvedAddrsMu.Lock()
	resolvedAddrs = make(map[int]string)
	resolvedAddrsMu.Unlock()

	log := zerolog.Nop()
	_, err := dialLocalWithFallback(log, "", 1, 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when nothing is listening")
	}
}

func TestDialLocalWithFallback_ExplicitAddr_NoListener(t *testing.T) {
	log := zerolog.Nop()
	_, err := dialLocalWithFallback(log, "127.0.0.1", 1, 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for explicit addr with no listener")
	}
}

func TestProbeLocalAddress_Explicit(t *testing.T) {
	log := zerolog.Nop()
	// Should return immediately without probing
	ProbeLocalAddress(log, "127.0.0.1", 9999)
}

func TestProbeLocalAddress_AlreadyCached(t *testing.T) {
	resolvedAddrsMu.Lock()
	resolvedAddrs[55555] = "127.0.0.1:55555"
	resolvedAddrsMu.Unlock()

	defer func() {
		resolvedAddrsMu.Lock()
		delete(resolvedAddrs, 55555)
		resolvedAddrsMu.Unlock()
	}()

	log := zerolog.Nop()
	ProbeLocalAddress(log, "", 55555)
}

func TestDialLocalWithFallback_Concurrent(t *testing.T) {
	resolvedAddrsMu.Lock()
	resolvedAddrs = make(map[int]string)
	resolvedAddrsMu.Unlock()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	log := zerolog.Nop()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := dialLocalWithFallback(log, "", port, 2*time.Second)
			if err != nil {
				t.Errorf("concurrent dial failed: %v", err)
				return
			}
			conn.Close()
		}()
	}
	wg.Wait()
}
