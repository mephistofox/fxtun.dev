package server

import (
	"net"
	"sync"
	"time"
)

const defaultProxyBufSize = 128 * 1024 // 128KB default buffer for proxying

// proxyBufSize is set during server init based on config; defaults to defaultProxyBufSize.
var proxyBufSize = defaultProxyBufSize

// proxyBufPool is a shared pool of large buffers for io.CopyBuffer,
// reducing allocations and improving throughput over the default 32KB.
var proxyBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, proxyBufSize)
		return &buf
	},
}

// initProxyBufSize sets the proxy buffer pool size from config.
func initProxyBufSize(size int) {
	if size > 0 {
		proxyBufSize = size
	}
}

// defaultTCPBufSize is the default TCP socket buffer size (256KB).
const defaultTCPBufSize = 256 * 1024

// tcpBufSize is set during server init; defaults to defaultTCPBufSize.
var tcpBufSize = defaultTCPBufSize

// tuneTCPConn applies low-latency and high-throughput settings to a TCP connection.
func tuneTCPConn(conn net.Conn) {
	tc, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	_ = tc.SetNoDelay(true)
	_ = tc.SetKeepAlive(true)
	_ = tc.SetKeepAlivePeriod(30 * time.Second)
	_ = tc.SetReadBuffer(tcpBufSize)
	_ = tc.SetWriteBuffer(tcpBufSize)
}
