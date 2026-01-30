package gui

import (
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
		ClientVersion: info.ClientVersion,
		ServerVersion: info.ServerVersion,
		DownloadURL:   info.DownloadURL,
	}, nil
}

// DownloadUpdate downloads and installs the update.
func (s *UpdateService) DownloadUpdate(downloadURL string) error {
	return client.SelfUpdate(downloadURL)
}
