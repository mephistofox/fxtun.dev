package protocol

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

const (
	compressNone byte = 0x00
	compressZstd byte = 0x01

	// maxZstdWindow is the largest zstd window we accept from a peer. It
	// matches the window zstd.SpeedDefault uses, the level of older peers.
	maxZstdWindow = 8 << 20
)

// ZstdLevel is the encoder level for transport compression. A variable so the
// benchmark can compare levels; both ends pick it independently.
// SpeedFastest: on the link-emulation bench it matches SpeedDefault on JSON
// (3.9 vs 3.9 MB/s at 10M/100ms, 3.25x over off; 37.2 vs 37.3 at 100M/30ms)
// and loses nothing on random data. With throughput tied, fastest wins as the
// cheaper level by construction; CPU was not measured here.
var ZstdLevel = zstd.SpeedFastest

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
	encoder, err := zstd.NewWriter(conn, zstd.WithEncoderLevel(ZstdLevel))
	if err != nil {
		return nil, false, fmt.Errorf("create zstd encoder: %w", err)
	}
	// Bound the decoder's window. The peer picks the window size in its frame
	// header, and the decoder allocates twice that up front — a 10-byte frame
	// header declaring the library default of 512 MB makes us allocate ~1 GB
	// before a single byte is authenticated. Our own encoder never exceeds
	// 8 MB (zstd SpeedDefault), so anything larger is not a peer we can talk to.
	decoder, err := zstd.NewReader(conn, zstd.WithDecoderMaxWindow(maxZstdWindow))
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
	// zstd's Decoder and Encoder must not be closed while in use, so Close
	// waits on these for an in-flight Read or Write to return.
	readMu, writeMu sync.Mutex
}

func (c *compressedConn) Read(p []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	return c.decoder.Read(p)
}

func (c *compressedConn) Write(p []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
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
	// Close the connection first: it unblocks a goroutine reading or writing
	// through the codec, and the locks then wait for it to leave before the
	// codec is torn down underneath it.
	err := c.Conn.Close()
	c.writeMu.Lock()
	c.encoder.Close()
	c.writeMu.Unlock()
	c.readMu.Lock()
	c.decoder.Close()
	c.readMu.Unlock()
	return err
}
