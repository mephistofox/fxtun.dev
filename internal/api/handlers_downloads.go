package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtunnel/internal/api/dto"
)

//go:embed install.sh
var installScriptTmpl string

var installTmpl = template.Must(template.New("install").Parse(installScriptTmpl))

// Platform information for downloads
type platformInfo struct {
	Filename   string
	OS         string
	Arch       string
	ClientType string // "cli" or "gui"
}

// CLI client platforms
var cliPlatforms = map[string]platformInfo{
	"cli-linux-amd64":   {Filename: "fxtunnel-linux-amd64", OS: "Linux", Arch: "amd64", ClientType: "cli"},
	"cli-linux-arm64":   {Filename: "fxtunnel-linux-arm64", OS: "Linux", Arch: "arm64", ClientType: "cli"},
	"cli-darwin-amd64":  {Filename: "fxtunnel-darwin-amd64", OS: "macOS", Arch: "amd64", ClientType: "cli"},
	"cli-darwin-arm64":  {Filename: "fxtunnel-darwin-arm64", OS: "macOS", Arch: "arm64", ClientType: "cli"},
	"cli-windows-amd64": {Filename: "fxtunnel-windows-amd64.exe", OS: "Windows", Arch: "amd64", ClientType: "cli"},
}

// GUI client platforms
var guiPlatforms = map[string]platformInfo{
	"gui-linux-amd64":   {Filename: "fxtunnel-gui-linux-amd64", OS: "Linux", Arch: "amd64", ClientType: "gui"},
	"gui-windows-amd64": {Filename: "fxtunnel-gui-windows-amd64.exe", OS: "Windows", Arch: "amd64", ClientType: "gui"},
}

// All platforms combined for download handler
var platforms = func() map[string]platformInfo {
	all := make(map[string]platformInfo)
	for k, v := range cliPlatforms {
		all[k] = v
	}
	for k, v := range guiPlatforms {
		all[k] = v
	}
	return all
}()

// handleListDownloads returns a list of available client downloads
func (s *Server) handleListDownloads(w http.ResponseWriter, r *http.Request) {
	var cliClients []*dto.DownloadDTO
	var guiClients []*dto.DownloadDTO

	// Collect CLI clients
	for platform, info := range cliPlatforms {
		filePath := filepath.Join(s.downloadsPath, info.Filename)
		stat, err := os.Stat(filePath)
		if err != nil {
			continue // Skip if file doesn't exist
		}
		cliClients = append(cliClients, &dto.DownloadDTO{
			Platform:   platform,
			OS:         info.OS,
			Arch:       info.Arch,
			Size:       stat.Size(),
			URL:        "/api/downloads/" + platform,
			ClientType: "cli",
		})
	}

	// Collect GUI clients
	for platform, info := range guiPlatforms {
		filePath := filepath.Join(s.downloadsPath, info.Filename)
		stat, err := os.Stat(filePath)
		if err != nil {
			continue // Skip if file doesn't exist
		}
		guiClients = append(guiClients, &dto.DownloadDTO{
			Platform:   platform,
			OS:         info.OS,
			Arch:       info.Arch,
			Size:       stat.Size(),
			URL:        "/api/downloads/" + platform,
			ClientType: "gui",
		})
	}

	// Combine all for backwards compatibility
	allClients := append(cliClients, guiClients...)

	s.respondJSON(w, http.StatusOK, dto.DownloadsListResponse{
		Clients: allClients,
		CLI:     cliClients,
		GUI:     guiClients,
	})
}

// handleDownload serves a client binary for download
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")

	info, ok := platforms[platform]
	if !ok {
		s.respondError(w, http.StatusNotFound, "platform not found")
		return
	}

	filePath := filepath.Join(s.downloadsPath, info.Filename)

	// Check if file exists
	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.respondError(w, http.StatusNotFound, "client binary not found")
			return
		}
		s.log.Error().Err(err).Msg("Failed to stat file")
		s.respondError(w, http.StatusInternalServerError, "failed to get file")
		return
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to open file")
		s.respondError(w, http.StatusInternalServerError, "failed to get file")
		return
	}
	defer file.Close()

	// Set headers for download
	filename := info.Filename
	if strings.HasSuffix(filename, ".exe") {
		w.Header().Set("Content-Type", "application/vnd.microsoft.portable-executable")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Length", string(rune(stat.Size())))

	// Serve file
	http.ServeFile(w, r, filePath)
}

// handleInstallScript serves a shell install script with the domain derived from the request Host
func (s *Server) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	domain := r.Host
	if i := strings.IndexByte(domain, ':'); i != -1 {
		domain = domain[:i]
	}
	if domain == "" || domain == "localhost" || domain == "127.0.0.1" {
		domain = s.baseDomain
	}
	if domain == "" {
		domain = "mfdev.ru"
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	data := struct{ BaseURL string }{
		BaseURL: fmt.Sprintf("https://%s/api/downloads", domain),
	}
	if err := installTmpl.Execute(w, data); err != nil {
		s.log.Error().Err(err).Msg("failed to execute install script template")
	}
}
