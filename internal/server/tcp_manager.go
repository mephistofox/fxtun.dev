package server

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/protocol"
)

// TCPManager manages TCP tunnel ports
type TCPManager struct {
	server     *Server
	log        zerolog.Logger
	usedPorts  map[int]bool
	mu         sync.Mutex
}

// NewTCPManager creates a new TCP manager
func NewTCPManager(server *Server, log zerolog.Logger) *TCPManager {
	return &TCPManager{
		server:    server,
		log:       log.With().Str("component", "tcp_manager").Logger(),
		usedPorts: make(map[int]bool),
	}
}

// AllocatePort allocates a port for a TCP tunnel
func (m *TCPManager) AllocatePort(requestedPort int) (int, net.Listener, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	portRange := m.server.cfg.Server.TCPPortRange

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
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", requestedPort))
		if err != nil {
			return 0, nil, fmt.Errorf("failed to bind port %d: %w", requestedPort, err)
		}

		m.usedPorts[requestedPort] = true
		return requestedPort, listener, nil
	}

	// Auto-assign port
	for port := portRange.Min; port <= portRange.Max; port++ {
		if m.usedPorts[port] {
			continue
		}

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}

		m.usedPorts[port] = true
		return port, listener, nil
	}

	return 0, nil, fmt.Errorf("no available ports in range %d-%d", portRange.Min, portRange.Max)
}

// ReleasePort releases a previously allocated port
func (m *TCPManager) ReleasePort(port int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.usedPorts, port)
}

// AcceptConnections accepts connections on a tunnel listener
func (m *TCPManager) AcceptConnections(tunnel *Tunnel, client *Client) {
	defer func() {
		m.ReleasePort(tunnel.RemotePort)
		if tunnel.listener != nil {
			tunnel.listener.Close()
		}
	}()

	for {
		conn, err := tunnel.listener.Accept()
		if err != nil {
			select {
			case <-client.ctx.Done():
				return
			default:
				m.log.Debug().Err(err).Int("port", tunnel.RemotePort).Msg("Accept failed")
				return
			}
		}

		go m.handleConnection(conn, tunnel, client)
	}
}

func (m *TCPManager) handleConnection(conn net.Conn, tunnel *Tunnel, client *Client) {
	defer conn.Close()

	tuneTCPConn(conn)

	// Open stream to client
	stream, err := client.OpenStream()
	if err != nil {
		m.log.Error().Err(err).Msg("Failed to open stream to client")
		return
	}
	defer stream.Close()

	// Send connection info through the stream
	connID := generateID()
	newConn := &protocol.NewConnectionMessage{
		Message:      protocol.NewMessage(protocol.MsgNewConnection),
		TunnelID:     tunnel.ID,
		ConnectionID: connID,
		RemoteAddr:   conn.RemoteAddr().String(),
	}

	streamCodec := protocol.NewCodec(stream, stream)
	if err := streamCodec.Encode(newConn); err != nil {
		m.log.Error().Err(err).Msg("Failed to send connection info")
		return
	}

	// Bidirectional copy with large buffers
	done := make(chan struct{}, 2)

	go func() {
		bp := proxyBufPool.Get().(*[]byte)
		io.CopyBuffer(stream, conn, *bp)
		proxyBufPool.Put(bp)
		done <- struct{}{}
	}()

	go func() {
		bp := proxyBufPool.Get().(*[]byte)
		io.CopyBuffer(conn, stream, *bp)
		proxyBufPool.Put(bp)
		done <- struct{}{}
	}()

	<-done
	// Close both to unblock the other goroutine
	conn.Close()
	stream.Close()
	<-done

	m.log.Debug().
		Str("tunnel_id", tunnel.ID).
		Str("remote", conn.RemoteAddr().String()).
		Msg("TCP connection completed")
}

// Stop stops the TCP manager
func (m *TCPManager) Stop() {
	// Ports are released when tunnels are closed
}
