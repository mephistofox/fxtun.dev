package core

import (
	"io"
	"net"
	"time"
)

// HTTPIdleTimeout cuts a tunnelled HTTP exchange or WebSocket only when no byte
// has moved for this long. A limit on the whole request killed SSE, long
// downloads and WebSockets at 30–60 s. A variable so tests can shorten it.
var HTTPIdleTimeout = 120 * time.Second

// idleBody extends the visitor connection's read deadline on every read of the
// request body. It is installed after the headers are parsed, so the header
// timeout that guards against slowloris keeps its own deadline.
type idleBody struct {
	io.ReadCloser
	extend func(time.Time) error
}

func (b idleBody) Read(p []byte) (int, error) {
	_ = b.extend(time.Now().Add(HTTPIdleTimeout))
	return b.ReadCloser.Read(p)
}

// idleConn extends both of a hijacked connection's deadlines on every read and
// write. Idleness counts across both directions: a push-only WebSocket keeps
// the pending Read in the other goroutine alive, since a deadline change
// applies to I/O already blocked.
type idleConn struct{ net.Conn }

func (c idleConn) Read(p []byte) (int, error) {
	_ = c.Conn.SetDeadline(time.Now().Add(HTTPIdleTimeout))
	return c.Conn.Read(p)
}

func (c idleConn) Write(p []byte) (int, error) {
	_ = c.Conn.SetDeadline(time.Now().Add(HTTPIdleTimeout))
	return c.Conn.Write(p)
}

// idleStreamReader extends the tunnel stream's read deadline before every read,
// so a local app that goes silent mid-response is cut after HTTPIdleTimeout even
// though nothing is being written to the visitor meanwhile.
type idleStreamReader struct {
	r io.Reader
	c net.Conn
}

func (s idleStreamReader) Read(p []byte) (int, error) {
	_ = s.c.SetReadDeadline(time.Now().Add(HTTPIdleTimeout))
	return s.r.Read(p)
}
