package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/yamux"

	"github.com/mephistofox/fxtunnel/internal/protocol"
)

// benchEnv sets up a full tunnel environment for benchmarking:
// server + yamux client session + authenticated tunnel + local echo server.
// Returns the HTTP address to send requests through the tunnel and the direct address.
type benchEnv struct {
	srv          *Server
	session      *yamux.Session
	tunnelID     string
	httpListener net.Listener
	echoAddr     string // direct address of the echo server
	tunnelHost   string // Host header for tunnel routing
}

func newBenchEnv(b *testing.B) *benchEnv {
	b.Helper()

	// --- Server setup ---
	srv, _, rawToken := testSetup(&testing.T{})

	// Start HTTP listener on random port
	httpLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("listen http: %v", err)
	}
	srv.httpListener = httpLn
	srv.httpRouter = NewHTTPRouter(srv, srv.log)

	// Serve HTTP via http.Server (supports keep-alive)
	httpServer := &http.Server{Handler: srv.httpRouter} //nolint:gosec // benchmark test
	srv.httpServer = httpServer
	go func() { _ = httpServer.Serve(httpLn) }()

	// --- Yamux client session via net.Pipe ---
	clientConn, serverConn := net.Pipe()
	srv.wg.Add(1)
	go srv.handleControlConnection(serverConn)

	// Compression handshake (client side, no compression in benchmarks)
	rwc, _, err := protocol.NegotiateCompression(clientConn, false, false)
	if err != nil {
		b.Fatalf("NegotiateCompression: %v", err)
	}

	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = false
	yamuxCfg.MaxStreamWindowSize = 4 * 1024 * 1024
	session, err := yamux.Client(rwc, yamuxCfg)
	if err != nil {
		b.Fatalf("yamux.Client: %v", err)
	}

	// --- Auth ---
	controlStream, err := session.Open()
	if err != nil {
		b.Fatalf("open control: %v", err)
	}
	codec := protocol.NewCodec(controlStream, controlStream)

	authMsg := &protocol.AuthMessage{
		Message: protocol.NewMessage(protocol.MsgAuth),
		Token:   rawToken,
	}
	if err := codec.Encode(authMsg); err != nil {
		b.Fatalf("encode auth: %v", err)
	}
	var authResult protocol.AuthResultMessage
	if err := codec.Decode(&authResult); err != nil {
		b.Fatalf("decode auth: %v", err)
	}
	if !authResult.Success {
		b.Fatalf("auth failed: %s", authResult.Error)
	}

	// --- Create HTTP tunnel ---
	tunnelReq := &protocol.TunnelRequestMessage{
		Message:    protocol.NewMessage(protocol.MsgTunnelRequest),
		TunnelType: protocol.TunnelHTTP,
		Name:       "bench",
		Subdomain:  "bench",
		LocalPort:  0, // will be set by the client-side handler
	}
	tunnelReq.RequestID = "bench-1"
	if err := codec.Encode(tunnelReq); err != nil {
		b.Fatalf("encode tunnel req: %v", err)
	}
	var created protocol.TunnelCreatedMessage
	if err := codec.Decode(&created); err != nil {
		b.Fatalf("decode tunnel created: %v", err)
	}

	// --- Local echo HTTP server ---
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("listen echo: %v", err)
	}
	echoMux := http.NewServeMux()
	echoMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		_, _ = io.Copy(w, r.Body)
		_ = r.Body.Close()
	})
	echoMux.HandleFunc("/big", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(200)
		buf := make([]byte, 64*1024)
		for i := range buf {
			buf[i] = byte(i)
		}
		// Write 1MB
		for i := 0; i < 16; i++ {
			_, _ = w.Write(buf)
		}
	})
	echoServer := &http.Server{Handler: echoMux} //nolint:gosec // benchmark test
	go func() { _ = echoServer.Serve(echoLn) }()

	// --- Client-side stream handler (simulates tunnel client) ---
	go func() {
		for {
			stream, err := session.Accept()
			if err != nil {
				return
			}
			go handleBenchStream(stream, echoLn.Addr().String())
		}
	}()

	// Wait for everything to settle
	time.Sleep(10 * time.Millisecond)

	return &benchEnv{
		srv:          srv,
		session:      session,
		tunnelID:     created.TunnelID,
		httpListener: httpLn,
		echoAddr:     echoLn.Addr().String(),
		tunnelHost:   "bench.test.local",
	}
}

func (e *benchEnv) close() {
	e.session.Close()
	e.httpListener.Close()
	e.srv.cancel()
}

// handleBenchStream simulates the client-side stream handler:
// reads NewConnectionMessage, then proxies to local echo server.
func handleBenchStream(stream net.Conn, echoAddr string) {
	defer stream.Close()

	streamCodec := protocol.NewCodec(stream, stream)
	var msg protocol.NewConnectionMessage
	if err := streamCodec.Decode(&msg); err != nil {
		return
	}

	local, err := net.DialTimeout("tcp", echoAddr, 2*time.Second)
	if err != nil {
		return
	}
	defer local.Close()

	if tc, ok := local.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
	}

	// Bidirectional proxy
	done := make(chan struct{}, 2)
	go func() {
		buf := make([]byte, 256*1024)
		_, _ = io.CopyBuffer(local, stream, buf)
		done <- struct{}{}
	}()
	go func() {
		buf := make([]byte, 256*1024)
		_, _ = io.CopyBuffer(stream, local, buf)
		done <- struct{}{}
	}()
	<-done
	local.Close()
	stream.Close()
	<-done
}

// tunnelHTTPAddr returns the address of the HTTP listener for tunnel requests.
func (e *benchEnv) tunnelHTTPAddr() string {
	return e.httpListener.Addr().String()
}

// BenchmarkLatency_Direct measures HTTP request latency directly to echo server.
func BenchmarkLatency_Direct(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			DisableKeepAlives:   false,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get("http://" + env.echoAddr + "/")
		if err != nil {
			b.Fatalf("direct request: %v", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkLatency_Tunnel measures HTTP request latency through the tunnel.
func BenchmarkLatency_Tunnel(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close()

	// Custom transport that connects to tunnel HTTP listener
	// but sets Host header for routing
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp", env.tunnelHTTPAddr())
			},
			DisableKeepAlives: true, // each request = new tunnel stream
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "http://"+env.tunnelHost+"/", nil)
		req.Header.Set("X-FxTunnel-Skip-Warning", "1") // skip interstitial
		resp, err := client.Do(req)
		if err != nil {
			b.Fatalf("tunnel request: %v", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkThroughput_Direct measures throughput directly to echo server.
func BenchmarkThroughput_Direct(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}

	b.SetBytes(1024 * 1024) // 1MB per op
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get("http://" + env.echoAddr + "/big")
		if err != nil {
			b.Fatalf("direct request: %v", err)
		}
		n, _ := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if n == 0 {
			b.Fatal("empty response")
		}
	}
}

// BenchmarkThroughput_Tunnel measures throughput through the tunnel.
func BenchmarkThroughput_Tunnel(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp", env.tunnelHTTPAddr())
			},
			DisableKeepAlives: true,
		},
	}

	b.SetBytes(1024 * 1024) // 1MB per op
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "http://"+env.tunnelHost+"/big", nil)
		req.Header.Set("X-FxTunnel-Skip-Warning", "1")
		resp, err := client.Do(req)
		if err != nil {
			b.Fatalf("tunnel request: %v", err)
		}
		n, _ := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if n == 0 {
			b.Fatal("empty response")
		}
	}
}

// BenchmarkConcurrentLatency_Tunnel measures latency under concurrent load.
func BenchmarkConcurrentLatency_Tunnel(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("tcp", env.tunnelHTTPAddr())
				},
				DisableKeepAlives: true,
			},
		}
		for pb.Next() {
			req, _ := http.NewRequest("GET", "http://"+env.tunnelHost+"/", nil)
			req.Header.Set("X-FxTunnel-Skip-Warning", "1")
			resp, err := client.Do(req)
			if err != nil {
				b.Errorf("tunnel request: %v", err)
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkRawProxy measures raw bidirectional data transfer through yamux
// without HTTP parsing overhead — pure proxy speed.
func BenchmarkRawProxy(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close()

	payload := make([]byte, 4096)
	for i := range payload {
		payload[i] = byte(i)
	}

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Connect through tunnel HTTP port
		conn, err := net.Dial("tcp", env.tunnelHTTPAddr())
		if err != nil {
			b.Fatalf("dial: %v", err)
		}

		// Send HTTP request with body
		req := fmt.Sprintf("POST / HTTP/1.1\r\nHost: %s\r\nContent-Length: %d\r\nX-FxTunnel-Skip-Warning: 1\r\nConnection: close\r\n\r\n",
			env.tunnelHost, len(payload))
		_, _ = conn.Write([]byte(req))
		_, _ = conn.Write(payload)

		// Read response
		_, _ = io.Copy(io.Discard, conn)
		conn.Close()
	}
}

// TestBenchEnvSetup verifies the benchmark environment works correctly.
func TestBenchEnvSetup(t *testing.T) {
	env := newBenchEnv(&testing.B{})
	defer env.close()

	// Test direct
	resp, err := http.Get("http://" + env.echoAddr + "/")
	if err != nil {
		t.Fatalf("direct: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("direct status: %d", resp.StatusCode)
	}

	// Test tunnel
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp", env.tunnelHTTPAddr())
			},
			DisableKeepAlives: true,
		},
	}
	req, _ := http.NewRequest("GET", "http://"+env.tunnelHost+"/", nil)
	req.Header.Set("X-FxTunnel-Skip-Warning", "1")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("tunnel: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("tunnel status: %d", resp.StatusCode)
	}
	t.Logf("Direct: %s, Tunnel HTTP: %s", env.echoAddr, env.tunnelHTTPAddr())
}

// BenchmarkParallelThroughput measures aggregate throughput with concurrent connections.
func BenchmarkParallelThroughput(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close()

	b.SetBytes(1024 * 1024)
	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return net.Dial("tcp", env.tunnelHTTPAddr())
					},
					DisableKeepAlives: true,
				},
			}
			req, _ := http.NewRequest("GET", "http://"+env.tunnelHost+"/big", nil)
			req.Header.Set("X-FxTunnel-Skip-Warning", "1")
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
	wg.Wait()
}
