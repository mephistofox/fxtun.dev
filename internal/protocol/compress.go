package protocol

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/klauspost/compress/zstd"
)

const (
	compressNone byte = 0x00
	compressZstd byte = 0x01
)

// NegotiateCompression performs a 1-byte handshake and wraps conn in zstd if both sides agree.
// For client: sends preference, reads server response.
// For server: reads client preference, sends response.
// Returns the (possibly wrapped) ReadWriteCloser, whether compression is active, and any error.
func NegotiateCompression(conn net.Conn, wantCompress bool, isServer bool) (io.ReadWriteCloser, bool, error) {
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	var pref byte
	if wantCompress {
		pref = compressZstd
	}

	if isServer {
		// Server: read client preference
		buf := []byte{0}
		if _, err := io.ReadFull(conn, buf); err != nil {
			return nil, false, fmt.Errorf("read compression preference: %w", err)
		}
		clientWants := buf[0] == compressZstd

		accepted := clientWants && wantCompress
		var resp byte
		if accepted {
			resp = compressZstd
		}
		if _, err := conn.Write([]byte{resp}); err != nil {
			return nil, false, fmt.Errorf("write compression response: %w", err)
		}

		if accepted {
			return wrapZstd(conn)
		}
		return conn, false, nil
	}

	// Client: send preference, read response
	if _, err := conn.Write([]byte{pref}); err != nil {
		return nil, false, fmt.Errorf("write compression preference: %w", err)
	}

	buf := []byte{0}
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, false, fmt.Errorf("read compression response: %w", err)
	}

	if buf[0] == compressZstd && wantCompress {
		return wrapZstd(conn)
	}
	return conn, false, nil
}

func wrapZstd(conn net.Conn) (io.ReadWriteCloser, bool, error) {
	encoder, err := zstd.NewWriter(conn, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return nil, false, fmt.Errorf("create zstd encoder: %w", err)
	}
	decoder, err := zstd.NewReader(conn)
	if err != nil {
		encoder.Close()
		return nil, false, fmt.Errorf("create zstd decoder: %w", err)
	}
	return &compressedConn{
		Conn:    conn,
		encoder: encoder,
		decoder: decoder,
	}, true, nil
}

// compressedConn wraps a net.Conn with zstd compression.
// It delegates all net.Conn methods except Read/Write/Close.
type compressedConn struct {
	net.Conn
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

func (c *compressedConn) Read(p []byte) (int, error) {
	return c.decoder.Read(p)
}

func (c *compressedConn) Write(p []byte) (int, error) {
	n, err := c.encoder.Write(p)
	if err != nil {
		return n, err
	}
	// Flush to ensure data is sent immediately (important for interactive protocols)
	if err := c.encoder.Flush(); err != nil {
		return n, err
	}
	return n, nil
}

func (c *compressedConn) Close() error {
	c.encoder.Close()
	c.decoder.Close()
	return c.Conn.Close()
}
