package core

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/inspect"
	"github.com/mephistofox/fxtunnel/internal/protocol"
)

const (
	// yamuxMaxStreamWindowSize is the yamux stream window size for high throughput.
	yamuxMaxStreamWindowSize = 16 * 1024 * 1024 // 16MB

	// yamuxKeepAliveInterval is the interval between yamux keepalive probes.
	yamuxKeepAliveInterval = 10 * time.Second

	// yamuxConnectionWriteTimeout is the timeout for writing to a yamux connection.
	yamuxConnectionWriteTimeout = 30 * time.Second

	// dialTimeout is the maximum time to wait when connecting to the server.
	dialTimeout = 30 * time.Second

	// authResponseTimeout is the maximum time to wait for an auth response from the server.
	authResponseTimeout = 30 * time.Second

	// tunnelResponseTimeout is the maximum time to wait for a tunnel creation response.
	tunnelResponseTimeout = 30 * time.Second

	// keepaliveInterval is the interval between client-side keepalive pings.
	keepaliveInterval = 30 * time.Second

	// localDialTimeout is the maximum time to wait when connecting to a local service.
	localDialTimeout = 5 * time.Second

	// trafficStatsInterval is the interval for emitting traffic statistics.
	trafficStatsInterval = 2 * time.Second

	// dataConnectionCount is the number of additional data connections to open (total = 1 primary + N data).
	dataConnectionCount = 16

	// maxOverflowGoroutines caps the number of goroutines spawned when the worker pool is full.
	maxOverflowGoroutines = 1024

	// defaultReconnectInterval is the default base interval for reconnection attempts.
	defaultReconnectInterval = 5 * time.Second

	// maxReconnectBackoff is the maximum backoff duration between reconnection attempts.
	maxReconnectBackoff = 2 * time.Minute
)

// TokenRefresher is a callback function that refreshes the authentication token.
// It receives the server address and should return a new token or an error.
type TokenRefresher func(serverAddr string) (newToken string, err error)

// Client is the tunnel client
type Client struct {
	cfg    *config.ClientConfig
	log    zerolog.Logger
	events *EventEmitter

	conn          net.Conn
	session       *yamux.Session
	controlStream net.Conn
	controlCodec  *protocol.Codec

	// Multi-session pool: additional data connections for parallelism
	dataSessions    []*yamux.Session
	dataConns       []net.Conn
	dataSessionMu   sync.Mutex
	maxDataSessions int // server-enforced limit (0 = use default)

	clientID      string
	sessionID     string
	sessionSecret string

	tunnels   map[string]*ActiveTunnel
	tunnelsMu sync.RWMutex

	pendingRequests map[string]chan *protocol.TunnelCreatedMessage
	pendingMu       sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	streamWorkers  chan net.Conn // bounded worker pool for incoming streams
	overflowCount  atomic.Int32 // current overflow goroutine count

	version string // protocol version sent to server during auth

	closed    atomic.Bool
	closeOnce sync.Once

	reconnecting   bool
	reconnectMu    sync.Mutex
	mu             sync.Mutex // for writing to control stream
	tokenRefresher TokenRefresher
	tokenMu        sync.RWMutex

	lastPong atomic.Int64 // unix nano timestamp of last pong received

	inspector  *Inspector
	inspectMgr *inspect.Manager

	// Edge node info (set after redirect)
	nodeName      string
	nodeRegion    string
	redirectCount int

	// Auto-close timers
	autoCloseTimers   map[string]*autoCloseTimer  // tunnelID -> timer
	maxLifetimeTimers map[string]*maxLifetimeTimer // tunnelID -> timer
	timersMu          sync.Mutex
}

// redirectError indicates the client should reconnect to a different address.
type redirectError struct {
	addr string
}

func (e *redirectError) Error() string {
	return "redirect to " + e.addr
}

// ActiveTunnel represents an active tunnel on the client side
type ActiveTunnel struct {
	ID         string
	Config     config.TunnelConfig
	URL        string // For HTTP tunnels
	HTTPSURL   string // For HTTP tunnels (HTTPS)
	RemoteAddr string // For TCP/UDP tunnels
	Connected  time.Time

	BytesSent     atomic.Int64
	BytesReceived atomic.Int64

	// Security status (echoed from server on tunnel creation)
	BasicAuthEnabled bool
	AllowIPsCount    int
	AutoClose        string
	MaxLifetime      string
}

// countingWriter wraps an io.Writer and counts bytes written.
type countingWriter struct {
	w     io.Writer
	count *atomic.Int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.count.Add(int64(n))
	return n, err
}

// New creates a new client
func New(cfg *config.ClientConfig, log zerolog.Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		cfg:               cfg,
		log:               log.With().Str("component", "client").Logger(),
		events:            NewEventEmitter(),
		tunnels:           make(map[string]*ActiveTunnel),
		pendingRequests:   make(map[string]chan *protocol.TunnelCreatedMessage),
		autoCloseTimers:   make(map[string]*autoCloseTimer),
		maxLifetimeTimers: make(map[string]*maxLifetimeTimer),
		ctx:               ctx,
		cancel:            cancel,
	}
}

// SetVersion sets the client version for protocol negotiation.
func (c *Client) SetVersion(v string) { c.version = v }

// Events returns the event emitter for subscribing to client events
func (c *Client) Events() *EventEmitter {
	return c.events
}

// SetTokenRefresher sets a callback function that will be called when the token expires.
// The callback should return a new valid token.
func (c *Client) SetTokenRefresher(refresher TokenRefresher) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.tokenRefresher = refresher
}

// UpdateToken updates the token used for authentication.
// This is useful when the token has been refreshed externally.
func (c *Client) UpdateToken(newToken string) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.cfg.Server.Token = newToken
}

// dialServer establishes a TCP connection to the server, with TLS if not in insecure mode.
func (c *Client) dialServer() (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", c.cfg.Server.Address, dialTimeout)
	if err != nil {
		return nil, err
	}
	tuneTCPConn(conn)

	if c.cfg.Server.Insecure {
		return conn, nil
	}

	// Wrap with TLS
	host, _, err := net.SplitHostPort(c.cfg.Server.Address)
	if err != nil {
		host = c.cfg.Server.Address
	}
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: !c.cfg.Server.TLSVerify,
		MinVersion:         tls.VersionTLS12,
	})
	if err := tlsConn.HandshakeContext(c.ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("TLS handshake: %w", err)
	}
	return tlsConn, nil
}

// Connect connects to the server
func (c *Client) Connect() error {
	c.log.Info().Str("server", c.cfg.Server.Address).Msg("Connecting to server")
	c.events.EmitType(EventConnecting)

	// Dial server
	conn, err := c.dialServer()
	if err != nil {
		c.events.EmitError(err)
		return fmt.Errorf("dial server: %w", err)
	}
	c.conn = conn

	// Negotiate compression before yamux
	rwc, compressed, err := protocol.NegotiateCompression(conn, c.cfg.Server.Compression, false)
	if err != nil {
		conn.Close()
		c.events.EmitError(err)
		return fmt.Errorf("compression negotiation: %w", err)
	}
	if compressed {
		c.log.Info().Msg("Compression enabled (zstd)")
	}

	// Create yamux session FIRST (client mode) with optimized config
	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = true
	yamuxCfg.KeepAliveInterval = yamuxKeepAliveInterval
	yamuxCfg.MaxStreamWindowSize = yamuxMaxStreamWindowSize
	yamuxCfg.ConnectionWriteTimeout = yamuxConnectionWriteTimeout
	c.session, err = yamux.Client(rwc, yamuxCfg)
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
		c.conn.Close()

		// Handle edge node redirect: reconnect to the node
		var rErr *redirectError
		if errors.As(err, &rErr) {
			c.redirectCount++
			if c.redirectCount > 3 {
				return fmt.Errorf("too many redirects (max 3)")
			}
			c.log.Info().Str("addr", rErr.addr).Msg("Reconnecting to edge node")
			c.cfg.Server.Address = rErr.addr
			return c.Connect()
		}

		return fmt.Errorf("authenticate: %w", err)
	}

	c.log.Info().Str("client_id", c.clientID).Msg("Connected to server")
	c.events.EmitWithPayload(EventConnected, map[string]interface{}{
		"client_id":  c.clientID,
		"session_id": c.sessionID,
		"server":     c.cfg.Server.Address,
	})

	// Start stream worker pool
	numWorkers := runtime.NumCPU() * 4
	c.streamWorkers = make(chan net.Conn, numWorkers)
	for i := 0; i < numWorkers; i++ {
		c.wg.Add(1)
		go c.streamWorker()
	}

	// Start message handler
	c.wg.Add(1)
	go c.handleMessages()

	// Start stream acceptor
	c.wg.Add(1)
	go c.acceptStreams()

	// Start keepalive
	c.wg.Add(1)
	go c.keepalive()

	// Open additional data connections for parallelism
	if c.sessionSecret != "" {
		c.openDataConnections()
	}

	// Request tunnels from config
	for _, tunnelCfg := range c.cfg.Tunnels {
		if err := c.RequestTunnel(tunnelCfg); err != nil {
			c.log.Error().Err(err).Str("name", tunnelCfg.Name).Msg("Failed to request tunnel")
		}
	}

	if c.inspector != nil {
		c.inspector.SetTunnels(c.tunnels, &c.tunnelsMu)
		if err := c.inspector.Start(c.ctx); err != nil {
			c.log.Warn().Err(err).Msg("Failed to start inspector")
		}
	}

	return nil
}

func (c *Client) authenticate() error {
	c.tokenMu.RLock()
	token := c.cfg.Server.Token
	c.tokenMu.RUnlock()

	authMsg := &protocol.AuthMessage{
		Message:   protocol.NewMessage(protocol.MsgAuth),
		Token:     token,
		ClientID:  generateID(),
		UserAgent: "fxtunnel-client/1.0",
		Version:   c.version,
	}

	if err := c.controlCodec.Encode(authMsg); err != nil {
		return fmt.Errorf("send auth: %w", err)
	}

	// Read response
	_ = c.controlStream.SetReadDeadline(time.Now().Add(authResponseTimeout))
	defer func() { _ = c.controlStream.SetReadDeadline(time.Time{}) }()

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

	// Handle edge node redirect
	if result.Code == protocol.ErrCodeRedirect && result.RedirectAddr != "" {
		c.log.Info().
			Str("node", result.RedirectNodeID).
			Str("region", result.RedirectRegion).
			Str("addr", result.RedirectAddr).
			Msg("Redirected to edge node")
		c.nodeName = result.RedirectNodeID
		c.nodeRegion = result.RedirectRegion
		c.events.EmitWithPayload(EventRedirected, map[string]interface{}{
			"node":    result.RedirectNodeID,
			"region":  result.RedirectRegion,
			"address": result.RedirectAddr,
		})
		return &redirectError{addr: result.RedirectAddr}
	}

	if !result.Success {
		// Check if the error is due to an expired token
		if result.Code == protocol.ErrCodeTokenExpired {
			return NewAuthError(result.Code, result.Error)
		}
		return fmt.Errorf("authentication failed: %s", result.Error)
	}

	c.clientID = result.ClientID
	c.sessionID = result.SessionID
	c.sessionSecret = result.SessionSecret

	// Apply server-enforced data session limit
	if result.MaxDataSessions > 0 {
		c.maxDataSessions = result.MaxDataSessions
	} else {
		c.maxDataSessions = dataConnectionCount // fallback to default 15
	}

	if result.Capabilities != nil {
		c.log.Debug().
			Bool("inspector_enabled", result.Capabilities.InspectorEnabled).
			Int("max_body_size", result.Capabilities.MaxBodySize).
			Int("max_buffer_entries", result.Capabilities.MaxBufferEntries).
			Msg("Server capabilities received")
	} else {
		c.log.Debug().Msg("No capabilities from server (nil)")
	}
	c.applyCapabilities(result.Capabilities)

	return nil
}

func (c *Client) applyCapabilities(caps *protocol.ClientCapabilities) {
	if caps == nil || !caps.InspectorEnabled || !c.cfg.Inspect.Enabled {
		return
	}

	maxBodySize := c.cfg.Inspect.MaxBodySize
	if caps.MaxBodySize > 0 {
		maxBodySize = caps.MaxBodySize
	}
	maxEntries := c.cfg.Inspect.MaxEntries
	if caps.MaxBufferEntries > 0 {
		maxEntries = caps.MaxBufferEntries
	}

	c.inspectMgr = inspect.NewManager(maxEntries, maxBodySize)
	c.inspector = NewInspector(c.inspectMgr, c.cfg.Inspect.Addr, maxBodySize, c.log)
}

// RequestTunnel requests a new tunnel
func (c *Client) RequestTunnel(tunnelCfg config.TunnelConfig) error {
	requestID := generateID()

	req := &protocol.TunnelRequestMessage{
		Message:       protocol.NewMessage(protocol.MsgTunnelRequest),
		TunnelType:    protocol.TunnelType(tunnelCfg.Type),
		Name:          tunnelCfg.Name,
		LocalPort:     tunnelCfg.LocalPort,
		RemotePort:    tunnelCfg.RemotePort,
		Subdomain:     tunnelCfg.Subdomain,
		BasicAuthHash: tunnelCfg.BasicAuthHash,
		AllowIPs:      tunnelCfg.AllowIPs,
		AutoClose:     tunnelCfg.AutoClose,
		MaxLifetime:   tunnelCfg.MaxLifetime,
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
			ID:               resp.TunnelID,
			Config:           tunnelCfg,
			URL:              resp.URL,
			HTTPSURL:         resp.HTTPSURL,
			RemoteAddr:       resp.RemoteAddr,
			Connected:        time.Now(),
			BasicAuthEnabled: resp.BasicAuthEnabled,
			AllowIPsCount:    resp.AllowIPsCount,
			AutoClose:        resp.AutoClose,
			MaxLifetime:      resp.MaxLifetime,
		}

		c.tunnelsMu.Lock()
		c.tunnels[resp.TunnelID] = tunnel
		c.tunnelsMu.Unlock()

		// Save assigned subdomain/port back to config for reconnect persistence
		if resp.Subdomain != "" && tunnelCfg.Subdomain == "" {
			for i := range c.cfg.Tunnels {
				if c.cfg.Tunnels[i].Name == tunnelCfg.Name && c.cfg.Tunnels[i].Type == tunnelCfg.Type && c.cfg.Tunnels[i].LocalPort == tunnelCfg.LocalPort {
					c.cfg.Tunnels[i].Subdomain = resp.Subdomain
					break
				}
			}
		}
		if resp.RemotePort > 0 && tunnelCfg.RemotePort == 0 {
			for i := range c.cfg.Tunnels {
				if c.cfg.Tunnels[i].Name == tunnelCfg.Name && c.cfg.Tunnels[i].Type == tunnelCfg.Type && c.cfg.Tunnels[i].LocalPort == tunnelCfg.LocalPort {
					c.cfg.Tunnels[i].RemotePort = resp.RemotePort
					break
				}
			}
		}

		// Pre-probe local address synchronously so first connection is instant
		ProbeLocalAddress(c.log, tunnelCfg.LocalAddr, tunnelCfg.LocalPort)

		// Start auto-close timer (idle timeout)
		if tunnelCfg.AutoClose != "" {
			d, _ := parseDuration(tunnelCfg.AutoClose) // already validated by CLI
			tunnelID := resp.TunnelID
			c.timersMu.Lock()
			c.autoCloseTimers[tunnelID] = newAutoCloseTimer(d, func() {
				c.log.Info().
					Str("tunnel_id", tunnelID).
					Str("reason", "idle for "+tunnelCfg.AutoClose).
					Msg("tunnel auto-closed")
				c.closeTunnel(tunnelID)
			})
			c.timersMu.Unlock()
		}

		// Start max-lifetime timer.
		// Note: max-lifetime timer resets on reconnect. This means a tunnel with
		// --max-lifetime 8h that reconnects after 7h gets another full 8h.
		// This is acceptable for MVP — the timer measures "time since last connect".
		if tunnelCfg.MaxLifetime != "" {
			d, _ := parseDuration(tunnelCfg.MaxLifetime) // already validated by CLI
			tunnelID := resp.TunnelID
			c.timersMu.Lock()
			c.maxLifetimeTimers[tunnelID] = newMaxLifetimeTimer(d, func() {
				c.log.Info().
					Str("tunnel_id", tunnelID).
					Str("reason", "max lifetime "+tunnelCfg.MaxLifetime+" reached").
					Msg("tunnel auto-closed")
				c.closeTunnel(tunnelID)
			})
			c.timersMu.Unlock()
		}

		// Emit tunnel created event
		c.events.EmitTunnelCreated(tunnel)

		// Start periodic traffic stats emitter
		go c.emitTrafficStats(tunnel)

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

	case <-time.After(tunnelResponseTimeout):
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
			c.lastPong.Store(time.Now().UnixNano())
		case protocol.MsgServerShutdown:
			c.handleServerShutdown(data)
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

	// Capture final traffic stats before removing tunnel
	var bytesSent, bytesReceived int64
	c.tunnelsMu.Lock()
	if tunnel, ok := c.tunnels[msg.TunnelID]; ok {
		bytesSent = tunnel.BytesSent.Load()
		bytesReceived = tunnel.BytesReceived.Load()
	}
	delete(c.tunnels, msg.TunnelID)
	c.tunnelsMu.Unlock()

	// Stop timers for this tunnel
	c.stopTunnelTimers(msg.TunnelID)

	// Emit tunnel closed event with final traffic stats
	c.events.EmitWithPayload(EventTunnelClosed, map[string]interface{}{
		"tunnel_id":      msg.TunnelID,
		"bytes_sent":     bytesSent,
		"bytes_received": bytesReceived,
	})

	c.log.Info().Str("tunnel_id", msg.TunnelID).Msg("Tunnel closed")
}

func (c *Client) handlePing() {
	pong := &protocol.PongMessage{
		Message: protocol.NewMessage(protocol.MsgPong),
	}
	_ = c.sendControl(pong)
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

func (c *Client) handleServerShutdown(data []byte) {
	parsed, err := protocol.ParseMessage(data, protocol.MsgServerShutdown)
	if err != nil {
		c.log.Error().Err(err).Msg("Failed to parse server shutdown")
		return
	}
	msg := parsed.(*protocol.ServerShutdownMessage)

	c.log.Warn().Str("reason", msg.Reason).Msg("Server is shutting down")
	c.events.EmitWithPayload(EventDisconnected, map[string]interface{}{
		"reason": "server_shutdown",
	})

	// Delay reconnect to give server time to fully shut down
	c.reconnectMu.Lock()
	c.reconnecting = true
	c.reconnectMu.Unlock()

	go func() {
		time.Sleep(5 * time.Second)
		c.reconnectMu.Lock()
		c.reconnecting = false
		c.reconnectMu.Unlock()
		c.handleDisconnect()
	}()
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

		select {
		case c.streamWorkers <- stream:
			// Dispatched to worker pool
		default:
			// Pool full — use overflow goroutine with cap
			if c.overflowCount.Load() >= maxOverflowGoroutines {
				c.log.Warn().Int32("overflow", c.overflowCount.Load()).Msg("Overflow goroutine limit reached, dropping stream")
				stream.Close()
				continue
			}
			c.overflowCount.Add(1)
			go func() {
				defer c.overflowCount.Add(-1)
				c.handleStream(stream)
			}()
		}
	}
}

func (c *Client) acceptDataStreams(session *yamux.Session) {
	defer c.wg.Done()

	for {
		stream, err := session.Accept()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return
			default:
				c.log.Debug().Err(err).Msg("Data session stream accept error")
				return
			}
		}

		select {
		case c.streamWorkers <- stream:
		default:
			if c.overflowCount.Load() >= maxOverflowGoroutines {
				stream.Close()
				continue
			}
			c.overflowCount.Add(1)
			go func() {
				defer c.overflowCount.Add(-1)
				c.handleStream(stream)
			}()
		}
	}
}

func (c *Client) streamWorker() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case stream := <-c.streamWorkers:
			c.handleStream(stream)
		}
	}
}

func (c *Client) handleStream(stream net.Conn) {
	defer stream.Close()

	// Read binary stream header
	hdr, err := protocol.ReadStreamHeader(stream)
	if err != nil {
		if c.ctx.Err() == nil && !c.closed.Load() && !errors.Is(err, io.EOF) {
			c.log.Error().Err(err).Msg("Failed to read connection info")
		}
		return
	}

	// Find tunnel (may arrive before control channel registers it, so retry briefly)
	var tunnel *ActiveTunnel
	for i := 0; i < 50; i++ {
		c.tunnelsMu.RLock()
		t, exists := c.tunnels[hdr.TunnelID]
		c.tunnelsMu.RUnlock()
		if exists {
			tunnel = t
			break
		}
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
	if tunnel == nil {
		c.log.Warn().Str("tunnel_id", hdr.TunnelID).Msg("Unknown tunnel")
		return
	}

	// Record activity for auto-close timer
	c.timersMu.Lock()
	if t, ok := c.autoCloseTimers[hdr.TunnelID]; ok {
		t.recordActivity()
	}
	c.timersMu.Unlock()

	// UDP tunnels use a different proxy path
	if tunnel.Config.Type == "udp" {
		c.handleUDPStream(stream, tunnel)
		return
	}

	// Connect to local service with IPv4/IPv6 fallback
	local, err := dialLocalWithFallback(c.log, tunnel.Config.LocalAddr, tunnel.Config.LocalPort, localDialTimeout)
	if err != nil {
		c.log.Error().Err(err).Int("port", tunnel.Config.LocalPort).Msg("Failed to connect to local service")
		return
	}
	defer local.Close()

	c.log.Debug().
		Str("tunnel", tunnel.Config.Name).
		Str("remote", hdr.RemoteAddr).
		Str("local", local.RemoteAddr().String()).
		Msg("Forwarding connection")

	// For HTTP tunnels, peek at the request line and print it
	var streamReader io.Reader = stream
	var reqStart time.Time
	var httpMethod, httpPath string
	if tunnel.Config.Type == "http" {
		br := bufio.NewReaderSize(stream, 4096)
		if line, err := br.ReadString('\n'); err == nil {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				httpMethod = parts[0]
				httpPath = parts[1]
				reqStart = time.Now()
			}
			// Prepend consumed line back
			streamReader = io.MultiReader(strings.NewReader(line), br)
		} else {
			streamReader = br
		}
	}

	// Bidirectional copy with byte counting and large buffers
	c.log.Debug().
		Str("tunnel_type", tunnel.Config.Type).
		Bool("inspector_exists", c.inspector != nil).
		Bool("inspectmgr_exists", c.inspectMgr != nil).
		Msg("handleStream capture check")
	if tunnel.Config.Type == "http" && c.inspector != nil {
		cap := NewCapture(tunnel.ID, tunnel.Config.Name, c.inspectMgr.MaxBodySize())

		// Parse HTTP request from the stream (server sends a complete HTTP request).
		reqBuf := bufio.NewReader(streamReader)
		httpReq, reqErr := http.ReadRequest(reqBuf)
		if reqErr != nil {
			// Not valid HTTP — fall back to raw bidirectional copy without capture.
			c.log.Debug().Err(reqErr).Msg("Inspector: not valid HTTP request, falling back to raw proxy")
			done := make(chan struct{}, 2)
			go func() {
				bp := proxyBufPool.Get().(*[]byte)
				_, _ = io.CopyBuffer(&countingWriter{w: local, count: &tunnel.BytesReceived}, reqBuf, *bp)
				proxyBufPool.Put(bp)
				done <- struct{}{}
			}()
			go func() {
				bp := proxyBufPool.Get().(*[]byte)
				_, _ = io.CopyBuffer(&countingWriter{w: stream, count: &tunnel.BytesSent}, local, *bp)
				proxyBufPool.Put(bp)
				done <- struct{}{}
			}()
			<-done
			_ = local.Close()
			_ = stream.Close()
			<-done
			return
		}

		// WebSocket/upgrade requests cannot be inspected — the connection must stay
		// open for bidirectional framing after the 101 handshake. Forward the request
		// and fall back to raw bidirectional copy.
		if isHTTPUpgrade(httpReq) {
			c.log.Debug().
				Str("upgrade", httpReq.Header.Get("Upgrade")).
				Str("path", httpReq.URL.Path).
				Msg("Inspector: upgrade request, falling back to raw proxy")
			if writeErr := httpReq.Write(local); writeErr != nil {
				c.log.Debug().Err(writeErr).Msg("Inspector: failed to forward upgrade request")
				return
			}
			done := make(chan struct{}, 2)
			go func() {
				bp := proxyBufPool.Get().(*[]byte)
				_, _ = io.CopyBuffer(&countingWriter{w: local, count: &tunnel.BytesReceived}, reqBuf, *bp)
				proxyBufPool.Put(bp)
				done <- struct{}{}
			}()
			go func() {
				bp := proxyBufPool.Get().(*[]byte)
				_, _ = io.CopyBuffer(&countingWriter{w: stream, count: &tunnel.BytesSent}, local, *bp)
				proxyBufPool.Put(bp)
				done <- struct{}{}
			}()
			<-done
			_ = local.Close()
			_ = stream.Close()
			<-done
			return
		}

		// Capture request metadata and body.
		cap.CaptureRequest(httpReq)

		// Forward the request to the local service.
		if writeErr := httpReq.Write(local); writeErr != nil {
			c.log.Debug().Err(writeErr).Msg("Inspector: failed to forward request to local")
			return
		}
		tunnel.BytesReceived.Add(httpReq.ContentLength)

		// Read the HTTP response from local service (respects Content-Length/chunked).
		localBuf := bufio.NewReader(local)
		resp, respErr := http.ReadResponse(localBuf, httpReq)
		if respErr != nil {
			c.log.Debug().Err(respErr).Msg("Inspector: failed to read HTTP response from local")
			return
		}

		// Capture response BEFORE Write (reads body, replaces with buffer).
		cap.CaptureResponse(resp)

		// Write the HTTP response back to the stream (server).
		if writeErr := resp.Write(stream); writeErr != nil {
			c.log.Debug().Err(writeErr).Msg("Inspector: failed to write response to stream")
		}
		resp.Body.Close()
		tunnel.BytesSent.Add(resp.ContentLength)

		// Finalize and store exchange.
		ex, err := cap.Finalize()
		if err != nil {
			c.log.Error().Err(err).Msg("Capture finalize failed")
		} else {
			c.log.Debug().Str("method", ex.Method).Str("path", ex.Path).Int("status", ex.StatusCode).Msg("Exchange captured")
			c.inspector.AddExchange(ex)
		}
	} else {
		done := make(chan struct{}, 2)
		download := &countingWriter{w: local, count: &tunnel.BytesReceived}
		upload := &countingWriter{w: stream, count: &tunnel.BytesSent}

		go func() {
			bp := proxyBufPool.Get().(*[]byte)
			_, _ = io.CopyBuffer(download, streamReader, *bp) // download: stream → local
			proxyBufPool.Put(bp)
			done <- struct{}{}
		}()

		go func() {
			bp := proxyBufPool.Get().(*[]byte)
			_, _ = io.CopyBuffer(upload, local, *bp) // upload: local → stream
			proxyBufPool.Put(bp)
			done <- struct{}{}
		}()

		<-done
		_ = local.Close()
		_ = stream.Close()
		<-done
	}

	if httpMethod != "" {
		elapsed := time.Since(reqStart).Milliseconds()
		var methodColor string
		switch httpMethod {
		case "GET":
			methodColor = "\033[32m" // green
		case "POST":
			methodColor = "\033[33m" // yellow
		case "PUT":
			methodColor = "\033[34m" // blue
		case "PATCH":
			methodColor = "\033[35m" // magenta
		case "DELETE":
			methodColor = "\033[31m" // red
		case "OPTIONS":
			methodColor = "\033[36m" // cyan
		default:
			methodColor = "\033[90m" // gray
		}
		fmt.Printf("  %s%s\033[0m %s \033[90m%dms\033[0m\n", methodColor, httpMethod, httpPath, elapsed)
	}
}

func (c *Client) keepalive() {
	defer c.wg.Done()

	// Initialize lastPong to now so we don't immediately timeout
	c.lastPong.Store(time.Now().UnixNano())

	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()

	consecutivePingFailures := 0
	const maxPingFailures = 3
	const pongTimeout = 3 * keepaliveInterval // 90s at default 30s interval

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// Check pong timeout
			if time.Since(time.Unix(0, c.lastPong.Load())) > pongTimeout {
				c.log.Warn().Msg("Pong timeout, server appears unresponsive")
				c.handleDisconnect()
				return
			}

			ping := &protocol.PingMessage{
				Message: protocol.NewMessage(protocol.MsgPing),
			}
			if err := c.sendControl(ping); err != nil {
				consecutivePingFailures++
				c.log.Warn().Err(err).Int("consecutive_failures", consecutivePingFailures).Msg("Failed to send ping")
				if consecutivePingFailures >= maxPingFailures {
					c.log.Warn().Msg("Too many consecutive ping failures, disconnecting")
					c.handleDisconnect()
					return
				}
			} else {
				consecutivePingFailures = 0
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

// backoffWithJitter returns the duration with ±20% jitter applied.
func backoffWithJitter(d time.Duration) time.Duration {
	// jitter ±20%: multiply by 0.8..1.2
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	jitter := 0.8 + float64(b[0])/255.0*0.4 // [0.8, 1.2]
	return time.Duration(float64(d) * jitter)
}

func (c *Client) reconnect() {
	attempts := 0
	baseInterval := c.cfg.Reconnect.Interval
	if baseInterval == 0 {
		baseInterval = defaultReconnectInterval
	}
	maxBackoff := maxReconnectBackoff
	currentBackoff := baseInterval

	for {
		if c.closed.Load() {
			return
		}

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

		c.log.Info().Int("attempt", attempts).Dur("backoff", currentBackoff).Msg("Attempting to reconnect...")
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

		// Close data sessions
		c.dataSessionMu.Lock()
		for _, ds := range c.dataSessions {
			ds.Close()
		}
		for _, dc := range c.dataConns {
			dc.Close()
		}
		c.dataSessions = nil
		c.dataConns = nil
		c.dataSessionMu.Unlock()

		// Clear tunnels and stop timers
		c.tunnelsMu.Lock()
		c.tunnels = make(map[string]*ActiveTunnel)
		c.tunnelsMu.Unlock()

		c.stopAllTimers()

		if c.inspector != nil {
			_ = c.inspector.Stop()
			c.inspector = nil
			c.inspectMgr = nil
		}

		// Cancel old context and wait for goroutines to finish
		c.cancel()
		c.wg.Wait()
		c.ctx, c.cancel = context.WithCancel(context.Background())

		if c.closed.Load() {
			c.cancel()
			return
		}

		// Try to connect
		if err := c.Connect(); err != nil {
			// Check if the error is due to an expired token (directly or wrapped)
			var authErr *AuthError
			if errors.As(err, &authErr) && authErr.IsTokenExpired() {
				c.log.Warn().Msg("Token expired, attempting refresh...")

				c.tokenMu.RLock()
				refresher := c.tokenRefresher
				c.tokenMu.RUnlock()

				if refresher != nil {
					newToken, refreshErr := refresher(c.cfg.Server.Address)
					if refreshErr != nil {
						c.log.Error().Err(refreshErr).Msg("Failed to refresh token")
						time.Sleep(backoffWithJitter(currentBackoff))
						// Don't reset backoff after token refresh — server may still be unavailable
						currentBackoff *= 2
						if currentBackoff > maxBackoff {
							currentBackoff = maxBackoff
						}
						continue
					}

					c.UpdateToken(newToken)
					c.log.Info().Msg("Token refreshed successfully, retrying connection...")
					// Don't sleep, try immediately with new token
					continue
				} else {
					c.log.Error().Msg("Token expired but no token refresher configured")
					c.Close()
					return
				}
			}

			c.log.Error().Err(err).Msg("Reconnection failed")
			time.Sleep(backoffWithJitter(currentBackoff))
			currentBackoff *= 2
			if currentBackoff > maxBackoff {
				currentBackoff = maxBackoff
			}
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

// CloseTunnel closes a specific tunnel by ID.
// It sends a close request to the server; the server will respond with
// TunnelClosed which triggers handleTunnelClosed for final cleanup.
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

// closeTunnel closes a single tunnel: sends a close request to the server,
// removes the tunnel from local state, and stops its timers.
// Used by auto-close and max-lifetime timer callbacks to close only
// the specific tunnel instead of the entire client.
func (c *Client) closeTunnel(tunnelID string) {
	// Send close request to server
	msg := &protocol.TunnelCloseMessage{
		Message:  protocol.NewMessage(protocol.MsgTunnelClose),
		TunnelID: tunnelID,
	}
	if err := c.sendControl(msg); err != nil {
		c.log.Error().Err(err).Str("tunnel_id", tunnelID).Msg("Failed to send tunnel close")
	}

	// Remove from local state
	c.tunnelsMu.Lock()
	delete(c.tunnels, tunnelID)
	c.tunnelsMu.Unlock()

	// Stop timers for this tunnel
	c.stopTunnelTimers(tunnelID)
}

// stopTunnelTimers stops and removes auto-close and max-lifetime timers
// for a specific tunnel. Safe to call even if no timers exist for the tunnel.
func (c *Client) stopTunnelTimers(tunnelID string) {
	c.timersMu.Lock()
	defer c.timersMu.Unlock()

	if t, ok := c.autoCloseTimers[tunnelID]; ok {
		t.stop()
		delete(c.autoCloseTimers, tunnelID)
	}
	if t, ok := c.maxLifetimeTimers[tunnelID]; ok {
		t.stop()
		delete(c.maxLifetimeTimers, tunnelID)
	}
}

// Wait waits for the client to close
func (c *Client) Wait() {
	c.wg.Wait()
}

// InspectorAddr returns the actual inspector address, or empty if not running.
func (c *Client) InspectorAddr() string {
	if c.inspector != nil {
		return c.inspector.Addr()
	}
	return ""
}

// Close closes the client. It is safe to call multiple times.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		c.cancel()

		// Stop all auto-close and max-lifetime timers
		c.stopAllTimers()

		if c.inspector != nil {
			_ = c.inspector.Stop()
		}

		if c.controlStream != nil {
			c.controlStream.Close()
		}
		if c.session != nil {
			c.session.Close()
		}
		if c.conn != nil {
			c.conn.Close()
		}

		// Close all data sessions
		c.dataSessionMu.Lock()
		for _, ds := range c.dataSessions {
			ds.Close()
		}
		for _, dc := range c.dataConns {
			dc.Close()
		}
		c.dataSessions = nil
		c.dataConns = nil
		c.dataSessionMu.Unlock()

		// Wait for goroutines to finish with timeout
		done := make(chan struct{})
		go func() { c.wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			c.log.Warn().Msg("Close: goroutine drain timeout")
		}

		c.log.Info().Msg("Client closed")
	})
}

// stopAllTimers stops and clears all auto-close and max-lifetime timers.
func (c *Client) stopAllTimers() {
	c.timersMu.Lock()
	defer c.timersMu.Unlock()

	for id, t := range c.autoCloseTimers {
		t.stop()
		delete(c.autoCloseTimers, id)
	}
	for id, t := range c.maxLifetimeTimers {
		t.stop()
		delete(c.maxLifetimeTimers, id)
	}
}

func (c *Client) emitTrafficStats(tunnel *ActiveTunnel) {
	ticker := time.NewTicker(trafficStatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// Check if tunnel still exists
			c.tunnelsMu.RLock()
			_, exists := c.tunnels[tunnel.ID]
			c.tunnelsMu.RUnlock()
			if !exists {
				return
			}

			c.events.EmitWithPayload(EventTrafficUpdate, map[string]interface{}{
				"tunnel_id":      tunnel.ID,
				"bytes_sent":     tunnel.BytesSent.Load(),
				"bytes_received": tunnel.BytesReceived.Load(),
			})
		}
	}
}

func (c *Client) openDataConnections() {
	var wg sync.WaitGroup
	var failCount atomic.Int32
	for i := 0; i < c.maxDataSessions; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := c.openDataConnection(idx); err != nil {
				failCount.Add(1)
				c.log.Debug().Err(err).Int("index", idx).Msg("Data connection failed")
			}
		}(i)
	}
	wg.Wait()

	failed := int(failCount.Load())
	opened := c.maxDataSessions - failed
	if failed > 0 && opened > 0 {
		c.log.Info().Int("opened", opened).Int("failed", failed).Int("requested", c.maxDataSessions).
			Msg("Some data connections could not be established (performance may be reduced)")
	} else if opened == 0 {
		c.log.Warn().Int("requested", c.maxDataSessions).
			Msg("No data connections established, using primary connection only")
	}
}

func (c *Client) openDataConnection(idx int) error {
	backoff := []time.Duration{100 * time.Millisecond, 300 * time.Millisecond, 1 * time.Second}
	var lastErr error
	for attempt := 0; attempt <= len(backoff); attempt++ {
		if attempt > 0 {
			c.log.Debug().Err(lastErr).Int("index", idx).Int("attempt", attempt).Msg("Data connection attempt failed, retrying")
			select {
			case <-c.ctx.Done():
				return c.ctx.Err()
			case <-time.After(backoff[attempt-1]):
			}
		}
		lastErr = c.tryOpenDataConnection(idx)
		if lastErr == nil {
			return nil
		}
		// Only retry on "join session rejected" errors (race condition on server side)
		if !strings.Contains(lastErr.Error(), "join session rejected") {
			return lastErr
		}
	}
	return lastErr
}

func (c *Client) tryOpenDataConnection(idx int) error {
	// Dial server
	conn, err := c.dialServer()
	if err != nil {
		return fmt.Errorf("dial server: %w", err)
	}

	// Negotiate compression
	rwc, _, err := protocol.NegotiateCompression(conn, c.cfg.Server.Compression, false)
	if err != nil {
		conn.Close()
		return fmt.Errorf("compression negotiation: %w", err)
	}

	// Create yamux session (client mode)
	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = true
	yamuxCfg.KeepAliveInterval = yamuxKeepAliveInterval
	yamuxCfg.MaxStreamWindowSize = yamuxMaxStreamWindowSize
	yamuxCfg.ConnectionWriteTimeout = yamuxConnectionWriteTimeout
	yamuxCfg.LogOutput = io.Discard
	session, err := yamux.Client(rwc, yamuxCfg)
	if err != nil {
		conn.Close()
		return fmt.Errorf("create yamux session: %w", err)
	}

	// Open control stream to send JoinSession
	stream, err := session.Open()
	if err != nil {
		session.Close()
		conn.Close()
		return fmt.Errorf("open stream: %w", err)
	}

	codec := protocol.NewCodec(stream, stream)

	// Send join session message
	joinMsg := &protocol.JoinSessionMessage{
		Message:  protocol.NewMessage(protocol.MsgJoinSession),
		ClientID: c.clientID,
		Secret:   c.sessionSecret,
	}
	if err := codec.Encode(joinMsg); err != nil {
		stream.Close()
		session.Close()
		conn.Close()
		return fmt.Errorf("send join_session: %w", err)
	}

	// Read result
	_ = stream.SetReadDeadline(time.Now().Add(authResponseTimeout))
	var result protocol.JoinSessionResult
	if err := codec.Decode(&result); err != nil {
		stream.Close()
		session.Close()
		conn.Close()
		return fmt.Errorf("read join_session result: %w", err)
	}
	_ = stream.SetReadDeadline(time.Time{})

	if !result.Success {
		stream.Close()
		session.Close()
		conn.Close()
		return fmt.Errorf("join session rejected: %s", result.Error)
	}

	// Close the handshake stream — server will Open() streams on this session
	stream.Close()

	// Store data session
	c.dataSessionMu.Lock()
	c.dataSessions = append(c.dataSessions, session)
	c.dataConns = append(c.dataConns, conn)
	c.dataSessionMu.Unlock()

	// Accept streams on this data session (Task 7)
	c.wg.Add(1)
	go c.acceptDataStreams(session)

	c.log.Info().Int("index", idx).Msg("Data connection established")
	return nil
}

// isHTTPUpgrade reports whether the request is a WebSocket or other HTTP upgrade.
func isHTTPUpgrade(req *http.Request) bool {
	return strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade") &&
		req.Header.Get("Upgrade") != ""
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
