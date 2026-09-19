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
