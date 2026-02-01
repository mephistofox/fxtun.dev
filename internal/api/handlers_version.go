package api

import (
	"net/http"
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

	s.respondJSON(w, http.StatusOK, VersionResponse{
		ServerVersion: s.version,
		ClientVersion: s.version,
		MinVersion:    s.minVersion,
		Downloads:     downloads,
	})
}
