package api

import (
	"net/http"
	"strings"
)

// VersionResponse represents the version endpoint response
type VersionResponse struct {
	ServerVersion string            `json:"server_version"`
	ClientVersion string            `json:"client_version"`
	MinVersion    string            `json:"min_version"`
	Downloads     map[string]string `json:"downloads"`
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	downloads := map[string]string{
		"linux_amd64":   "/api/downloads/cli-linux-amd64",
		"linux_arm64":   "/api/downloads/cli-linux-arm64",
		"darwin_amd64":  "/api/downloads/cli-darwin-amd64",
		"darwin_arm64":  "/api/downloads/cli-darwin-arm64",
		"windows_amd64": "/api/downloads/cli-windows-amd64",
	}

	// Desktop builds get their own "gui_<os>_<arch>" keys. Without them a GUI
	// client looking up its platform finds only a "cli-*" path and would install
	// the command-line binary over itself. Only builds actually present in the
	// downloads directory are advertised.
	_, _, guiClients := s.scanDownloads()
	for _, g := range guiClients {
		downloads[strings.ReplaceAll(g.Platform, "-", "_")] = g.URL
	}

	s.respondJSON(w, http.StatusOK, VersionResponse{
		ServerVersion: s.version,
		ClientVersion: s.version,
		MinVersion:    s.minVersion,
		Downloads:     downloads,
	})
}
