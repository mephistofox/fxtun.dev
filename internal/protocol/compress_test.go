package protocol

import (
	"io"
	"net"
	"runtime"
	"testing"
	"time"
)

// TestZstdRoundTrip checks that the bounded decoder still talks to our own
// encoder: both sides negotiate zstd and exchange a payload larger than a
// single block.
func TestZstdRoundTrip(t *testing.T) {
	srvConn, cliConn := net.Pipe()
	defer srvConn.Close()
	defer cliConn.Close()

	payload := make([]byte, 512*1024)
	for i := range payload {
		payload[i] = byte(i % 251)
	}

	type res struct {
		rwc io.ReadWriteCloser
		err error
	}
	srvCh := make(chan res, 1)
	go func() {
		rwc, _, err := NegotiateCompression(srvConn, true, true)
		srvCh <- res{rwc, err}
	}()

	client, compressed, err := NegotiateCompression(cliConn, true, false)
	if err != nil || !compressed {
		t.Fatalf("client negotiate: compressed=%v err=%v", compressed, err)
	}
	srv := <-srvCh
	if srv.err != nil {
		t.Fatalf("server negotiate: %v", srv.err)
	}

	go func() { _, _ = client.Write(payload) }()

	got := make([]byte, len(payload))
	if _, err := io.ReadFull(srv.rwc, got); err != nil {
		t.Fatalf("read back: %v", err)
	}
	for i := range got {
		if got[i] != payload[i] {
			t.Fatalf("payload differs at byte %d", i)
		}
	}
}

// TestZstdWindowIsBounded pins the pre-auth memory cost of compression.
//
// The peer chooses the zstd window in its frame header and the decoder
// allocates about twice that on the first read. With the library default of
// 512 MB, the 10-byte frame header below made an unauthenticated control
// connection cost the server ~1 GB.
func TestZstdWindowIsBounded(t *testing.T) {
	// magic, descriptor 0x00 (no content size, not single-segment),
	// window descriptor 0x98 => exponent 19 => 512 MiB window,
	// then one raw 1-byte block marked as last.
	frame := []byte{0x28, 0xb5, 0x2f, 0xfd, 0x00, 0x98, 0x09, 0x00, 0x00, 'x'}

	local, remote := net.Pipe()
	defer local.Close()
	go func() {
		defer remote.Close()
		_ = remote.SetDeadline(time.Now().Add(5 * time.Second))
		_, _ = remote.Write([]byte{compressZstd})
		ack := make([]byte, 1)
		_, _ = io.ReadFull(remote, ack)
		_, _ = remote.Write(frame)
		_, _ = io.Copy(io.Discard, remote)
	}()

	conn, compressed, err := NegotiateCompression(local, true, true)
	if err != nil || !compressed {
		t.Fatalf("negotiate: compressed=%v err=%v", compressed, err)
	}

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	_ = local.SetDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 16)
	_, _ = conn.Read(buf)

	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc
	if allocated > 64<<20 {
		t.Fatalf("a 10-byte frame header made the decoder allocate %d MiB", allocated>>20)
	}
}
