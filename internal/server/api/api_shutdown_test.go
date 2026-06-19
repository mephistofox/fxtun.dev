package api

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

// Shutdown must be idempotent: it is invoked both by the Start ctx.Done watcher
// and explicitly from main on signal, so a second call must not panic on a
// double close of shutdownCh (which the deploy verify caught as a crash).
func TestServer_ShutdownIdempotent(t *testing.T) {
	s := &Server{shutdownCh: make(chan struct{}), log: zerolog.Nop()}

	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("first shutdown: %v", err)
	}
	// Must not panic on "close of closed channel".
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
}
