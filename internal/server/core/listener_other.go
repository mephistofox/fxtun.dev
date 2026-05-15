//go:build !linux

package core

import (
	"context"
	"net"
)

// newReusePortListener creates a standard TCP listener (SO_REUSEPORT not available).
func newReusePortListener(_ context.Context, addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}
