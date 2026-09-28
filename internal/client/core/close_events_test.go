package core

import (
	"encoding/json"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/protocol"
)

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A duplicate reply to a request whose buffered channel is already full must
// not block the message loop.
func TestHandleReply_DuplicateDoesNotBlock(t *testing.T) {
	c := New(&config.ClientConfig{}, zerolog.Nop())
	c.pendingRequests["r"] = make(chan any, 1)

	created := &protocol.TunnelCreatedMessage{Message: protocol.NewMessage(protocol.MsgTunnelCreated)}
	created.RequestID = "r"
	tErr := &protocol.TunnelErrorMessage{Message: protocol.NewMessage(protocol.MsgTunnelError)}
	tErr.RequestID = "r"

	done := make(chan struct{})
	go func() {
		c.handleTunnelCreated(mustJSON(t, created))
		c.handleTunnelCreated(mustJSON(t, created))
		c.handleTunnelError(mustJSON(t, tErr))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("duplicate reply blocked the message loop")
	}
}

// A timer close emits EventTunnelClosed locally (the server's reply may be
// lost with the connection), and the server's later reply does not emit it
// a second time.
func TestCloseTunnel_EmitsClosedOnce(t *testing.T) {
	c := New(&config.ClientConfig{}, zerolog.Nop())
	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()
	go func() { _, _ = io.Copy(io.Discard, remote) }()
	c.controlCodec = protocol.NewCodec(local, local)
	c.tunnels["t1"] = &ActiveTunnel{ID: "t1"}

	var closed atomic.Int32
	c.Events().Subscribe(func(e Event) {
		if e.Type == EventTunnelClosed {
			closed.Add(1)
		}
	})

	c.closeTunnel("t1")
	time.Sleep(200 * time.Millisecond) // handlers run in goroutines
	if n := closed.Load(); n != 1 {
		t.Fatalf("after closeTunnel: EventTunnelClosed emitted %d times, want 1", n)
	}

	reply := &protocol.TunnelClosedMessage{Message: protocol.NewMessage(protocol.MsgTunnelClosed), TunnelID: "t1"}
	c.handleTunnelClosed(mustJSON(t, reply))
	time.Sleep(200 * time.Millisecond)
	if n := closed.Load(); n != 1 {
		t.Fatalf("after server reply: EventTunnelClosed emitted %d times, want 1", n)
	}
}
