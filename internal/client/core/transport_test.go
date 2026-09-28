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

	utls "github.com/refraction-networking/utls"
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
// for a stalled endpoint, without waiting out the 10s handshake deadline).
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

// tls13CapableServer accepts one connection over a listener that happily speaks
// TLS 1.3 and reports the version it ended up negotiating.
func tls13CapableServer(t *testing.T) (addr string, version <-chan uint16, stop func()) {
	t.Helper()
	cfg := selfSignedTLS(t)
	cfg.MinVersion = tls.VersionTLS12
	cfg.MaxVersion = tls.VersionTLS13
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatalf("tls listen: %v", err)
	}
	ch := make(chan uint16, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		tc := conn.(*tls.Conn)
		if err := tc.Handshake(); err != nil {
			return
		}
		ch <- tc.ConnectionState().Version
	}()
	return ln.Addr().String(), ch, func() { ln.Close() }
}

// TestDialEndpoint_PinsTLS12 pins the control-plane handshake to TLS 1.2 even
// when the server offers 1.3. On some networks a session whose ClientHello
// merely offers 1.3 goes silent right after a successful handshake, until
// keepalive times out.
func TestDialEndpoint_PinsTLS12(t *testing.T) {
	addr, serverVersion, stop := tls13CapableServer(t)
	defer stop()

	c := New(&config.ClientConfig{}, zerolog.Nop())
	defer c.cancel()

	conn, err := c.dialEndpoint(endpoint{addr: addr, useTLS: true, tlsVerify: false, serverName: "tunnel.test"})
	if err != nil {
		t.Fatalf("dialEndpoint: %v", err)
	}
	defer conn.Close()

	uconn, ok := conn.(*utls.UConn)
	if !ok {
		t.Fatalf("expected *utls.UConn, got %T", conn)
	}
	if got := uconn.ConnectionState().Version; got != tls.VersionTLS12 {
		t.Fatalf("client negotiated TLS version %#04x, want TLS 1.2 (%#04x)", got, tls.VersionTLS12)
	}

	select {
	case got := <-serverVersion:
		if got != tls.VersionTLS12 {
			t.Fatalf("server negotiated TLS version %#04x, want TLS 1.2 (%#04x)", got, tls.VersionTLS12)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server never reported a negotiated TLS version")
	}
}

// TestChromeSpecTLS12_DropsTLS13Extensions guards the ClientHello against being
// a fingerprint no browser produces. The Chrome preset carries key_share (with
// a post-quantum ML-KEM share), psk_key_exchange_modes, GREASE ECH and
// compress_certificate, all meaningless under TLS 1.2. Chrome with TLS 1.3
// disabled sends none of them, so leaving them in produces a unique JA4 —
// exactly what the uTLS layer exists to avoid. (Keeping them is not by itself
// what loses the session; that is governed by the offered version.)
func TestChromeSpecTLS12_DropsTLS13Extensions(t *testing.T) {
	spec, err := chromeSpecTLS12()
	if err != nil {
		t.Fatalf("chromeSpecTLS12: %v", err)
	}
	for _, ext := range spec.Extensions {
		switch ext.(type) {
		case *utls.KeyShareExtension:
			t.Errorf("ClientHello still carries key_share (TLS 1.3 only)")
		case *utls.PSKKeyExchangeModesExtension:
			t.Errorf("ClientHello still carries psk_key_exchange_modes (TLS 1.3 only)")
		case *utls.GREASEEncryptedClientHelloExtension:
			t.Errorf("ClientHello still carries GREASE ECH (TLS 1.3 only)")
		case *utls.UtlsCompressCertExtension:
			t.Errorf("ClientHello still carries compress_certificate (TLS 1.3 only)")
		}
	}
}

// A suppressed primary connects fine and dies ~40s later, so the reconnect loop
// would ride it forever and never reach the plaintext fallback. A session that
// short marks the primary as suppressed, and reconnects inside the penalty
// window try the fallback first.
func TestEndpoints_FallbackFirstWhilePrimaryPenalised(t *testing.T) {
	c := newTestClient("primary.example:443", "fallback.example:4443")
	defer c.cancel()

	if got := c.endpoints()[0].addr; got != "primary.example:443" {
		t.Fatalf("without a penalty the primary goes first, got %s", got)
	}

	c.primaryBadUntil = time.Now().Add(time.Minute)
	eps := c.endpoints()
	if eps[0].addr != "fallback.example:4443" {
		t.Fatalf("penalised: want fallback first, got %s", eps[0].addr)
	}
	if len(eps) != 2 || eps[1].addr != "primary.example:443" {
		t.Fatalf("penalised: primary must stay as second attempt, got %+v", eps)
	}
}

func TestEndpoints_PrimaryReturnsAfterPenaltyExpires(t *testing.T) {
	c := newTestClient("primary.example:443", "fallback.example:4443")
	defer c.cancel()

	c.primaryBadUntil = time.Now().Add(-time.Second) // window already over
	if got := c.endpoints()[0].addr; got != "primary.example:443" {
		t.Fatalf("expired penalty: want primary first, got %s", got)
	}
}

func TestPenalisePrimaryIfShortLived(t *testing.T) {
	tests := []struct {
		name        string
		active      string
		aliveFor    time.Duration
		wantPenalty bool
	}{
		{"short session on primary is suppressed", "primary.example:443", 40 * time.Second, true},
		{"long session on primary is a normal drop", "primary.example:443", 10 * time.Minute, false},
		{"short session on fallback does not blame the primary", "fallback.example:4443", 40 * time.Second, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient("primary.example:443", "fallback.example:4443")
			defer c.cancel()
			c.activeEndpoint = endpoint{addr: tt.active}
			c.transportStart = time.Now().Add(-tt.aliveFor)

			c.penalisePrimaryIfShortLived()

			if got := time.Now().Before(c.primaryBadUntil); got != tt.wantPenalty {
				t.Fatalf("penalty active = %v, want %v", got, tt.wantPenalty)
			}
		})
	}
}
