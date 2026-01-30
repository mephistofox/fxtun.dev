package server

import (
	"net"
	"sync"
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
	tc.SetNoDelay(true)
	tc.SetReadBuffer(512 * 1024)  // 512KB read buffer
	tc.SetWriteBuffer(512 * 1024) // 512KB write buffer
}
