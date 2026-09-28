package core

import (
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

// tuneSessionConn tunes a socket that carries a yamux session to the server.
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
