package transport

import (
	"context"
	"io"
	"testing"
)

func TestStreamInterfaceCompiles(t *testing.T) {
	var _ Stream = (*mockStream)(nil)
	var _ Session = (*mockSession)(nil)
}

type mockStream struct{}

func (m *mockStream) Read(p []byte) (int, error)  { return 0, io.EOF }
func (m *mockStream) Write(p []byte) (int, error) { return len(p), nil }
func (m *mockStream) Close() error                { return nil }

type mockSession struct{}

func (m *mockSession) OpenStream(ctx context.Context) (Stream, error)   { return &mockStream{}, nil }
func (m *mockSession) AcceptStream(ctx context.Context) (Stream, error) { return &mockStream{}, nil }
func (m *mockSession) Close() error                                     { return nil }
func (m *mockSession) CloseWithError(code uint64, msg string) error     { return nil }
func (m *mockSession) IsClosed() bool                                   { return false }
