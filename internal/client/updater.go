package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	ServerVersion string            `json:"server_version"`
	ClientVersion string            `json:"client_version"`
	Downloads     map[string]string `json:"downloads"`
	DownloadURL   string            `json:"-"` // resolved for current platform
}

// CheckUpdate checks the server for available updates.
// Returns nil if the client is up to date.
func CheckUpdate(serverAddr, currentVersion string) (*UpdateInfo, error) {
	// Build URL: strip port and use HTTP scheme for API
	scheme := "http"
	url := fmt.Sprintf("%s://%s/api/version", scheme, serverAddr)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url) //nolint:gosec // URL is constructed from user-provided server address
	if err != nil {
		return nil, fmt.Errorf("check update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check update: server returned %d", resp.StatusCode)
	}

	var info UpdateInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("check update: decode: %w", err)
	}

	// Compare versions
	if !isNewerVersion(info.ClientVersion, currentVersion) {
		return nil, nil // up to date
	}

	// Resolve download URL for current platform
	platform := runtime.GOOS + "_" + runtime.GOARCH
	if dlPath, ok := info.Downloads[platform]; ok {
		info.DownloadURL = fmt.Sprintf("%s://%s%s", scheme, serverAddr, dlPath)
	}

	return &info, nil
}

// SelfUpdate downloads a new binary and replaces the current executable.
func SelfUpdate(downloadURL string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURL) //nolint:gosec // URL is from trusted server
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download update: server returned %d", resp.StatusCode)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	// Write to temp file next to the executable
	tmpFile, err := os.CreateTemp("", "fxtunnel-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write update: %w", err)
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("chmod update: %w", err)
	}

	// Replace current binary
	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replace binary: %w", err)
	}

	return nil
}

// isNewerVersion returns true if remote is newer than local.
// Supports "vX.Y.Z" and "dev" formats.
func isNewerVersion(remote, local string) bool {
	remote = strings.TrimPrefix(remote, "v")
	local = strings.TrimPrefix(local, "v")

	if local == "dev" || local == "" {
		return false // dev builds don't auto-update
	}
	if remote == "dev" || remote == "" {
		return false
	}

	return remote != local && compareVersions(remote, local) > 0
}

// compareVersions compares two semver strings (without "v" prefix).
// Returns >0 if a > b, <0 if a < b, 0 if equal.
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		var av, bv int
		if i < len(aParts) {
			fmt.Sscanf(aParts[i], "%d", &av)
		}
		if i < len(bParts) {
			fmt.Sscanf(bParts[i], "%d", &bv)
		}
		if av != bv {
			return av - bv
		}
	}
	return 0
}
