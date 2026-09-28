package core

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/inspect"
	"github.com/mephistofox/fxtun.dev/internal/protocol"
)

// An inspected streaming response must count as activity while it streams,
// not only once it ends, and its bytes must be counted once.
func TestAutoCloseSeesInspectedStream(t *testing.T) {
	const chunk, chunks = "data: tick\n\n", 8
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for i := 0; i < chunks; i++ {
			_, _ = io.WriteString(w, chunk)
			w.(http.Flusher).Flush()
			time.Sleep(200 * time.Millisecond)
		}
	}))
	defer local.Close()
	port := local.Listener.Addr().(*net.TCPAddr).Port

	c := New(&config.ClientConfig{}, zerolog.Nop())
	defer c.cancel()
	c.inspectMgr = inspect.NewManager(10, 1024)
	c.inspector = NewInspector(c.inspectMgr, "127.0.0.1:0", 1024, zerolog.Nop())
	tunnel := &ActiveTunnel{ID: "t", Config: config.TunnelConfig{Type: "http", LocalAddr: "127.0.0.1", LocalPort: port}}
	c.tunnels["t"] = tunnel

	var fired atomic.Bool
	timer := newAutoCloseTimer(500*time.Millisecond,
		func() int64 { return tunnel.BytesSent.Load() + tunnel.BytesReceived.Load() },
		func() { fired.Store(true) })
	defer timer.stop()

	server, stream := net.Pipe()
	defer server.Close()
	go c.handleStream(stream)
	go func() {
		_ = protocol.WriteStreamHeader(server, "t", "1.2.3.4:5")
		_, _ = io.WriteString(server, "GET /events HTTP/1.1\r\nHost: x\r\n\r\n")
	}()

	resp, err := http.ReadResponse(bufio.NewReader(server), nil)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body) // ~1.6s of streaming, over three idle periods
	require.NoError(t, err)
	require.Len(t, body, len(chunk)*chunks)
	require.False(t, fired.Load(), "auto-close fired while the inspected response was streaming")
	require.Equal(t, int64(len(body)), tunnel.BytesSent.Load(), "response bytes counted once")
}
