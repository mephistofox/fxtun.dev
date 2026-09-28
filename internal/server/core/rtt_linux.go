//go:build linux

package core

import (
	"crypto/tls"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

// sessionRTT returns the kernel's smoothed round-trip time of a client's
// session socket: the tunnel's share of a request's wait, measured for free.
func sessionRTT(conn net.Conn) (time.Duration, bool) {
	if tc, ok := conn.(*tls.Conn); ok {
		conn = tc.NetConn()
	}
	tc, ok := conn.(*net.TCPConn)
	if !ok {
		return 0, false
	}
	raw, err := tc.SyscallConn()
	if err != nil {
		return 0, false
	}
	var info *unix.TCPInfo
	var infoErr error
	if err := raw.Control(func(fd uintptr) {
		info, infoErr = unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO)
	}); err != nil || infoErr != nil {
		return 0, false
	}
	return time.Duration(info.Rtt) * time.Microsecond, true
}
