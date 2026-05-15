package daemon

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// IsProcessAlive checks whether a process with the given PID is still running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	return strings.Contains(err.Error(), "operation not permitted")
}

// IsDaemonRunning loads the state file and verifies the daemon is reachable via HTTP.
// If the daemon is not reachable, it cleans up the stale state file.
func IsDaemonRunning(statePath string) (*State, bool) {
	st, err := LoadState(statePath)
	if err != nil {
		return nil, false
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/status", st.APIAddr))
	if err != nil {
		RemoveState(statePath)
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return st, true
	}

	RemoveState(statePath)
	return nil, false
}
