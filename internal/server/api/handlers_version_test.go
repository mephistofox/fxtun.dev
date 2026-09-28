package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// TestHandleVersion_AdvertisesGUIBuilds: the version endpoint must expose the
// desktop builds under their own keys. A GUI client that only finds a "cli-*"
// path installs the command-line binary over itself.
func TestHandleVersion_AdvertisesGUIBuilds(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"fxtunnel-linux-amd64", "fxtunnel-gui-linux-amd64"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("BINARY"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	s := &Server{downloadsPath: dir, log: zerolog.Nop()}
	rec := httptest.NewRecorder()
	s.handleVersion(rec, httptest.NewRequest(http.MethodGet, "/api/version", nil))

	var resp VersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	got, ok := resp.Downloads["gui_linux_amd64"]
	if !ok {
		t.Fatalf("no gui_linux_amd64 entry in %v", resp.Downloads)
	}
	if strings.Contains(got, "cli-") {
		t.Fatalf("gui entry points at a CLI artifact: %q", got)
	}
	if got != "/api/downloads/gui-linux-amd64" {
		t.Fatalf("gui download path = %q", got)
	}
}

// TestHandleVersion_NoGUIBuildPublished: nothing is advertised that is not on
// disk, so a GUI client is told there is no build rather than handed a CLI path.
func TestHandleVersion_NoGUIBuildPublished(t *testing.T) {
	s := &Server{downloadsPath: t.TempDir(), log: zerolog.Nop()}
	rec := httptest.NewRecorder()
	s.handleVersion(rec, httptest.NewRequest(http.MethodGet, "/api/version", nil))

	var resp VersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for k := range resp.Downloads {
		if strings.HasPrefix(k, "gui_") {
			t.Fatalf("unexpected gui entry %q with an empty downloads directory", k)
		}
	}
}

// client_version must describe the binaries actually being served, not the
// server's own build. They diverge on every client release: sync-clients uploads
// the new binaries, but the running server keeps reporting the tag it was built
// from, so `fxtunnel update` answers "already up to date" and nobody upgrades.
func TestHandleVersion_ClientVersionComesFromDownloads(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fxtunnel-linux-amd64"), []byte("BINARY"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("v3.16.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := &Server{downloadsPath: dir, version: "v3.15.0", log: zerolog.Nop()}
	rec := httptest.NewRecorder()
	s.handleVersion(rec, httptest.NewRequest(http.MethodGet, "/api/version", nil))

	var resp VersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ClientVersion != "v3.16.0" {
		t.Fatalf("client_version = %q, want the served binaries' v3.16.0", resp.ClientVersion)
	}
	if resp.ServerVersion != "v3.15.0" {
		t.Fatalf("server_version = %q, want the server's own v3.15.0", resp.ServerVersion)
	}
}

// Without the marker file — an older downloads directory, or a hand-populated
// one — the server's own version stays the answer rather than an empty string.
func TestHandleVersion_ClientVersionFallsBackToServerBuild(t *testing.T) {
	s := &Server{downloadsPath: t.TempDir(), version: "v3.15.0", log: zerolog.Nop()}
	rec := httptest.NewRecorder()
	s.handleVersion(rec, httptest.NewRequest(http.MethodGet, "/api/version", nil))

	var resp VersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ClientVersion != "v3.15.0" {
		t.Fatalf("client_version = %q, want fallback to v3.15.0", resp.ClientVersion)
	}
}

// The marker must not leak into the download list as a fake platform.
func TestHandleVersion_VersionFileIsNotADownload(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("v3.16.0"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Server{downloadsPath: dir, log: zerolog.Nop()}
	_, cli, gui := s.scanDownloads()
	if len(cli) != 0 || len(gui) != 0 {
		t.Fatalf("VERSION parsed as a binary: cli=%v gui=%v", cli, gui)
	}
}
