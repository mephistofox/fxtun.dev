// cmd/client/credentials.go
package main

import (
	"os"
	"path/filepath"

	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/keyring"
)

// checkAuth checks if user has valid credentials from keyring or ~/.fxtunnel/client.yaml.
// Returns (token, serverAddress, ok).
func checkAuth() (string, string, bool) {
	// 1. Check keyring
	kr := keyring.New()
	if creds, err := kr.LoadCredentials(); err == nil && creds.Token != "" {
		return creds.Token, creds.ServerAddress, true
	}

	// 2. Check ~/.fxtunnel/client.yaml
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", false
	}
	homeCfgPath := filepath.Join(home, ".fxtunnel", "client.yaml")
	if _, err := os.Stat(homeCfgPath); err != nil {
		return "", "", false
	}
	cfg, err := config.LoadClientConfig(homeCfgPath)
	if err != nil {
		return "", "", false
	}
	if cfg.Server.Token != "" {
		return cfg.Server.Token, cfg.Server.Address, true
	}

	return "", "", false
}
