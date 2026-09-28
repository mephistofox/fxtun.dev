//go:build linux || darwin

package core

import (
	"net"

	"golang.org/x/sys/unix"
)

// setNotSentLowat keeps at most 128 KiB of not-yet-sent data in the kernel
// queue, so a yamux frame of another stream waits behind that much of a bulk
// stream instead of the whole send buffer.
func setNotSentLowat(tc *net.TCPConn) error {
	raw, err := tc.SyscallConn()
	if err != nil {
		return err
	}
	var serr error
	if err := raw.Control(func(fd uintptr) {
		serr = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT, 128<<10)
	}); err != nil {
		return err
	}
	return serr
}
