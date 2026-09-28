package main

import (
	"errors"
	"testing"
	"time"
)

// SUSPECT: newAPIClient's error on a missing token was worded differently
// from the one runClient gives for the same situation. Both must point at
// `fxtunnel login` through the same errNotLoggedIn.
func TestNewAPIClient_NotLoggedIn(t *testing.T) {
	isolate(t)

	if _, err := newAPIClient(); !errors.Is(err, errNotLoggedIn) {
		t.Fatalf("got %v, want errNotLoggedIn", err)
	}
}

// SUSPECT: apiClient had no HTTP timeout — a stuck server hung `domains`
// commands forever.
func TestNewAPIClient_HasTimeout(t *testing.T) {
	isolate(t)
	token = "tok"

	c, err := newAPIClient()
	if err != nil {
		t.Fatal(err)
	}
	if c.http.Timeout != 30*time.Second {
		t.Fatalf("got timeout %v, want 30s", c.http.Timeout)
	}
}
