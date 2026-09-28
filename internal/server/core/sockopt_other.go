//go:build !linux && !darwin

package core

import (
	"errors"
	"net"
)

// setNotSentLowat: TCP_NOTSENT_LOWAT does not exist here (Windows).
func setNotSentLowat(*net.TCPConn) error { return errors.ErrUnsupported }
