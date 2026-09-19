package gui

import (
	"fmt"
	"runtime"
	"strings"

	client "github.com/mephistofox/fxtunnel/internal/client/core"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
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

// guiDownloadURL resolves the desktop build for the running platform. The
// server's download map also carries the CLI paths, and the GUI must never take
// one of those: installing it over os.Executable() replaces the desktop app with
// the command-line binary. Returns "" when no desktop build is published for
// this platform.
func guiDownloadURL(info *client.UpdateInfo) string {
	if info == nil || info.ServerHost == "" {
		return ""
	}
	path, ok := info.Downloads["gui_"+runtime.GOOS+"_"+runtime.GOARCH]
	if !ok || path == "" {
		return ""
	}
	return "https://" + info.ServerHost + path
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
		DownloadURL:   guiDownloadURL(info),
	}, nil
}

// DownloadUpdate opens the new desktop build in the user's browser.
//
// The GUI deliberately does not replace its own executable: the only artifact
// the update check approves for installation is the CLI binary, so a self-update
// here would leave the user with the command-line tool in place of the app.
func (s *UpdateService) DownloadUpdate(downloadURL string) error {
	if downloadURL == "" {
		host, _, _ := strings.Cut(s.app.serverAddress, ":")
		return fmt.Errorf("no desktop build published for %s/%s, download it from https://%s/downloads",
			runtime.GOOS, runtime.GOARCH, host)
	}

	host, _, _ := strings.Cut(s.app.serverAddress, ":")
	if err := client.ValidateUpdateURL(downloadURL, host); err != nil {
		return err
	}
	if s.app.ctx == nil {
		return fmt.Errorf("open %s to download the update", downloadURL)
	}

	wailsRuntime.BrowserOpenURL(s.app.ctx, downloadURL)
	return nil
}

// ApplyUpdateAndRestart opens the new desktop build in the browser. The GUI
// cannot swap its own executable safely (see DownloadUpdate), so the user
// installs the downloaded build and restarts the app themselves.
func (s *UpdateService) ApplyUpdateAndRestart(downloadURL string) error {
	return s.DownloadUpdate(downloadURL)
}
