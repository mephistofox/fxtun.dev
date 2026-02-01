//go:build windows

package client

import (
	"fmt"
	"os"
)

func restartProcess(execPath string) error {
	proc, err := os.StartProcess(execPath, os.Args, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Env:   os.Environ(),
	})
	if err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	_ = proc.Release()
	os.Exit(0)
	return nil // unreachable
}
