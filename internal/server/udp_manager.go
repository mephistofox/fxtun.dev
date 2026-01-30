package server

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/protocol"
)

const (
	maxUDPPacketSize = 65507
	udpHeaderSize    = 6 // 2 bytes length + 4 bytes addr hash
)

var udpFramePool = sync.Pool{
	New: func() any {
		buf := make([]byte, udpHeaderSize+maxUDPPacketSize)
		return buf
	},
}

// UDPManager manages UDP tunnel ports
type UDPManager struct {
	server    *Server
	log       zerolog.Logger
	usedPorts map[int]bool
	mu        sync.Mutex
}

// NewUDPManager creates a new UDP manager
func NewUDPManager(server *Server, log zerolog.Logger) *UDPManager {
	return &UDPManager{
		server:    server,
		log:       log.With().Str("component", "udp_manager").Logger(),
		usedPorts: make(map[int]bool),
	}
}

// AllocatePort allocates a port for a UDP tunnel
func (m *UDPManager) AllocatePort(requestedPort int) (int, *net.UDPConn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	portRange := m.server.cfg.Server.UDPPortRange

	if requestedPort != 0 {
		// Check if requested port is in range
		if requestedPort < portRange.Min || requestedPort > portRange.Max {
			return 0, nil, fmt.Errorf("port %d is outside allowed range (%d-%d)",
				requestedPort, portRange.Min, portRange.Max)
		}

		// Check if port is already used
		if m.usedPorts[requestedPort] {
			return 0, nil, fmt.Errorf("port %d is already in use", requestedPort)
		}

		// Try to bind
		addr := &net.UDPAddr{Port: requestedPort}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to bind port %d: %w", requestedPort, err)
		}

		m.usedPorts[requestedPort] = true
		return requestedPort, conn, nil
	}

	// Auto-assign port
	for port := portRange.Min; port <= portRange.Max; port++ {
		if m.usedPorts[port] {
			continue
		}

		addr := &net.UDPAddr{Port: port}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			continue
		}

		m.usedPorts[port] = true
		return port, conn, nil
	}

	return 0, nil, fmt.Errorf("no available ports in range %d-%d", portRange.Min, portRange.Max)
}

// ReleasePort releases a previously allocated port
func (m *UDPManager) ReleasePort(port int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.usedPorts, port)
}

// HandlePackets handles incoming UDP packets for a tunnel
func (m *UDPManager) HandlePackets(tunnel *Tunnel, client *Client) {
	defer func() {
		m.ReleasePort(tunnel.RemotePort)
		if tunnel.udpConn != nil {
			tunnel.udpConn.Close()
		}
	}()

	// Open a stream for this UDP tunnel
	stream, err := client.OpenStream()
	if err != nil {
		m.log.Error().Err(err).Msg("Failed to open stream for UDP tunnel")
		return
	}
	defer stream.Close()

	// Send tunnel info
	connID := generateID()
	newConn := &protocol.NewConnectionMessage{
		Message:      protocol.NewMessage(protocol.MsgNewConnection),
		TunnelID:     tunnel.ID,
		ConnectionID: connID,
		RemoteAddr:   "udp",
	}

	streamCodec := protocol.NewCodec(stream, stream)
	if err := streamCodec.Encode(newConn); err != nil {
		m.log.Error().Err(err).Msg("Failed to send UDP tunnel info")
		return
	}

	// Track client addresses for responses (keyed by addr.String() to avoid hash collisions)
	clientAddrs := make(map[string]*net.UDPAddr)
	hashToKey := make(map[uint32]string)
	clientLastSeen := make(map[string]time.Time)
	var addrMu sync.RWMutex

	// Cleanup goroutine to evict stale entries
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-client.ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				addrMu.Lock()
				for key, lastSeen := range clientLastSeen {
					if now.Sub(lastSeen) > 60*time.Second {
						delete(clientAddrs, key)
						delete(clientLastSeen, key)
						// Note: don't clean hashToKey as it's small and hash collisions are rare
					}
				}
				addrMu.Unlock()
			}
		}
	}()

	// Read from UDP and send to stream
	go func() {
		buf := make([]byte, maxUDPPacketSize)
		for {
			select {
			case <-client.ctx.Done():
				return
			default:
			}

			tunnel.udpConn.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, addr, err := tunnel.udpConn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				m.log.Debug().Err(err).Msg("UDP read error")
				return
			}

			// Use string key to avoid hash collisions
			addrKey := addr.String()
			addrHash := hashAddr(addr)

			// Store address for responses
			addrMu.Lock()
			clientAddrs[addrKey] = addr
			hashToKey[addrHash] = addrKey
			clientLastSeen[addrKey] = time.Now()
			addrMu.Unlock()

			// Frame: [2 bytes length][4 bytes addr hash][payload]
			frame := udpFramePool.Get().([]byte)
			frameLen := udpHeaderSize + n
			binary.BigEndian.PutUint16(frame[0:2], uint16(n))
			binary.BigEndian.PutUint32(frame[2:6], addrHash)
			copy(frame[udpHeaderSize:], buf[:n])

			_, werr := stream.Write(frame[:frameLen])
			udpFramePool.Put(frame)
			if werr != nil {
				m.log.Debug().Err(werr).Msg("Failed to write to stream")
				return
			}
		}
	}()

	// Read from stream and send to UDP
	header := make([]byte, udpHeaderSize)
	for {
		select {
		case <-client.ctx.Done():
			return
		default:
		}

		// Read frame header
		if _, err := io.ReadFull(stream, header); err != nil {
			m.log.Debug().Err(err).Msg("Failed to read UDP frame header")
			return
		}

		length := binary.BigEndian.Uint16(header[0:2])
		addrHash := binary.BigEndian.Uint32(header[2:6])

		// Read payload into pooled buffer
		frame := udpFramePool.Get().([]byte)
		if _, err := io.ReadFull(stream, frame[:length]); err != nil {
			udpFramePool.Put(frame)
			m.log.Debug().Err(err).Msg("Failed to read UDP payload")
			return
		}

		// Find client address via hash-to-key reverse lookup
		addrMu.RLock()
		key := hashToKey[addrHash]
		addr := clientAddrs[key]
		addrMu.RUnlock()

		if addr != nil {
			tunnel.udpConn.WriteToUDP(frame[:length], addr)
		}
		udpFramePool.Put(frame)
	}
}

// hashAddr creates a hash of a UDP address for tracking
func hashAddr(addr *net.UDPAddr) uint32 {
	// Simple hash combining IP and port
	var hash uint32
	for _, b := range addr.IP {
		hash = hash*31 + uint32(b)
	}
	hash = hash*31 + uint32(addr.Port)
	return hash
}

// Stop stops the UDP manager
func (m *UDPManager) Stop() {
	// Ports are released when tunnels are closed
}
