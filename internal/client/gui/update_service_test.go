package gui

import (
	"runtime"
	"testing"

	client "github.com/mephistofox/fxtun.dev/internal/client/core"
)

// TestGUIDownloadURL_NeverResolvesCLIArtifact: the server's download map is
// mostly CLI paths. Picking one of those would make the GUI overwrite itself
// with the command-line binary, so only a "gui_<os>_<arch>" entry counts.
func TestGUIDownloadURL_NeverResolvesCLIArtifact(t *testing.T) {
	platform := runtime.GOOS + "_" + runtime.GOARCH

	cliOnly := &client.UpdateInfo{
		ServerHost: "fxtun.dev",
		Downloads:  map[string]string{platform: "/api/downloads/cli-" + runtime.GOOS + "-" + runtime.GOARCH},
	}
	if got := guiDownloadURL(cliOnly); got != "" {
		t.Fatalf("resolved a CLI artifact for the GUI: %q", got)
	}

	withGUI := &client.UpdateInfo{
		ServerHost: "fxtun.dev",
		Downloads: map[string]string{
			platform:          "/api/downloads/cli-" + runtime.GOOS + "-" + runtime.GOARCH,
			"gui_" + platform: "/api/downloads/gui-" + runtime.GOOS + "-" + runtime.GOARCH,
		},
	}
	want := "https://fxtun.dev/api/downloads/gui-" + runtime.GOOS + "-" + runtime.GOARCH
	if got := guiDownloadURL(withGUI); got != want {
		t.Fatalf("guiDownloadURL = %q, want %q", got, want)
	}
}
