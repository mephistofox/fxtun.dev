//go:build !linux

package core

import (
	"net"
	"time"
)

// sessionRTT reads TCP_INFO, which is linux-only; elsewhere the wait is not split.
func sessionRTT(net.Conn) (time.Duration, bool) { return 0, false }
