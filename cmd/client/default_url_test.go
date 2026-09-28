package main

import (
	"testing"

	client "github.com/mephistofox/fxtun.dev/internal/client/core"
	"github.com/mephistofox/fxtun.dev/internal/config"
)

// The fallback web URL must be the host the default control endpoint maps to.
// When fxtun.dev became a 301 to fxtun.ru, the stale fallback broke browser
// login: Go replays a POST as GET across a 301 and the API answers 405.
func TestDefaultServerURLMatchesDefaultServer(t *testing.T) {
	want := client.WebBaseURL(config.DefaultServerAddress)
	if DefaultServerURL != want {
		t.Fatalf("DefaultServerURL = %q, want %q", DefaultServerURL, want)
	}
}
