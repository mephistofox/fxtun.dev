package transport

import (
	"context"
	"io"
)

// Stream represents a single bidirectional stream within a multiplexed session.
type Stream interface {
	io.ReadWriteCloser
}

// Session represents a multiplexed connection with multiple streams.
type Session interface {
	OpenStream(ctx context.Context) (Stream, error)
	AcceptStream(ctx context.Context) (Stream, error)
	Close() error
	CloseWithError(code uint64, msg string) error
	IsClosed() bool
}

// Listener accepts incoming multiplexed sessions.
type Listener interface {
	Accept(ctx context.Context) (Session, error)
	Close() error
	Addr() string
}
