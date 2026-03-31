package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	if !IsProcessAlive(os.Getpid()) {
		t.Fatal("expected current process to be alive")
	}
}

func TestIsProcessAlive_Dead(t *testing.T) {
	if IsProcessAlive(999999999) {
		t.Fatal("expected PID 999999999 to not be alive")
	}
}

func TestIsDaemonRunning_NoStateFile(t *testing.T) {
	st, ok := IsDaemonRunning(filepath.Join(t.TempDir(), "nonexistent.json"))
	if ok || st != nil {
		t.Fatal("expected nil, false for missing state file")
	}
}

func TestIsDaemonRunning_StaleState(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "daemon.json")

	err := SaveState(statePath, &State{
		PID:     99999,
		APIAddr: "127.0.0.1:19999",
		Server:  "example.com:4443",
	})
	if err != nil {
		t.Fatal(err)
	}

	st, ok := IsDaemonRunning(statePath)
	if ok || st != nil {
		t.Fatal("expected nil, false for stale state")
	}

	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatal("expected state file to be cleaned up")
	}
}
