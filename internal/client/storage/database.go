// Package storage provides SQLite-based persistence for the GUI client.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents the SQLite database connection
type Database struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS bundles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL CHECK(type IN ('http', 'tcp', 'udp')),
    local_port INTEGER NOT NULL,
    subdomain TEXT,
    remote_port INTEGER,
    auto_connect BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bundle_id INTEGER REFERENCES bundles(id) ON DELETE SET NULL,
    bundle_name TEXT,
    tunnel_type TEXT NOT NULL,
    local_port INTEGER NOT NULL,
    remote_addr TEXT,
    url TEXT,
    connected_at TIMESTAMP NOT NULL,
    disconnected_at TIMESTAMP,
    bytes_sent INTEGER DEFAULT 0,
    bytes_received INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_history_connected_at ON history(connected_at);
CREATE INDEX IF NOT EXISTS idx_history_bundle_id ON history(bundle_id);
`

// New creates a new database connection
func New(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Initialize schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Database{db: db}, nil
}

// NewDefault creates a database at the default location
func NewDefault() (*Database, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "data.db")
	return New(dbPath)
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// DB returns the underlying sql.DB instance
func (d *Database) DB() *sql.DB {
	return d.db
}

// getDataDir returns the platform-specific data directory
func getDataDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "fxTunnel"), nil

	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "fxTunnel"), nil

	default: // Linux and others
		// Check XDG_DATA_HOME first
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			return filepath.Join(xdgData, "fxtunnel"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "fxtunnel"), nil
	}
}
