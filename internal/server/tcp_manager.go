package server

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/protocol"
)

// TCPManager manages TCP tunnel ports
type TCPManager struct {
	server *Server
	log    zerolog.Logger
	ports  *PortAllocator
}

// NewTCPManager creates a new TCP manager
func NewTCPManager(server *Server, log zerolog.Logger) *TCPManager {
	return &TCPManager{
		server: server,
		log:    log.With().Str("component", "tcp_manager").Logger(),
		ports:  NewPortAllocator(server.cfg.Server.TCPPortRange),
	}
}

// AllocatePort allocates a port for a TCP tunnel
func (m *TCPManager) AllocatePort(requestedPort int) (int, net.Listener, error) {
	port, err := m.ports.Allocate(requestedPort)
	if err != nil {
		return 0, nil, err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		m.ports.Release(port)
		return 0, nil, fmt.Errorf("failed to bind port %d: %w", port, err)
	}

	return port, listener, nil
}

// ReleasePort releases a previously allocated port
func (m *TCPManager) ReleasePort(port int) {
	m.ports.Release(port)
}

// AcceptConnections accepts connections on a tunnel listener
func (m *TCPManager) AcceptConnections(tunnel *Tunnel, client *Client) {
	defer func() {
		m.ReleasePort(tunnel.RemotePort)
		if tunnel.listener != nil {
			tunnel.listener.Close()
		}
	}()

	const maxTempErrors = 10
	tempErrors := 0

	for {
		conn, err := tunnel.listener.Accept()
		if err != nil {
			select {
			case <-client.ctx.Done():
				return
			default:
			}

			// Check for temporary errors (e.g. EMFILE, ENFILE)
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				tempErrors++
				m.log.Warn().Err(err).Int("port", tunnel.RemotePort).Int("consecutive_temp_errors", tempErrors).Msg("Temporary accept error, retrying")
				if tempErrors >= maxTempErrors {
					m.log.Error().Int("port", tunnel.RemotePort).Msg("Too many consecutive temporary accept errors, stopping")
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			m.log.Debug().Err(err).Int("port", tunnel.RemotePort).Msg("Accept failed")
			return
		}

		tempErrors = 0
		go m.handleConnection(conn, tunnel, client)
	}
}

func (m *TCPManager) handleConnection(conn net.Conn, tunnel *Tunnel, client *Client) {
	m.server.activeConns.Add(1)
	defer m.server.activeConns.Done()
	defer conn.Close()

	tuneTCPConn(conn)

	// Open stream to client
	stream, err := client.OpenStream()
	if err != nil {
		m.log.Error().Err(err).Msg("Failed to open stream to client")
		return
	}
	defer stream.Close()

	// Send binary stream header
	if err := protocol.WriteStreamHeader(stream, tunnel.ID, conn.RemoteAddr().String()); err != nil {
		m.log.Error().Err(err).Msg("Failed to send connection info")
		return
	}

	// Apply bandwidth throttling if configured
	var streamReader io.Reader = stream
	var streamWriter io.Writer = stream
	if client.bwLimiter != nil {
		streamReader = client.bwLimiter.Reader(stream)
		streamWriter = client.bwLimiter.Writer(stream)
	}

	// Bidirectional copy with large buffers
	done := make(chan struct{}, 2)

	go func() {
		bp := proxyBufPool.Get().(*[]byte)
		_, _ = io.CopyBuffer(streamWriter, conn, *bp)
		proxyBufPool.Put(bp)
		done <- struct{}{}
	}()

	go func() {
		bp := proxyBufPool.Get().(*[]byte)
		_, _ = io.CopyBuffer(conn, streamReader, *bp)
		proxyBufPool.Put(bp)
		done <- struct{}{}
	}()

	<-done
	// Close both to unblock the other goroutine
	_ = conn.Close()
	_ = stream.Close()
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
