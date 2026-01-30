package client

import (
	"net"
	"testing"
)

func TestTuneTCPConn_TCPConn(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	// Should not panic on a real TCP connection
	tuneTCPConn(conn)
}

func TestTuneTCPConn_NonTCPConn(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	// Should not panic on non-TCP connection (net.Pipe returns non-TCP)
	tuneTCPConn(c1)
}

func TestProxyBufPool(t *testing.T) {
	buf := proxyBufPool.Get().(*[]byte)
	if len(*buf) != proxyBufSize {
		t.Fatalf("expected buffer size %d, got %d", proxyBufSize, len(*buf))
	}
	proxyBufPool.Put(buf)

	// Get again — should reuse
	buf2 := proxyBufPool.Get().(*[]byte)
	if len(*buf2) != proxyBufSize {
		t.Fatalf("expected buffer size %d, got %d", proxyBufSize, len(*buf2))
	}
	proxyBufPool.Put(buf2)
}
