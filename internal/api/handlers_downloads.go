package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtunnel/internal/api/dto"
)

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

// handleInstallScript serves a shell install script with the configured base domain
func (s *Server) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	domain := s.baseDomain
	if domain == "" {
		domain = "mfdev.ru"
	}

	script := fmt.Sprintf(`#!/bin/sh
set -e

BINARY_NAME="fxtunnel"
INSTALL_DIR="/usr/local/bin"
BASE_URL="https://%s/api/downloads"

main() {
    detect_os
    detect_arch
    check_dependencies

    echo "Downloading fxTunnel for ${OS}/${ARCH}..."

    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    DOWNLOAD_URL="${BASE_URL}/cli-${OS}-${ARCH}"
    TARGET="${TMP_DIR}/${BINARY_NAME}"

    download "$DOWNLOAD_URL" "$TARGET"

    chmod +x "$TARGET"

    echo "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TARGET" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "$TARGET" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    echo "fxTunnel installed successfully!"
    "${INSTALL_DIR}/${BINARY_NAME}" --version || true
}

detect_os() {
    case "$(uname -s)" in
        Linux*)  OS="linux" ;;
        Darwin*) OS="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
        *)
            echo "Error: unsupported operating system '$(uname -s)'" >&2
            exit 1
            ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)
            echo "Error: unsupported architecture '$(uname -m)'" >&2
            exit 1
            ;;
    esac

    # Windows only supports amd64
    if [ "$OS" = "windows" ] && [ "$ARCH" != "amd64" ]; then
        echo "Error: Windows builds are only available for amd64" >&2
        exit 1
    fi
}

check_dependencies() {
    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl"
    elif command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget"
    else
        echo "Error: curl or wget is required" >&2
        exit 1
    fi
}

download() {
    url="$1"
    output="$2"

    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fSL --progress-bar -o "$output" "$url"
    else
        wget -q --show-progress -O "$output" "$url"
    fi

    if [ ! -f "$output" ] || [ ! -s "$output" ]; then
        echo "Error: download failed" >&2
        exit 1
    fi
}

main
`, domain)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(script))
}
