package client

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
	go c.handleUDPStream(streamServer, tunnel)

	// Write a framed UDP packet to the stream
	payload := []byte("hello udp")
	frame := make([]byte, udpHeaderSize+len(payload))
	binary.BigEndian.PutUint16(frame[0:2], uint16(len(payload))) //nolint:gosec // G115: test uses fixed short payload
	binary.BigEndian.PutUint64(frame[2:10], 0xDEADBEEF)
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
	respHash := binary.BigEndian.Uint64(respHeader[2:10])

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
