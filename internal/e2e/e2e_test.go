package e2e

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clientcore "github.com/mephistofox/fxtun.dev/internal/client/core"
	"github.com/mephistofox/fxtun.dev/internal/config"
)

// --- Test 1: HTTP Tunnel Full Lifecycle ---

func TestHTTPTunnelLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Start a local HTTP server that returns a known response
	localPort := getFreePort(t)
	localServer := &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", localPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-E2E-Test", "hello")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "e2e-response-body")
		}),
	}
	ln, err := net.Listen("tcp", localServer.Addr)
	require.NoError(t, err)
	go localServer.Serve(ln)
	t.Cleanup(func() { localServer.Close() })

	// Start harness
	h := NewHarness(t)
	h.Start()
	t.Cleanup(func() { h.Stop() })

	// Connect client with HTTP tunnel
	subdomain := "myapp"
	client := h.ConnectClient([]config.TunnelConfig{
		{
			Name:      "web",
			Type:      "http",
			LocalPort: localPort,
			Subdomain: subdomain,
		},
	})

	// Verify tunnel is registered on server
	stats := h.Server.GetStats()
	assert.Equal(t, 1, stats.ActiveClients, "expected 1 client")
	assert.Equal(t, 1, stats.HTTPTunnels, "expected 1 HTTP tunnel")

	// Make HTTP request through the tunnel
	httpClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/test-path", h.HTTPAddr), nil)
	require.NoError(t, err)
	req.Host = fmt.Sprintf("%s.%s", subdomain, testDomain)

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "hello", resp.Header.Get("X-E2E-Test"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "e2e-response-body", string(body))

	// Close client and verify tunnel no longer routes
	client.Close()
	time.Sleep(200 * time.Millisecond)

	req2, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/", h.HTTPAddr), nil)
	req2.Host = fmt.Sprintf("%s.%s", subdomain, testDomain)
	resp2, err := httpClient.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode, "tunnel should be gone after client close")
}

// --- Test 2: TCP Tunnel Echo ---

func TestTCPTunnelEcho(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Start a local TCP echo server
	localPort := getFreePort(t)
	echoLn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	require.NoError(t, err)
	t.Cleanup(func() { echoLn.Close() })

	go func() {
		for {
			conn, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()

	// Start harness
	h := NewHarness(t)
	h.Start()
	t.Cleanup(func() { h.Stop() })

	// Connect client with TCP tunnel
	remotePort := h.ServerCfg.Server.TCPPortRange.Min + 1
	_ = h.ConnectClient([]config.TunnelConfig{
		{
			Name:       "echo-tcp",
			Type:       "tcp",
			LocalPort:  localPort,
			RemotePort: remotePort,
		},
	})

	// Connect to the remote TCP port
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", remotePort), 5*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	// Send data and verify echo
	testData := "hello-tcp-tunnel\n"
	_, err = conn.Write([]byte(testData))
	require.NoError(t, err)

	buf := make([]byte, len(testData))
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = io.ReadFull(conn, buf)
	require.NoError(t, err)
	assert.Equal(t, testData, string(buf))
}

// --- Test 3: UDP Tunnel Echo ---

func TestUDPTunnelEcho(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Start a local UDP echo server
	localPort := getFreePort(t)
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", localPort))
	require.NoError(t, err)

	udpConn, err := net.ListenUDP("udp", udpAddr)
	require.NoError(t, err)
	t.Cleanup(func() { udpConn.Close() })

	go func() {
		buf := make([]byte, 65536)
		for {
			n, addr, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			udpConn.WriteToUDP(buf[:n], addr)
		}
	}()

	// Start harness
	h := NewHarness(t)
	h.Start()
	t.Cleanup(func() { h.Stop() })

	// Connect client with UDP tunnel
	remotePort := h.ServerCfg.Server.UDPPortRange.Min + 1
	_ = h.ConnectClient([]config.TunnelConfig{
		{
			Name:       "echo-udp",
			Type:       "udp",
			LocalPort:  localPort,
			RemotePort: remotePort,
		},
	})

	// Send UDP datagram and verify echo
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", remotePort))
	require.NoError(t, err)

	clientUDP, err := net.DialUDP("udp", nil, remoteUDPAddr)
	require.NoError(t, err)
	defer clientUDP.Close()

	testData := []byte("hello-udp-tunnel")
	_, err = clientUDP.Write(testData)
	require.NoError(t, err)

	buf := make([]byte, len(testData))
	clientUDP.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := clientUDP.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, string(testData), string(buf[:n]))
}

// --- Test 4: Authentication Failures ---

func TestAuthInvalidToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	h := NewHarness(t)
	h.Start()
	t.Cleanup(func() { h.Stop() })

	// Try connecting with invalid token
	cfg := &config.ClientConfig{
		Server: config.ClientServerSettings{
			Address:     h.ServerAddr,
			Token:       "sk_invalid_token_that_does_not_exist",
			Insecure:    true,
			Compression: false,
		},
		Reconnect: config.ReconnectSettings{Enabled: false},
		Inspect:   config.InspectSettings{Enabled: false},
	}

	client := newClientFromCfg(t, cfg, h.log)
	err := client.Connect()
	assert.Error(t, err, "expected error for invalid token")
	assert.Contains(t, err.Error(), "invalid token", "error should mention invalid token")
}

func TestAuthEmptyToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	h := NewHarness(t)
	h.Start()
	t.Cleanup(func() { h.Stop() })

	// Try connecting with empty token
	cfg := &config.ClientConfig{
		Server: config.ClientServerSettings{
			Address:     h.ServerAddr,
			Token:       "",
			Insecure:    true,
			Compression: false,
		},
		Reconnect: config.ReconnectSettings{Enabled: false},
		Inspect:   config.InspectSettings{Enabled: false},
	}

	client := newClientFromCfg(t, cfg, h.log)
	err := client.Connect()
	assert.Error(t, err, "expected error for empty token")
}

// --- Test 5: Multiple Tunnels Simultaneously ---

func TestMultipleTunnels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Start local HTTP server
	httpLocalPort := getFreePort(t)
	httpServer := &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", httpLocalPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "http-ok")
		}),
	}
	httpLn, err := net.Listen("tcp", httpServer.Addr)
	require.NoError(t, err)
	go httpServer.Serve(httpLn)
	t.Cleanup(func() { httpServer.Close() })

	// Start local TCP echo server
	tcpLocalPort := getFreePort(t)
	tcpLn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", tcpLocalPort))
	require.NoError(t, err)
	t.Cleanup(func() { tcpLn.Close() })
	go func() {
		for {
			conn, err := tcpLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()

	// Start local UDP echo server
	udpLocalPort := getFreePort(t)
	udpAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", udpLocalPort))
	udpConn, err := net.ListenUDP("udp", udpAddr)
	require.NoError(t, err)
	t.Cleanup(func() { udpConn.Close() })
	go func() {
		buf := make([]byte, 65536)
		for {
			n, addr, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			udpConn.WriteToUDP(buf[:n], addr)
		}
	}()

	// Start harness
	h := NewHarness(t)
	h.Start()
	t.Cleanup(func() { h.Stop() })

	tcpRemotePort := h.ServerCfg.Server.TCPPortRange.Min + 2
	udpRemotePort := h.ServerCfg.Server.UDPPortRange.Min + 2

	_ = h.ConnectClient([]config.TunnelConfig{
		{
			Name:      "multi-http",
			Type:      "http",
			LocalPort: httpLocalPort,
			Subdomain: "multi",
		},
		{
			Name:       "multi-tcp",
			Type:       "tcp",
			LocalPort:  tcpLocalPort,
			RemotePort: tcpRemotePort,
		},
		{
			Name:       "multi-udp",
			Type:       "udp",
			LocalPort:  udpLocalPort,
			RemotePort: udpRemotePort,
		},
	})

	// Verify server stats
	stats := h.Server.GetStats()
	assert.Equal(t, 1, stats.ActiveClients)
	assert.Equal(t, 3, stats.ActiveTunnels)
	assert.Equal(t, 1, stats.HTTPTunnels)
	assert.Equal(t, 1, stats.TCPTunnels)
	assert.Equal(t, 1, stats.UDPTunnels)

	// Test HTTP tunnel
	httpClient := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/", h.HTTPAddr), nil)
	req.Host = fmt.Sprintf("multi.%s", testDomain)
	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, "http-ok", string(body))

	// Test TCP tunnel
	tcpConn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", tcpRemotePort), 5*time.Second)
	require.NoError(t, err)
	defer tcpConn.Close()
	tcpConn.Write([]byte("tcp-data"))
	tcpBuf := make([]byte, 8)
	tcpConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	io.ReadFull(tcpConn, tcpBuf)
	assert.Equal(t, "tcp-data", string(tcpBuf))

	// Test UDP tunnel
	udpRemoteAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", udpRemotePort))
	udpClient, err := net.DialUDP("udp", nil, udpRemoteAddr)
	require.NoError(t, err)
	defer udpClient.Close()
	udpClient.Write([]byte("udp-data"))
	udpBuf := make([]byte, 8)
	udpClient.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := udpClient.Read(udpBuf)
	require.NoError(t, err)
	assert.Equal(t, "udp-data", string(udpBuf[:n]))
}

// --- Test 6: Concurrent HTTP Requests ---

func TestConcurrentHTTPRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Start local HTTP server that echoes a request counter
	var counter atomic.Int64
	localPort := getFreePort(t)
	localServer := &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", localPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := counter.Add(1)
			fmt.Fprintf(w, "request-%d", n)
		}),
	}
	ln, err := net.Listen("tcp", localServer.Addr)
	require.NoError(t, err)
	go localServer.Serve(ln)
	t.Cleanup(func() { localServer.Close() })

	// Start harness
	h := NewHarness(t)
	h.Start()
	t.Cleanup(func() { h.Stop() })

	subdomain := "concurrent"
	_ = h.ConnectClient([]config.TunnelConfig{
		{
			Name:      "concurrent-test",
			Type:      "http",
			LocalPort: localPort,
			Subdomain: subdomain,
		},
	})

	// Fire 50 concurrent HTTP requests
	const numRequests = 50
	var wg sync.WaitGroup
	errors := make(chan error, numRequests)
	successes := atomic.Int64{}

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: numRequests,
			MaxConnsPerHost:     numRequests,
		},
	}

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/", h.HTTPAddr), nil)
			req.Host = fmt.Sprintf("%s.%s", subdomain, testDomain)

			resp, err := httpClient.Do(req)
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errors <- err
				return
			}

			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("status %d, body: %s", resp.StatusCode, body)
				return
			}

			if !strings.HasPrefix(string(body), "request-") {
				errors <- fmt.Errorf("unexpected body: %s", body)
				return
			}

			successes.Add(1)
		}()
	}

	wg.Wait()
	close(errors)

	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	assert.Equal(t, int64(numRequests), successes.Load(),
		"all requests should succeed, errors: %v", errs)
	assert.Equal(t, int64(numRequests), counter.Load(),
		"local server should receive all requests")
}

// --- Test 7: Large Payload (10MB) ---

func TestLargePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Generate 10MB of random data
	const payloadSize = 10 * 1024 * 1024
	payload := make([]byte, payloadSize)
	_, err := rand.Read(payload)
	require.NoError(t, err)

	// Start local HTTP server that returns the large payload
	localPort := getFreePort(t)
	localServer := &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", localPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", payloadSize))
			w.WriteHeader(http.StatusOK)
			w.Write(payload)
		}),
	}
	ln, err := net.Listen("tcp", localServer.Addr)
	require.NoError(t, err)
	go localServer.Serve(ln)
	t.Cleanup(func() { localServer.Close() })

	// Start harness
	h := NewHarness(t)
	h.Start()
	t.Cleanup(func() { h.Stop() })

	subdomain := "bigdata"
	_ = h.ConnectClient([]config.TunnelConfig{
		{
			Name:      "large-payload",
			Type:      "http",
			LocalPort: localPort,
			Subdomain: subdomain,
		},
	})

	// Request the large payload through the tunnel
	httpClient := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/big", h.HTTPAddr), nil)
	req.Host = fmt.Sprintf("%s.%s", subdomain, testDomain)

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, payloadSize, len(body), "received payload size should match")
	assert.True(t, bytes.Equal(payload, body), "payload data should be identical")
}

// --- helper ---

func newClientFromCfg(t *testing.T, cfg *config.ClientConfig, log zerolog.Logger) *clientcore.Client {
	t.Helper()
	return clientcore.New(cfg, log)
}
