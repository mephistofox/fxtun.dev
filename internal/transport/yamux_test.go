package transport

import (
	"context"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestYamuxTransport(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	serverSession, err := NewYamuxSession(serverConn, true)
	require.NoError(t, err)
	defer serverSession.Close()

	clientSession, err := NewYamuxSession(clientConn, false)
	require.NoError(t, err)
	defer clientSession.Close()

	ctx := context.Background()

	go func() {
		stream, err := clientSession.OpenStream(ctx)
		require.NoError(t, err)
		_, err = stream.Write([]byte("hello"))
		require.NoError(t, err)
		stream.Close()
	}()

	stream, err := serverSession.AcceptStream(ctx)
	require.NoError(t, err)

	buf, err := io.ReadAll(stream)
	require.NoError(t, err)
	require.Equal(t, "hello", string(buf))
}

func TestYamuxListener(t *testing.T) {
	ln, err := NewYamuxListener(":0", nil)
	require.NoError(t, err)
	defer ln.Close()

	require.NotEmpty(t, ln.Addr())
}
