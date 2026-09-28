package core

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	udpHeaderSize    = 6     // 2 bytes length + 4 bytes addr hash (uint32)
	maxUDPPacketSize = 65507 // max UDP payload
)

// Each remote peer gets its own local socket, so the local service's replies
// can be told apart. These bound how many sockets a tunnel keeps open.
var (
	udpPeerIdle = 60 * time.Second
	udpMaxPeers = 1024
)

type udpPeer struct {
	conn     *net.UDPConn
	lastUsed atomic.Int64 // unix nanos of the last packet in either direction
}

// handleUDPStream proxies a yamux stream (with UDP framing) to a local UDP service.
// Frames carry the remote peer's addrHash; replies read from that peer's local
// socket go back with the same hash, so the server can route them to it.
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

	c.log.Debug().
		Str("tunnel", tunnel.Config.Name).
		Str("local", addr.String()).
		Msg("UDP proxy started")

	// Unblock the frame read below on shutdown.
	stopWatch := context.AfterFunc(c.ctx, func() { _ = stream.SetReadDeadline(time.Now()) })
	defer stopWatch()

	var (
		mu      sync.Mutex // guards peers
		peers   = make(map[uint32]*udpPeer)
		writeMu sync.Mutex // peer readers share the one stream
		wg      sync.WaitGroup
	)
	defer func() {
		// Closing the stream fails any reader blocked writing to it.
		_ = stream.Close()
		mu.Lock()
		for _, p := range peers {
			_ = p.conn.Close()
		}
		mu.Unlock()
		wg.Wait()
	}()

	// Local replies of one peer → stream, tagged with that peer's hash.
	readReplies := func(hash uint32, p *udpPeer) {
		defer wg.Done()
		defer func() {
			mu.Lock()
			if peers[hash] == p {
				delete(peers, hash)
			}
			mu.Unlock()
			_ = p.conn.Close()
		}()
		buf := make([]byte, udpHeaderSize+maxUDPPacketSize)
		for {
			last := time.Unix(0, p.lastUsed.Load())
			_ = p.conn.SetReadDeadline(last.Add(udpPeerIdle))
			n, err := p.conn.Read(buf[udpHeaderSize:])
			if err != nil {
				// The deadline is from an older lastUsed if the peer sent
				// something since: then it is not idle yet.
				if ne, ok := err.(net.Error); ok && ne.Timeout() &&
					time.Since(time.Unix(0, p.lastUsed.Load())) < udpPeerIdle {
					continue
				}
				return
			}
			p.lastUsed.Store(time.Now().UnixNano())

			binary.BigEndian.PutUint16(buf[0:2], uint16(n)) //nolint:gosec // n bounded by UDP read
			binary.BigEndian.PutUint32(buf[2:6], hash)
			writeMu.Lock()
			_, err = stream.Write(buf[:udpHeaderSize+n])
			writeMu.Unlock()
			if err != nil {
				c.log.Debug().Err(err).Msg("UDP stream write error")
				return
			}
			tunnel.BytesSent.Add(int64(n))
		}
	}

	// peerFor returns the peer's local socket, opening one on first contact.
	peerFor := func(hash uint32) (*udpPeer, error) {
		mu.Lock()
		defer mu.Unlock()
		if p := peers[hash]; p != nil {
			return p, nil
		}
		if len(peers) >= udpMaxPeers {
			// ponytail: O(n) scan only when full; a list-backed LRU if that shows up.
			var oldest *udpPeer
			var oldestHash uint32
			for h, p := range peers {
				if oldest == nil || p.lastUsed.Load() < oldest.lastUsed.Load() {
					oldest, oldestHash = p, h
				}
			}
			delete(peers, oldestHash)
			_ = oldest.conn.Close() // ends its reader
		}
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			return nil, err
		}
		p := &udpPeer{conn: conn}
		p.lastUsed.Store(time.Now().UnixNano())
		peers[hash] = p
		wg.Add(1)
		go readReplies(hash, p)
		return p, nil
	}

	// Stream → local UDP: read framed packets from yamux, send each from its peer's socket.
	header := make([]byte, udpHeaderSize)
	payload := make([]byte, maxUDPPacketSize)
	for {
		if _, err := io.ReadFull(stream, header); err != nil {
			c.log.Debug().Err(err).Msg("UDP stream read header error")
			return
		}

		length := binary.BigEndian.Uint16(header[0:2])
		addrHash := binary.BigEndian.Uint32(header[2:6])

		// A hostile server can declare more than the payload buffer holds,
		// which would panic this goroutine.
		if !udpFrameLenValid(length) {
			c.log.Warn().Uint16("length", length).Msg("UDP frame length exceeds buffer, closing stream")
			return
		}

		if _, err := io.ReadFull(stream, payload[:length]); err != nil {
			c.log.Debug().Err(err).Msg("UDP stream read payload error")
			return
		}

		p, err := peerFor(addrHash)
		if err != nil {
			c.log.Error().Err(err).Int("port", tunnel.Config.LocalPort).Msg("Failed to dial local UDP service")
			continue
		}
		p.lastUsed.Store(time.Now().UnixNano())
		// UDP drops are normal; a failed send loses this datagram, not the tunnel.
		if _, err := p.conn.Write(payload[:length]); err != nil {
			c.log.Debug().Err(err).Msg("UDP local write error")
			continue
		}
		tunnel.BytesReceived.Add(int64(length))
	}
}

// udpFrameLenValid reports whether a peer-declared frame length fits the
// buffer it will be read into.
func udpFrameLenValid(length uint16) bool {
	return int(length) <= maxUDPPacketSize
}
