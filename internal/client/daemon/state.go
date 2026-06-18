package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	PID       int       `json:"pid"`
	APIAddr   string    `json:"api_addr"`
	Server    string    `json:"server"`
	Token     string    `json:"token"`
	StartedAt time.Time `json:"started_at"`
}

// GenerateToken returns a random 256-bit hex token used to authenticate local
// daemon API requests. It is persisted in the 0600 state file.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func DefaultStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".fxtunnel", "daemon.json")
}

func SaveState(path string, s *State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	// WriteFile does not tighten the mode of a pre-existing file; enforce 0600
	// explicitly since the state now holds the daemon API token.
	return os.Chmod(path, 0o600)
}

func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func RemoveState(path string) {
	_ = os.Remove(path)
}
