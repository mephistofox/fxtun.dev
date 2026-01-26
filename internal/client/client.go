package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/protocol"
)

// Client is the tunnel client
type Client struct {
	cfg    *config.ClientConfig
	log    zerolog.Logger
	events *EventEmitter

	conn          net.Conn
	session       *yamux.Session
	controlStream net.Conn
	controlCodec  *protocol.Codec

	clientID  string
	sessionID string

	tunnels   map[string]*ActiveTunnel
	tunnelsMu sync.RWMutex

	pendingRequests map[string]chan *protocol.TunnelCreatedMessage
	pendingMu       sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	reconnecting bool
	reconnectMu  sync.Mutex
	mu           sync.Mutex // for writing to control stream
}

// ActiveTunnel represents an active tunnel on the client side
type ActiveTunnel struct {
	ID         string
	Config     config.TunnelConfig
	URL        string // For HTTP tunnels
	RemoteAddr string // For TCP/UDP tunnels
	Connected  time.Time
}

// New creates a new client
func New(cfg *config.ClientConfig, log zerolog.Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		cfg:             cfg,
		log:             log.With().Str("component", "client").Logger(),
		events:          NewEventEmitter(),
		tunnels:         make(map[string]*ActiveTunnel),
		pendingRequests: make(map[string]chan *protocol.TunnelCreatedMessage),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Events returns the event emitter for subscribing to client events
func (c *Client) Events() *EventEmitter {
	return c.events
}

// Connect connects to the server
func (c *Client) Connect() error {
	c.log.Info().Str("server", c.cfg.Server.Address).Msg("Connecting to server")
	c.events.EmitType(EventConnecting)

	// Dial server
	conn, err := net.DialTimeout("tcp", c.cfg.Server.Address, 30*time.Second)
	if err != nil {
		c.events.EmitError(err)
		return fmt.Errorf("dial server: %w", err)
	}
	c.conn = conn

	// Create yamux session FIRST (client mode) with optimized config
	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = true
	yamuxCfg.KeepAliveInterval = 10 * time.Second
	yamuxCfg.MaxStreamWindowSize = 1024 * 1024 // 1MB window for better throughput
	c.session, err = yamux.Client(conn, yamuxCfg)
	if err != nil {
		conn.Close()
		return fmt.Errorf("create yamux session: %w", err)
	}

	// Open control stream (first stream)
	c.controlStream, err = c.session.Open()
	if err != nil {
		c.session.Close()
		return fmt.Errorf("open control stream: %w", err)
	}

	c.controlCodec = protocol.NewCodec(c.controlStream, c.controlStream)

	// Authenticate
	if err := c.authenticate(); err != nil {
		c.session.Close()
		return fmt.Errorf("authenticate: %w", err)
	}

	c.log.Info().Str("client_id", c.clientID).Msg("Connected to server")
	c.events.EmitWithPayload(EventConnected, map[string]interface{}{
		"client_id":  c.clientID,
		"session_id": c.sessionID,
		"server":     c.cfg.Server.Address,
	})

	// Start message handler
	c.wg.Add(1)
	go c.handleMessages()

	// Start stream acceptor
	c.wg.Add(1)
	go c.acceptStreams()

	// Start keepalive
	c.wg.Add(1)
	go c.keepalive()

	// Request tunnels from config
	for _, tunnelCfg := range c.cfg.Tunnels {
		if err := c.RequestTunnel(tunnelCfg); err != nil {
			c.log.Error().Err(err).Str("name", tunnelCfg.Name).Msg("Failed to request tunnel")
		}
	}

	return nil
}

func (c *Client) authenticate() error {
	authMsg := &protocol.AuthMessage{
		Message:   protocol.NewMessage(protocol.MsgAuth),
		Token:     c.cfg.Server.Token,
		ClientID:  generateID(),
		UserAgent: "fxtunnel-client/1.0",
	}

	if err := c.controlCodec.Encode(authMsg); err != nil {
		return fmt.Errorf("send auth: %w", err)
	}

	// Read response
	c.controlStream.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer c.controlStream.SetReadDeadline(time.Time{})

	data, baseMsg, err := c.controlCodec.DecodeRaw()
	if err != nil {
		return fmt.Errorf("read auth result: %w", err)
	}

	if baseMsg.Type != protocol.MsgAuthResult {
		return fmt.Errorf("unexpected message type: %s", baseMsg.Type)
	}

	parsed, err := protocol.ParseMessage(data, protocol.MsgAuthResult)
	if err != nil {
		return fmt.Errorf("parse auth result: %w", err)
	}

	result := parsed.(*protocol.AuthResultMessage)
	if !result.Success {
		return fmt.Errorf("authentication failed: %s", result.Error)
	}

	c.clientID = result.ClientID
	c.sessionID = result.SessionID

	return nil
}

// RequestTunnel requests a new tunnel
func (c *Client) RequestTunnel(tunnelCfg config.TunnelConfig) error {
	requestID := generateID()

	req := &protocol.TunnelRequestMessage{
		Message:    protocol.NewMessage(protocol.MsgTunnelRequest),
		TunnelType: protocol.TunnelType(tunnelCfg.Type),
		Name:       tunnelCfg.Name,
		LocalPort:  tunnelCfg.LocalPort,
		RemotePort: tunnelCfg.RemotePort,
		Subdomain:  tunnelCfg.Subdomain,
	}
	req.RequestID = requestID

	// Create response channel
	respChan := make(chan *protocol.TunnelCreatedMessage, 1)
	c.pendingMu.Lock()
	c.pendingRequests[requestID] = respChan
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pendingRequests, requestID)
		c.pendingMu.Unlock()
	}()

	if err := c.sendControl(req); err != nil {
		return fmt.Errorf("send tunnel request: %w", err)
	}

	// Wait for response
	select {
	case resp := <-respChan:
		tunnel := &ActiveTunnel{
			ID:         resp.TunnelID,
			Config:     tunnelCfg,
			URL:        resp.URL,
			RemoteAddr: resp.RemoteAddr,
			Connected:  time.Now(),
		}

		c.tunnelsMu.Lock()
		c.tunnels[resp.TunnelID] = tunnel
		c.tunnelsMu.Unlock()

		// Emit tunnel created event
		c.events.EmitTunnelCreated(tunnel)

		if resp.URL != "" {
			c.log.Info().
				Str("name", tunnelCfg.Name).
				Str("url", resp.URL).
				Msg("HTTP tunnel created")
		} else {
			c.log.Info().
				Str("name", tunnelCfg.Name).
				Str("addr", resp.RemoteAddr).
				Msg("Tunnel created")
		}

		return nil

	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for tunnel response")

	case <-c.ctx.Done():
		return fmt.Errorf("client closed")
	}
}

func (c *Client) sendControl(msg any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.controlCodec.Encode(msg)
}

func (c *Client) handleMessages() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		data, baseMsg, err := c.controlCodec.DecodeRaw()
		if err != nil {
			c.log.Debug().Err(err).Msg("Read error")
			c.handleDisconnect()
			return
		}

		switch baseMsg.Type {
		case protocol.MsgTunnelCreated:
			c.handleTunnelCreated(data)
		case protocol.MsgTunnelError:
			c.handleTunnelError(data)
		case protocol.MsgTunnelClosed:
			c.handleTunnelClosed(data)
		case protocol.MsgPing:
			c.handlePing()
		case protocol.MsgPong:
			// Keepalive response
		case protocol.MsgError:
			c.handleError(data)
		default:
			c.log.Warn().Str("type", string(baseMsg.Type)).Msg("Unknown message type")
		}
	}
}

func (c *Client) handleTunnelCreated(data []byte) {
	parsed, err := protocol.ParseMessage(data, protocol.MsgTunnelCreated)
	if err != nil {
		c.log.Error().Err(err).Msg("Failed to parse tunnel created")
		return
	}
	msg := parsed.(*protocol.TunnelCreatedMessage)

	c.pendingMu.Lock()
	if ch, ok := c.pendingRequests[msg.RequestID]; ok {
		ch <- msg
	}
	c.pendingMu.Unlock()
}

func (c *Client) handleTunnelError(data []byte) {
	parsed, err := protocol.ParseMessage(data, protocol.MsgTunnelError)
	if err != nil {
		c.log.Error().Err(err).Msg("Failed to parse tunnel error")
		return
	}
	msg := parsed.(*protocol.TunnelErrorMessage)

	c.log.Error().
		Str("tunnel_id", msg.TunnelID).
		Str("code", msg.Code).
		Str("error", msg.Error).
		Msg("Tunnel error")
}

func (c *Client) handleTunnelClosed(data []byte) {
	parsed, err := protocol.ParseMessage(data, protocol.MsgTunnelClosed)
	if err != nil {
		c.log.Error().Err(err).Msg("Failed to parse tunnel closed")
		return
	}
	msg := parsed.(*protocol.TunnelClosedMessage)

	c.tunnelsMu.Lock()
	delete(c.tunnels, msg.TunnelID)
	c.tunnelsMu.Unlock()

	// Emit tunnel closed event
	c.events.EmitTunnelClosed(msg.TunnelID)

	c.log.Info().Str("tunnel_id", msg.TunnelID).Msg("Tunnel closed")
}

func (c *Client) handlePing() {
	pong := &protocol.PongMessage{
		Message: protocol.NewMessage(protocol.MsgPong),
	}
	c.sendControl(pong)
}

func (c *Client) handleError(data []byte) {
	parsed, err := protocol.ParseMessage(data, protocol.MsgError)
	if err != nil {
		c.log.Error().Err(err).Msg("Failed to parse error")
		return
	}
	msg := parsed.(*protocol.ErrorMessage)

	c.log.Error().
		Str("code", msg.Code).
		Str("error", msg.Error).
		Bool("fatal", msg.Fatal).
		Msg("Server error")

	if msg.Fatal {
		c.Close()
	}
}

func (c *Client) acceptStreams() {
	defer c.wg.Done()

	for {
		stream, err := c.session.Accept()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return
			default:
				c.log.Debug().Err(err).Msg("Stream accept error")
				c.handleDisconnect()
				return
			}
		}

		go c.handleStream(stream)
	}
}

func (c *Client) handleStream(stream net.Conn) {
	defer stream.Close()

	// Read connection info
	streamCodec := protocol.NewCodec(stream, stream)

	var msg protocol.NewConnectionMessage
	if err := streamCodec.Decode(&msg); err != nil {
		c.log.Error().Err(err).Msg("Failed to read connection info")
		return
	}

	// Find tunnel
	c.tunnelsMu.RLock()
	tunnel, exists := c.tunnels[msg.TunnelID]
	c.tunnelsMu.RUnlock()

	if !exists {
		c.log.Warn().Str("tunnel_id", msg.TunnelID).Msg("Unknown tunnel")
		return
	}

	// Connect to local service with IPv4/IPv6 fallback
	local, err := dialLocalWithFallback(c.log, tunnel.Config.LocalAddr, tunnel.Config.LocalPort, 10*time.Second)
	if err != nil {
		c.log.Error().Err(err).Int("port", tunnel.Config.LocalPort).Msg("Failed to connect to local service")
		return
	}
	defer local.Close()

	c.log.Debug().
		Str("tunnel", tunnel.Config.Name).
		Str("remote", msg.RemoteAddr).
		Str("local", local.RemoteAddr().String()).
		Msg("Forwarding connection")

	// Bidirectional copy
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(local, stream)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(stream, local)
		done <- struct{}{}
	}()

	<-done
}

func (c *Client) keepalive() {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			ping := &protocol.PingMessage{
				Message: protocol.NewMessage(protocol.MsgPing),
			}
			if err := c.sendControl(ping); err != nil {
				c.log.Debug().Err(err).Msg("Failed to send ping")
				c.handleDisconnect()
				return
			}
		}
	}
}

func (c *Client) handleDisconnect() {
	c.reconnectMu.Lock()
	if c.reconnecting {
		c.reconnectMu.Unlock()
		return
	}
	c.reconnecting = true
	c.reconnectMu.Unlock()

	c.log.Warn().Msg("Disconnected from server")
	c.events.EmitType(EventDisconnected)

	if !c.cfg.Reconnect.Enabled {
		c.Close()
		return
	}

	// Start reconnection
	go c.reconnect()
}

func (c *Client) reconnect() {
	attempts := 0
	interval := c.cfg.Reconnect.Interval
	if interval == 0 {
		interval = 5 * time.Second
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		attempts++
		if c.cfg.Reconnect.MaxAttempts > 0 && attempts > c.cfg.Reconnect.MaxAttempts {
			c.log.Error().Msg("Max reconnection attempts reached")
			c.Close()
			return
		}

		c.log.Info().Int("attempt", attempts).Msg("Attempting to reconnect...")
		c.events.EmitWithPayload(EventReconnecting, map[string]interface{}{
			"attempt": attempts,
		})

		// Close existing connections
		if c.controlStream != nil {
			c.controlStream.Close()
		}
		if c.session != nil {
			c.session.Close()
		}
		if c.conn != nil {
			c.conn.Close()
		}

		// Clear tunnels
		c.tunnelsMu.Lock()
		c.tunnels = make(map[string]*ActiveTunnel)
		c.tunnelsMu.Unlock()

		// Try to connect
		if err := c.Connect(); err != nil {
			c.log.Error().Err(err).Msg("Reconnection failed")
			time.Sleep(interval)
			continue
		}

		c.reconnectMu.Lock()
		c.reconnecting = false
		c.reconnectMu.Unlock()

		c.log.Info().Msg("Reconnected successfully")
		return
	}
}

// GetTunnels returns a list of active tunnels
func (c *Client) GetTunnels() []*ActiveTunnel {
	c.tunnelsMu.RLock()
	defer c.tunnelsMu.RUnlock()

	tunnels := make([]*ActiveTunnel, 0, len(c.tunnels))
	for _, t := range c.tunnels {
		tunnels = append(tunnels, t)
	}
	return tunnels
}

// CloseTunnel closes a specific tunnel by ID
func (c *Client) CloseTunnel(tunnelID string) error {
	c.tunnelsMu.RLock()
	_, exists := c.tunnels[tunnelID]
	c.tunnelsMu.RUnlock()

	if !exists {
		return fmt.Errorf("tunnel not found: %s", tunnelID)
	}

	msg := &protocol.TunnelCloseMessage{
		Message:  protocol.NewMessage(protocol.MsgTunnelClose),
		TunnelID: tunnelID,
	}

	if err := c.sendControl(msg); err != nil {
		return fmt.Errorf("send tunnel close: %w", err)
	}

	c.log.Info().Str("tunnel_id", tunnelID).Msg("Tunnel close requested")
	return nil
}

// Wait waits for the client to close
func (c *Client) Wait() {
	c.wg.Wait()
}

// Close closes the client
func (c *Client) Close() {
	c.cancel()

	if c.controlStream != nil {
		c.controlStream.Close()
	}
	if c.session != nil {
		c.session.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}

	c.log.Info().Msg("Client closed")
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
