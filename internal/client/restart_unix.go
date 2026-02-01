//go:build !windows

package client

import (
	"fmt"
	"os"
	"syscall"
)

func restartProcess(execPath string) error {
	if err := syscall.Exec(execPath, os.Args, os.Environ()); err != nil { //nolint:gosec // execPath is our own binary from os.Executable()
		return fmt.Errorf("restart: %w", err)
	}
	return nil // unreachable
}
