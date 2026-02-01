package client

import (
	"net"
	"sync"
	"time"
)

const proxyBufSize = 256 * 1024 // 256KB buffer for proxying

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
