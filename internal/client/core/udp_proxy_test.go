package core

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/config"
)

// TestUDPProxyFraming verifies that the UDP proxy correctly deframes stream data
// into UDP datagrams and frames UDP responses back onto the stream.
func TestUDPProxyFraming(t *testing.T) {
	// Start a local UDP echo server
	echoConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoConn.Close()

	go func() {
		buf := make([]byte, maxUDPPacketSize)
		for {
			n, addr, err := echoConn.ReadFrom(buf)
			if err != nil {
				return
			}
			_, _ = echoConn.WriteTo(buf[:n], addr)
		}
	}()

	echoAddr := echoConn.LocalAddr().(*net.UDPAddr)

	// Create a pipe to simulate the yamux stream
	streamClient, streamServer := net.Pipe()
	defer streamClient.Close()
	defer streamServer.Close()

	// Build a minimal Client and ActiveTunnel
	c := &Client{}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	defer c.cancel()
	c.log = zerolog.New(os.Stderr).Level(zerolog.DebugLevel)

	tunnel := &ActiveTunnel{
		Config: config.TunnelConfig{
			Type:      "udp",
			LocalAddr: echoAddr.IP.String(),
			LocalPort: echoAddr.Port,
			Name:      "test-udp",
		},
	}

	// Run the proxy in background
	// Wait for the proxy to stop, so its peer readers do not outlive the test.
	done := make(chan struct{})
	defer func() { streamClient.Close(); <-done }()
	go func() { c.handleUDPStream(streamServer, tunnel); close(done) }()

	// Write a framed UDP packet to the stream
	payload := []byte("hello udp")
	frame := make([]byte, udpHeaderSize+len(payload))
	binary.BigEndian.PutUint16(frame[0:2], uint16(len(payload))) //nolint:gosec // G115: test uses fixed short payload
	binary.BigEndian.PutUint32(frame[2:6], 0xDEADBEEF)
	copy(frame[udpHeaderSize:], payload)

	if _, err := streamClient.Write(frame); err != nil {
		t.Fatal(err)
	}

	// Read the echoed response frame from the stream
	_ = streamClient.SetReadDeadline(time.Now().Add(5 * time.Second))
	respHeader := make([]byte, udpHeaderSize)
	if _, err := io.ReadFull(streamClient, respHeader); err != nil {
		t.Fatal("failed to read response header:", err)
	}

	respLen := binary.BigEndian.Uint16(respHeader[0:2])
	respHash := binary.BigEndian.Uint32(respHeader[2:6])

	if int(respLen) != len(payload) {
		t.Fatalf("expected response length %d, got %d", len(payload), respLen)
	}
	if respHash != 0xDEADBEEF {
		t.Fatalf("expected addr hash 0xDEADBEEF, got 0x%X", respHash)
	}

	respPayload := make([]byte, respLen)
	if _, err := io.ReadFull(streamClient, respPayload); err != nil {
		t.Fatal("failed to read response payload:", err)
	}

	if string(respPayload) != "hello udp" {
		t.Fatalf("expected 'hello udp', got %q", respPayload)
	}
}

// TestUDPProxyPeerSockets checks that each addrHash gets its own local socket,
// that idle and least recently used sockets are evicted, and that closing the
// stream ends the proxy with all its peer readers.
func TestUDPProxyPeerSockets(t *testing.T) {
	oldIdle, oldMax := udpPeerIdle, udpMaxPeers
	t.Cleanup(func() { udpPeerIdle, udpMaxPeers = oldIdle, oldMax })
	udpPeerIdle, udpMaxPeers = 300*time.Millisecond, 2

	// Echo that answers with the source address the proxy used.
	echo, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer echo.Close()
	go func() {
		buf := make([]byte, 2048)
		for {
			_, from, err := echo.ReadFromUDP(buf)
			if err != nil {
				return
			}
			_, _ = echo.WriteToUDP([]byte(from.String()), from)
		}
	}()

	streamClient, streamServer := net.Pipe()
	c := &Client{log: zerolog.Nop()}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	defer c.cancel()
	echoAddr := echo.LocalAddr().(*net.UDPAddr)
	tunnel := &ActiveTunnel{Config: config.TunnelConfig{Type: "udp", LocalAddr: "127.0.0.1", LocalPort: echoAddr.Port}}
	done := make(chan struct{})
	go func() { c.handleUDPStream(streamServer, tunnel); close(done) }()

	// send returns the local source the proxy used for this hash.
	send := func(hash uint32) string {
		t.Helper()
		frame := make([]byte, udpHeaderSize+1)
		binary.BigEndian.PutUint16(frame[0:2], 1)
		binary.BigEndian.PutUint32(frame[2:6], hash)
		if _, err := streamClient.Write(frame); err != nil {
			t.Fatal(err)
		}
		_ = streamClient.SetReadDeadline(time.Now().Add(2 * time.Second))
		hdr := make([]byte, udpHeaderSize)
		if _, err := io.ReadFull(streamClient, hdr); err != nil {
			t.Fatal(err)
		}
		if got := binary.BigEndian.Uint32(hdr[2:6]); got != hash {
			t.Fatalf("reply tagged %d, want %d", got, hash)
		}
		body := make([]byte, binary.BigEndian.Uint16(hdr[0:2]))
		if _, err := io.ReadFull(streamClient, body); err != nil {
			t.Fatal(err)
		}
		return string(body)
	}

	a, b := send(1), send(2)
	if a == b {
		t.Fatalf("peers 1 and 2 share local socket %s", a)
	}
	if send(1) != a {
		t.Fatal("peer 1 got a new socket while active")
	}
	// Peer 2 is least recently used, so peer 3 pushes it out.
	send(3)
	if send(2) == b {
		t.Fatal("peer 2 kept its socket past the cap")
	}
	time.Sleep(2 * udpPeerIdle)
	if send(1) == a {
		t.Fatal("peer 1 kept its socket past the idle timeout")
	}

	streamClient.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("proxy did not stop after the stream closed")
	}
	if tunnel.BytesSent.Load() == 0 || tunnel.BytesReceived.Load() == 0 {
		t.Fatalf("traffic counters not updated: sent=%d received=%d", tunnel.BytesSent.Load(), tunnel.BytesReceived.Load())
	}
}
