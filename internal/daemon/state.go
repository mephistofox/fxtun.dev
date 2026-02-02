package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	PID       int       `json:"pid"`
	APIAddr   string    `json:"api_addr"`
	Server    string    `json:"server"`
	StartedAt time.Time `json:"started_at"`
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
	return os.WriteFile(path, data, 0o644)
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
