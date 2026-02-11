package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"
)

const (
	udpHeaderSize    = 10    // 2 bytes length + 8 bytes addr hash (fnv64a)
	maxUDPPacketSize = 65507 // max UDP payload
)

// handleUDPStream proxies a yamux stream (with UDP framing) to a local UDP service.
func (c *Client) handleUDPStream(stream net.Conn, tunnel *ActiveTunnel) {
	localAddr := tunnel.Config.LocalAddr
	if localAddr == "" {
		localAddr = "127.0.0.1"
	}
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", localAddr, tunnel.Config.LocalPort))
	if err != nil {
		c.log.Error().Err(err).Msg("Failed to resolve local UDP address")
		return
	}

	udpConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		c.log.Error().Err(err).Int("port", tunnel.Config.LocalPort).Msg("Failed to dial local UDP service")
		return
	}
	defer udpConn.Close()

	c.log.Debug().
		Str("tunnel", tunnel.Config.Name).
		Str("local", udpConn.RemoteAddr().String()).
		Msg("UDP proxy started")

	// Store last seen addr hash for responses
	var lastAddrHash atomic.Uint64

	done := make(chan struct{}, 2)

	// Stream → local UDP: read framed packets from yamux, send as UDP datagrams
	go func() {
		defer func() { done <- struct{}{} }()
		header := make([]byte, udpHeaderSize)
		payload := make([]byte, maxUDPPacketSize)
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
			}

			if _, err := io.ReadFull(stream, header); err != nil {
				c.log.Debug().Err(err).Msg("UDP stream read header error")
				return
			}

			length := binary.BigEndian.Uint16(header[0:2])
			addrHash := binary.BigEndian.Uint64(header[2:10])
			lastAddrHash.Store(addrHash)

			if _, err := io.ReadFull(stream, payload[:length]); err != nil {
				c.log.Debug().Err(err).Msg("UDP stream read payload error")
				return
			}

			if _, err := udpConn.Write(payload[:length]); err != nil {
				c.log.Debug().Err(err).Msg("UDP local write error")
				return
			}
			tunnel.BytesReceived.Add(int64(length))
		}
	}()

	// Local UDP → stream: read UDP datagrams, write framed packets to yamux
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, maxUDPPacketSize)
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
			}

			_ = udpConn.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, err := udpConn.Read(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				c.log.Debug().Err(err).Msg("UDP local read error")
				return
			}

			frame := make([]byte, udpHeaderSize+n)
			binary.BigEndian.PutUint16(frame[0:2], uint16(n)) //nolint:gosec // n bounded by UDP read
			binary.BigEndian.PutUint64(frame[2:10], lastAddrHash.Load())
			copy(frame[udpHeaderSize:], buf[:n])

			if _, err := stream.Write(frame); err != nil {
				c.log.Debug().Err(err).Msg("UDP stream write error")
				return
			}
			tunnel.BytesSent.Add(int64(n))
		}
	}()

	// Wait for one goroutine to finish or context cancellation
	select {
	case <-done:
	case <-c.ctx.Done():
	}
}
