package core

import (
	"crypto/tls"
	"net"
	"sync"
	"time"
)

// proxyBufSize is the copy chunk, and each write becomes one yamux frame that
// other streams' frames wait behind: 256 KB held them ~0.28 s at 0.9 MB/s,
// 64 KB holds them ~0.07 s.
const proxyBufSize = 64 * 1024

// proxyBufPool is a shared pool of large buffers for io.CopyBuffer,
// reducing allocations and improving throughput over the default 32KB.
var proxyBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, proxyBufSize)
		return &buf
	},
}

// tuneTCPConn applies low-latency and high-throughput settings to a TCP connection.
func tuneTCPConn(conn net.Conn) {
	tc, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	_ = tc.SetNoDelay(true)
	_ = tc.SetKeepAlive(true)
	_ = tc.SetKeepAlivePeriod(30 * time.Second)
	_ = tc.SetReadBuffer(2 * 1024 * 1024)  // 2MB read buffer
	_ = tc.SetWriteBuffer(2 * 1024 * 1024) // 2MB write buffer
}

// tuneSessionConn tunes a socket that carries a yamux session to a client.
//
// A fixed 2 MB send buffer (4 MB after the kernel doubles it) let one download
// park 3-4 MB in the kernel queue, and every other stream's frames waited
// behind it: SSE events stalled up to 5.3 s on prod. With TCP_NOTSENT_LOWAT at
// 128 KiB the queue stayed under 567 KB, the worst SSE gap fell to 1.2 s
// (about one RTT) and download speed did not change, so the big buffer stays
// for bytes in flight. Without the option, leave the buffer to OS autotuning.
func tuneSessionConn(conn net.Conn) {
	tc, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	_ = tc.SetNoDelay(true)
	_ = tc.SetKeepAlive(true)
	_ = tc.SetKeepAlivePeriod(30 * time.Second)
	_ = tc.SetReadBuffer(2 * 1024 * 1024)
	if setNotSentLowat(tc) == nil {
		_ = tc.SetWriteBuffer(2 * 1024 * 1024)
	}
}

// sessionListener tunes each accepted socket as a session socket. It sits
// under tls.NewListener, whose *tls.Conn would hide the TCP socket.
type sessionListener struct{ net.Listener }

func (l sessionListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err == nil {
		tuneSessionConn(conn)
	}
	return conn, err
}

// controlTLSConfig is the TLS configuration shared by every control-plane
// listener, pinned to exactly 1.2 so no client can end up on 1.3, which some
// networks refuse to carry for the tunnel.
//
// This does NOT rescue already-installed clients: what matters is a ClientHello
// that offers 1.3, not the version the server selects, so a client still
// offering 0x0304 loses its session even though it settles on 1.2. The
// client-side cap in chromeSpecTLS12 is what actually helps; this cap only
// keeps the server from being the side that raises the version.
func controlTLSConfig(cert tls.Certificate) *tls.Config {
	return &tls.Config{ //nolint:gosec // G402: capping at 1.2 is the point — see above
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS12,
	}
}

// listenSessionTLS is tls.Listen with session tuning on the raw sockets.
func listenSessionTLS(addr string, cfg *tls.Config) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return tls.NewListener(sessionListener{l}, cfg), nil
}
