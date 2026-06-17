package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/protocol"
)

// goodControlServer accepts one connection and completes the server side of the
// compression handshake, then holds the connection open until the test ends.
func goodControlServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Complete the server side of the handshake (compression disabled).
			if _, _, err := protocol.NegotiateCompression(conn, false, true); err != nil {
				conn.Close()
				continue
			}
			// Keep the connection open until the test finishes.
			go func(c net.Conn) {
				<-done
				c.Close()
			}(conn)
		}
	}()
	return ln.Addr().String(), func() {
		close(done)
		ln.Close()
		wg.Wait()
	}
}

// brokenControlServer accepts connections and immediately closes them, so the
// client's compression-response read fails fast with EOF (the cheap stand-in
// for a DPI-stalled endpoint, without waiting out the 10s handshake deadline).
func brokenControlServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }
}

// selfSignedTLS returns a tls.Config with a freshly generated self-signed cert.
func selfSignedTLS(t *testing.T) *tls.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "tunnel.test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"tunnel.test"},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
}

// goodTLSControlServer is goodControlServer wrapped in a real TLS listener,
// mirroring the server-side control_tls listener.
func goodTLSControlServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := tls.Listen("tcp", "127.0.0.1:0", selfSignedTLS(t))
	if err != nil {
		t.Fatalf("tls listen: %v", err)
	}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			if _, _, err := protocol.NegotiateCompression(conn, false, true); err != nil {
				conn.Close()
				continue
			}
			go func(c net.Conn) {
				<-done
				c.Close()
			}(conn)
		}
	}()
	return ln.Addr().String(), func() {
		close(done)
		ln.Close()
		wg.Wait()
	}
}

func TestConnectTransport_TLSEndpoint(t *testing.T) {
	tlsAddr, stop := goodTLSControlServer(t)
	defer stop()

	cfg := &config.ClientConfig{}
	cfg.Server.Address = tlsAddr
	cfg.Server.Insecure = false  // use TLS for the primary
	cfg.Server.TLSVerify = false // self-signed cert in test
	cfg.Server.Compression = false
	c := New(cfg, zerolog.Nop())
	defer c.cancel()

	conn, _, _, ep, err := c.connectTransport()
	if err != nil {
		t.Fatalf("connectTransport over TLS: %v", err)
	}
	defer conn.Close()
	if !ep.useTLS || ep.addr != tlsAddr {
		t.Fatalf("expected TLS endpoint %s, got %+v", tlsAddr, ep)
	}
}

func newTestClient(primary, fallback string) *Client {
	cfg := &config.ClientConfig{}
	cfg.Server.Address = primary
	cfg.Server.Insecure = true
	cfg.Server.FallbackAddress = fallback
	cfg.Server.FallbackInsecure = true
	cfg.Server.Compression = false
	return New(cfg, zerolog.Nop())
}

func TestConnectTransport_FallbackOnBrokenPrimary(t *testing.T) {
	brokenAddr, stopBroken := brokenControlServer(t)
	defer stopBroken()
	goodAddr, stopGood := goodControlServer(t)
	defer stopGood()

	c := newTestClient(brokenAddr, goodAddr)
	defer c.cancel()

	conn, _, _, ep, err := c.connectTransport()
	if err != nil {
		t.Fatalf("connectTransport: expected fallback success, got error: %v", err)
	}
	defer conn.Close()

	if ep.addr != goodAddr {
		t.Fatalf("expected to connect via fallback %s, got %s", goodAddr, ep.addr)
	}
}

func TestConnectTransport_PrimaryPreferred(t *testing.T) {
	goodAddr, stopGood := goodControlServer(t)
	defer stopGood()
	brokenAddr, stopBroken := brokenControlServer(t)
	defer stopBroken()

	c := newTestClient(goodAddr, brokenAddr)
	defer c.cancel()

	conn, _, _, ep, err := c.connectTransport()
	if err != nil {
		t.Fatalf("connectTransport: expected primary success, got error: %v", err)
	}
	defer conn.Close()

	if ep.addr != goodAddr {
		t.Fatalf("expected to connect via primary %s, got %s", goodAddr, ep.addr)
	}
}

func TestConnectTransport_AllEndpointsFail(t *testing.T) {
	brokenA, stopA := brokenControlServer(t)
	defer stopA()
	brokenB, stopB := brokenControlServer(t)
	defer stopB()

	c := newTestClient(brokenA, brokenB)
	defer c.cancel()

	if _, _, _, _, err := c.connectTransport(); err == nil {
		t.Fatal("expected error when all endpoints fail, got nil")
	}
}

// TestConnectTransport_FallbackTiming guards against a regression where a dead
// primary would make the whole connect wait out the full handshake deadline.
func TestConnectTransport_FallbackTiming(t *testing.T) {
	brokenAddr, stopBroken := brokenControlServer(t)
	defer stopBroken()
	goodAddr, stopGood := goodControlServer(t)
	defer stopGood()

	c := newTestClient(brokenAddr, goodAddr)
	defer c.cancel()

	start := time.Now()
	conn, _, _, _, err := c.connectTransport()
	if err != nil {
		t.Fatalf("connectTransport: %v", err)
	}
	defer conn.Close()
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("fallback took too long (%v); broken primary should fail fast", elapsed)
	}
}
