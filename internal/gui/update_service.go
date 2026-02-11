package gui

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mephistofox/fxtunnel/internal/client"
)

// UpdateService provides update checking and downloading for the GUI.
type UpdateService struct {
	app *App
}

// NewUpdateService creates a new UpdateService.
func NewUpdateService(app *App) *UpdateService {
	return &UpdateService{app: app}
}

// UpdateInfo mirrors client.UpdateInfo for Wails bindings.
type UpdateInfo struct {
	Available     bool   `json:"available"`
	ForceUpdate   bool   `json:"force_update"`
	ClientVersion string `json:"client_version"`
	ServerVersion string `json:"server_version"`
	DownloadURL   string `json:"download_url"`
}

// CheckUpdate checks the server for available updates.
func (s *UpdateService) CheckUpdate() (*UpdateInfo, error) {
	addr := s.app.serverAddress
	if addr == "" {
		return &UpdateInfo{Available: false}, nil
	}

	info, err := client.CheckUpdate(addr, s.app.version)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return &UpdateInfo{Available: false}, nil
	}

	return &UpdateInfo{
		Available:     true,
		ForceUpdate:   client.IsVersionIncompatible(info.MinVersion, s.app.version),
		ClientVersion: info.ClientVersion,
		ServerVersion: info.ServerVersion,
		DownloadURL:   info.DownloadURL,
	}, nil
}

// DownloadUpdate downloads and installs the update.
func (s *UpdateService) DownloadUpdate(downloadURL string) error {
	u, err := url.Parse(downloadURL)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	// Only allow HTTPS downloads from trusted domains
	if u.Scheme != "https" {
		return fmt.Errorf("download URL must use HTTPS")
	}
	// Validate domain - must be from GitHub releases or project domain
	allowedHosts := []string{"github.com", "api.github.com", "objects.githubusercontent.com"}
	hostAllowed := false
	for _, h := range allowedHosts {
		if u.Host == h || strings.HasSuffix(u.Host, "."+h) {
			hostAllowed = true
			break
		}
	}
	if !hostAllowed {
		return fmt.Errorf("download URL host not allowed: %s", u.Host)
	}
	return client.SelfUpdate(downloadURL)
}

// ApplyUpdateAndRestart downloads the update and restarts the process.
func (s *UpdateService) ApplyUpdateAndRestart(downloadURL string) error {
	return client.SelfUpdateAndRestart(downloadURL)
}
