//go:build linux

package core

import (
	"net"
	"testing"

	"golang.org/x/sys/unix"
)

func TestTuneSessionConnSetsNotSentLowat(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	tuneSessionConn(conn)

	raw, err := conn.(*net.TCPConn).SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var got int
	var gerr error
	_ = raw.Control(func(fd uintptr) {
		got, gerr = unix.GetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT)
	})
	if gerr != nil {
		t.Fatal(gerr)
	}
	if got != 128<<10 {
		t.Fatalf("TCP_NOTSENT_LOWAT = %d, want %d", got, 128<<10)
	}
}
