package api

import (
	"net/http"
)

// VersionResponse represents the version endpoint response
type VersionResponse struct {
	ServerVersion string            `json:"server_version"`
	ClientVersion string            `json:"client_version"`
	Downloads     map[string]string `json:"downloads"`
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	downloads := map[string]string{
		"linux_amd64":   "/api/downloads/fxtunnel-linux-amd64",
		"linux_arm64":   "/api/downloads/fxtunnel-linux-arm64",
		"darwin_amd64":  "/api/downloads/fxtunnel-darwin-amd64",
		"darwin_arm64":  "/api/downloads/fxtunnel-darwin-arm64",
		"windows_amd64": "/api/downloads/fxtunnel-windows-amd64.exe",
	}

	s.respondJSON(w, http.StatusOK, VersionResponse{
		ServerVersion: s.version,
		ClientVersion: s.version,
		Downloads:     downloads,
	})
}
