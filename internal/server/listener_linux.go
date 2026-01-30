//go:build linux

package server

import (
	"context"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

// newReusePortListener creates a TCP listener with SO_REUSEPORT enabled.
func newReusePortListener(ctx context.Context, addr string) (net.Listener, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}
	return lc.Listen(ctx, "tcp", addr)
}
