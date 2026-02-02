package daemon

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "daemon.json")
	want := &State{
		PID:       1234,
		APIAddr:   "127.0.0.1:9090",
		Server:    "example.com:4443",
		StartedAt: time.Now().Truncate(time.Second),
	}
	if err := SaveState(path, want); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got.PID != want.PID || got.APIAddr != want.APIAddr || got.Server != want.Server || !got.StartedAt.Equal(want.StartedAt) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := LoadState(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRemoveState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	if err := SaveState(path, &State{PID: 1}); err != nil {
		t.Fatal(err)
	}
	RemoveState(path)
	if _, err := LoadState(path); err == nil {
		t.Fatal("file should have been removed")
	}
}

func TestDefaultStatePath(t *testing.T) {
	p := DefaultStatePath()
	if p == "" {
		t.Fatal("DefaultStatePath returned empty string")
	}
}
