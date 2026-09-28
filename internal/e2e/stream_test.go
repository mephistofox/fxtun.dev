package e2e

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	clientcore "github.com/mephistofox/fxtun.dev/internal/client/core"
	"github.com/mephistofox/fxtun.dev/internal/config"
	servercore "github.com/mephistofox/fxtun.dev/internal/server/core"
	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

func streamApp(t *testing.T, handler http.HandlerFunc) int {
	t.Helper()
	port := getFreePort(t)
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: handler}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return port
}

// withIdle mutates a package global, so tests using it must not call t.Parallel().
func withIdle(t *testing.T, d time.Duration) {
	old := servercore.HTTPIdleTimeout
	servercore.HTTPIdleTimeout = d
	t.Cleanup(func() { servercore.HTTPIdleTimeout = old })
}

func tunnelGet(t *testing.T, h *E2EHarness, sub, path string) *http.Response {
	req, _ := http.NewRequest("GET", "http://"+h.HTTPAddr+path, nil)
	req.Host = sub + "." + testDomain
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// A stream that keeps sending outlives the idle timeout many times over.
func TestStreamOutlivesIdleTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	withIdle(t, time.Second)
	port := streamApp(t, func(w http.ResponseWriter, r *http.Request) {
		f := w.(http.Flusher)
		for i := 0; i < 20; i++ {
			fmt.Fprintf(w, "data: %d\n\n", i)
			f.Flush()
			time.Sleep(300 * time.Millisecond)
		}
	})
	h := NewHarness(t)
	h.Start()
	t.Cleanup(h.Stop)
	h.ConnectClient([]config.TunnelConfig{{Name: "s", Type: "http", LocalPort: port, Subdomain: "sse"}})

	resp := tunnelGet(t, h, "sse", "/")
	defer resp.Body.Close()
	n := 0
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "data:") {
			n++
		}
	}
	if n != 20 || sc.Err() != nil {
		t.Fatalf("got %d/20 events, err=%v", n, sc.Err())
	}
}

// A response that goes silent for longer than the idle timeout is cut.
func TestSilentStreamIsCut(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	withIdle(t, time.Second)
	port := streamApp(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		time.Sleep(5 * time.Second)
		fmt.Fprint(w, "data: late\n\n")
	})
	h := NewHarness(t)
	h.Start()
	t.Cleanup(h.Stop)
	h.ConnectClient([]config.TunnelConfig{{Name: "s", Type: "http", LocalPort: port, Subdomain: "quiet"}})

	start := time.Now()
	resp := tunnelGet(t, h, "quiet", "/")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "late") {
		t.Fatalf("silent stream was not cut: %q", body)
	}
	if d := time.Since(start); d > 4*time.Second {
		t.Fatalf("cut after %v, want about the idle timeout", d)
	}
}

// Slowloris protection on headers stays at 10 s regardless of the idle timeout.
func TestSlowHeadersAreCut(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	h := NewHarness(t)
	h.Start()
	t.Cleanup(h.Stop)
	c, err := net.Dial("tcp", h.HTTPAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	start := time.Now()
	for _, b := range []byte("GET / HTTP/1.1\r\nHost: x." + testDomain + "\r\nX-Slow: ") {
		if _, err := c.Write([]byte{b}); err != nil {
			break
		}
		time.Sleep(time.Second)
		if time.Since(start) > 15*time.Second {
			break
		}
	}
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := c.Read(make([]byte, 64))
	// The server must have given up: a 408 reply (n > 0) or a closed
	// connection. Our own read deadline firing means it was still waiting.
	if ne, ok := err.(net.Error); n == 0 && (err == nil || (ok && ne.Timeout())) {
		t.Fatalf("slow-header connection still open after %v (err=%v)", time.Since(start), err)
	}
}

// A push-mostly WebSocket (app pushes, visitor silent for longer than the idle
// timeout, then speaks) stays whole: idleness counts across both directions, so
// the visitor's late message still reaches the app.
func TestPushOnlyUpgradeOutlivesIdleTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	withIdle(t, time.Second)
	const pushes = 14 // one every 300 ms, about 4 s
	port := streamApp(t, func(w http.ResponseWriter, r *http.Request) {
		c, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer c.Close()
		fmt.Fprint(c, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		for i := 0; i < pushes; i++ {
			if _, err := c.Write([]byte{'x'}); err != nil {
				return
			}
			time.Sleep(300 * time.Millisecond)
		}
		// Echo the visitor's reply, then hang up.
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		b := make([]byte, 1)
		if _, err := c.Read(b); err == nil {
			_, _ = c.Write(b)
		}
	})
	h := NewHarness(t)
	h.Start()
	t.Cleanup(h.Stop)
	h.ConnectClient([]config.TunnelConfig{{Name: "s", Type: "http", LocalPort: port, Subdomain: "push"}})

	c, err := net.Dial("tcp", h.HTTPAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	fmt.Fprintf(c, "GET / HTTP/1.1\r\nHost: push.%s\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n", testDomain)
	_ = c.SetReadDeadline(time.Now().Add(15 * time.Second))
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, nil)
	if err != nil || resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("upgrade failed: %v %v", resp, err)
	}
	got := make([]byte, pushes)
	if n, err := io.ReadFull(br, got); err != nil {
		t.Fatalf("got %d/%d pushes, err=%v", n, pushes, err)
	}
	if _, err := c.Write([]byte{'q'}); err != nil {
		t.Fatalf("visitor write after silence: %v", err)
	}
	rest, err := io.ReadAll(br)
	if string(rest) != "q" {
		t.Fatalf("echo after silence = %q, want \"q\" (err=%v)", rest, err)
	}
}

// tcpTunnelAddr returns the local dial address of the client's TCP tunnel.
func tcpTunnelAddr(t *testing.T, c *clientcore.Client) string {
	t.Helper()
	for _, tun := range c.GetTunnels() {
		if tun.Config.Type != "tcp" {
			continue
		}
		_, port, err := net.SplitHostPort(tun.RemoteAddr)
		if err != nil {
			t.Fatalf("tunnel remote addr %q: %v", tun.RemoteAddr, err)
		}
		return net.JoinHostPort("127.0.0.1", port)
	}
	t.Fatal("no tcp tunnel")
	return ""
}

// A client that sends a request and closes its write side must still get the
// whole reply: the first EOF used to tear down both directions.
func TestTCPHalfClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	// Accept in a loop: the client's dialer may probe the port before the real
	// connection arrives.
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				data, _ := io.ReadAll(c) // read until the peer closes its write side
				time.Sleep(200 * time.Millisecond)
				_, _ = c.Write([]byte(strings.ToUpper(string(data))))
			}()
		}
	}()
	localPort := ln.Addr().(*net.TCPAddr).Port

	h := NewHarness(t)
	h.Start()
	t.Cleanup(h.Stop)
	client := h.ConnectClient([]config.TunnelConfig{{Name: "t", Type: "tcp", LocalPort: localPort}})
	remote := tcpTunnelAddr(t, client)

	c, err := net.Dial("tcp", remote)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, _ = c.Write([]byte("hello"))
	_ = c.(*net.TCPConn).CloseWrite()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	reply, err := io.ReadAll(c)
	if string(reply) != "HELLO" {
		t.Fatalf("reply %q, err=%v", reply, err)
	}
}

// Once the local app has closed, a visitor that stays silent and never closes
// must not hold the public connection open forever.
func TestTCPHalfCloseSilentVisitorIsCut(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	withIdle(t, time.Second)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_, _ = c.Write([]byte("hi"))
			_ = c.Close()
		}
	}()
	localPort := ln.Addr().(*net.TCPAddr).Port

	h := NewHarness(t)
	h.Start()
	t.Cleanup(h.Stop)
	client := h.ConnectClient([]config.TunnelConfig{{Name: "t", Type: "tcp", LocalPort: localPort}})

	c, err := net.Dial("tcp", tcpTunnelAddr(t, client))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if got, err := io.ReadAll(c); string(got) != "hi" {
		t.Fatalf("greeting %q, err=%v", got, err)
	}
	// The server closing its side fully shows up as a failing write (RST).
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := c.Write([]byte("x")); err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("public connection still open 5s after the local app closed")
}

// farRegistry places one subdomain on another node; everything else is local.
type farRegistry struct{ sub, node string }

func (f farRegistry) Register(store.TunnelEntry) error              { return nil }
func (f farRegistry) Unregister(string) error                       { return nil }
func (f farRegistry) Heartbeat(string) error                        { return nil }
func (f farRegistry) ListByUser(int64) ([]store.TunnelEntry, error) { return nil, nil }
func (f farRegistry) LookupBySubdomain(sub string) (*store.TunnelEntry, error) {
	if sub != f.sub {
		return nil, nil
	}
	return &store.TunnelEntry{Subdomain: sub, ServerID: f.node}, nil
}

// A stream proxied to another node on a reused keep-alive connection must not
// be cut by the write deadline the previous, local request left behind.
func TestRemoteStreamAfterLocalRequestOnSameConn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	withIdle(t, time.Second)
	const events = 10 // one every 300 ms, about 3 s
	nearPort := streamApp(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	farPort := streamApp(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i := 0; i < events; i++ {
			fmt.Fprintf(w, "data: %d\n\n", i)
			w.(http.Flusher).Flush()
			time.Sleep(300 * time.Millisecond)
		}
	})
	h := NewHarness(t)
	h.Server.SetTunnelRegistry(farRegistry{sub: "far", node: fmt.Sprintf("127.0.0.1:%d", farPort)})
	h.Start()
	t.Cleanup(h.Stop)
	h.ConnectClient([]config.TunnelConfig{{Name: "n", Type: "http", LocalPort: nearPort, Subdomain: "near"}})

	c, err := net.Dial("tcp", h.HTTPAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(15 * time.Second))
	br := bufio.NewReader(c)

	fmt.Fprintf(c, "GET / HTTP/1.1\r\nHost: near.%s\r\n\r\n", testDomain)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := io.ReadAll(resp.Body); string(b) != "ok" {
		t.Fatalf("local reply %q", b)
	}

	fmt.Fprintf(c, "GET / HTTP/1.1\r\nHost: far.%s\r\n\r\n", testDomain)
	resp, err = http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "data:") {
			n++
		}
	}
	if n != events || sc.Err() != nil {
		t.Fatalf("got %d/%d remote events, err=%v", n, events, sc.Err())
	}
}

// After the visitor leaves, a local app that only pushes and never reads must
// not be held open: the tunnel stops granting window, so the client's upload
// has to give up and close the local connection.
func TestTCPPushOnlyAppIsClosedAfterVisitorLeaves(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	old := clientcore.TCPWriteIdle
	clientcore.TCPWriteIdle = time.Second
	t.Cleanup(func() { clientcore.TCPWriteIdle = old })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	appDone := make(chan time.Time, 16)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				buf := make([]byte, 32*1024)
				for {
					if _, err := c.Write(buf); err != nil {
						select {
						case appDone <- time.Now():
						default:
						}
						return
					}
				}
			}()
		}
	}()
	localPort := ln.Addr().(*net.TCPAddr).Port

	h := NewHarness(t)
	h.Start()
	t.Cleanup(h.Stop)
	client := h.ConnectClient([]config.TunnelConfig{{Name: "t", Type: "tcp", LocalPort: localPort}})

	c, err := net.Dial("tcp", tcpTunnelAddr(t, client))
	if err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.ReadFull(c, make([]byte, 64*1024)); err != nil {
		t.Fatalf("visitor read: %v", err)
	}
	left := time.Now()
	_ = c.Close()

	// Earlier failures belong to the client dialer's port probes; skip them.
	timeout := time.After(10 * time.Second)
	for {
		select {
		case at := <-appDone:
			if at.After(left) {
				t.Logf("local app's writes failed %v after the visitor left", at.Sub(left))
				return
			}
		case <-timeout:
			t.Fatal("local app still writing 10s after the visitor left")
		}
	}
}
