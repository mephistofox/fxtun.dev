package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtun.dev/internal/server/api/dto"
)

//go:embed install.sh
var installScriptTmpl string

//go:embed install.ps1
var installPS1Tmpl string

var installTmpl = template.Must(template.New("install").Parse(installScriptTmpl))
var installPS1 = template.Must(template.New("installps1").Parse(installPS1Tmpl))

// Platform information for downloads
type platformInfo struct {
	Filename   string
	OS         string
	Arch       string
	ClientType string // "cli" or "gui"
}

var osNames = map[string]string{
	"linux": "Linux", "darwin": "macOS", "windows": "Windows",
}

// parseBinaryName extracts platform info from binary filename.
// Patterns: fxtunnel-{os}-{arch}[.exe], fxtunnel-gui-{os}-{arch}[.exe]
func parseBinaryName(filename string) (platform string, info platformInfo, ok bool) {
	name := strings.TrimSuffix(filename, ".exe")

	var clientType, remainder string
	if strings.HasPrefix(name, "fxtunnel-gui-") {
		clientType = "gui"
		remainder = strings.TrimPrefix(name, "fxtunnel-gui-")
	} else if strings.HasPrefix(name, "fxtunnel-") {
		clientType = "cli"
		remainder = strings.TrimPrefix(name, "fxtunnel-")
	} else {
		return "", platformInfo{}, false
	}

	parts := strings.SplitN(remainder, "-", 2)
	if len(parts) != 2 {
		return "", platformInfo{}, false
	}

	osKey, arch := parts[0], parts[1]
	osName, known := osNames[osKey]
	if !known {
		return "", platformInfo{}, false
	}

	platform = clientType + "-" + osKey + "-" + arch
	return platform, platformInfo{
		Filename:   filename,
		OS:         osName,
		Arch:       arch,
		ClientType: clientType,
	}, true
}

// scanDownloads reads the downloads directory and returns discovered platforms.
func (s *Server) scanDownloads() (map[string]platformInfo, []*dto.DownloadDTO, []*dto.DownloadDTO) {
	found := make(map[string]platformInfo)
	var cliClients, guiClients []*dto.DownloadDTO

	entries, err := os.ReadDir(s.downloadsPath)
	if err != nil {
		return found, nil, nil
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		platform, info, ok := parseBinaryName(e.Name())
		if !ok {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		found[platform] = info
		d := &dto.DownloadDTO{
			Platform:   platform,
			OS:         info.OS,
			Arch:       info.Arch,
			Size:       fi.Size(),
			URL:        "/api/downloads/" + platform,
			ClientType: info.ClientType,
		}
		if info.ClientType == "gui" {
			guiClients = append(guiClients, d)
		} else {
			cliClients = append(cliClients, d)
		}
	}

	return found, cliClients, guiClients
}

// handleListDownloads returns a list of available client downloads
func (s *Server) handleListDownloads(w http.ResponseWriter, r *http.Request) {
	_, cliClients, guiClients := s.scanDownloads()

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

	found, _, _ := s.scanDownloads()
	info, ok := found[platform]
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
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))

	// Serve file
	http.ServeFile(w, r, filePath)
}

// handleInstallScript serves a shell install script with the domain derived from the request Host
func (s *Server) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	// Use request host to match the domain the user is accessing
	domain := requestHost(r)
	if domain == "" {
		domain = s.baseDomain
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	data := struct {
		BaseURL    string
		WebsiteURL string
	}{
		BaseURL:    fmt.Sprintf("https://%s/api/downloads", domain),
		WebsiteURL: fmt.Sprintf("https://%s", domain),
	}
	if err := installTmpl.Execute(w, data); err != nil {
		s.log.Error().Err(err).Msg("failed to execute install script template")
	}
}

// handleInstallPS1 serves a PowerShell install script for Windows
func (s *Server) handleInstallPS1(w http.ResponseWriter, r *http.Request) {
	domain := requestHost(r)
	if domain == "" {
		domain = s.baseDomain
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	data := struct {
		BaseURL    string
		WebsiteURL string
	}{
		BaseURL:    fmt.Sprintf("https://%s/api/downloads", domain),
		WebsiteURL: fmt.Sprintf("https://%s", domain),
	}
	if err := installPS1.Execute(w, data); err != nil {
		s.log.Error().Err(err).Msg("failed to execute PowerShell install script template")
	}
}
