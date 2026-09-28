package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// clientVersionFile names the marker the client sync drops next to the binaries
// it uploads, holding the release tag they were taken from.
const clientVersionFile = "VERSION"

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
		ClientVersion: s.servedClientVersion(),
		MinVersion:    s.minVersion,
		Downloads:     downloads,
	})
}

// servedClientVersion reports the release the binaries in the downloads
// directory came from, which is what `fxtunnel update` compares itself against.
// It cannot be the server's own build: a client release uploads new binaries
// without redeploying the server, so the server would keep advertising the tag
// it was compiled at and every client would be told it is already up to date.
//
// Falls back to the server's version when the marker is missing, so a
// hand-populated downloads directory still answers something sane.
func (s *Server) servedClientVersion() string {
	data, err := os.ReadFile(filepath.Join(s.downloadsPath, clientVersionFile))
	if err != nil {
		return s.version
	}
	if v := strings.TrimSpace(string(data)); v != "" {
		return v
	}
	return s.version
}
