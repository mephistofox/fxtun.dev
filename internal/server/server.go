package server

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/database"
	"github.com/mephistofox/fxtunnel/internal/inspect"
	"github.com/mephistofox/fxtunnel/internal/protocol"
	fxtls "github.com/mephistofox/fxtunnel/internal/tls"
)

// Server is the main tunnel server
type Server struct {
	cfg    *config.ServerConfig
	log    zerolog.Logger

	// Listeners
	controlListener net.Listener
	httpListener    net.Listener

	// Client manager
	clientMgr *ClientManager

	// Tunnel managers
	httpRouter  *HTTPRouter
	httpServer  *http.Server
	tcpManager  *TCPManager
	udpManager  *UDPManager
	inspectMgr  *inspect.Manager

	// Database integration
	db          *database.Database
	authService *auth.Service

	// Custom domains
	certManager    *fxtls.CertManager
	customDomains  map[string]*database.CustomDomain // domain -> entry
	customDomainMu sync.RWMutex

	// Active connections tracking for graceful drain
	activeConns sync.WaitGroup

	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Client represents a connected client
type Client struct {
	ID           string
	RemoteAddr   string
	Token        *config.TokenConfig
	Session      *yamux.Session
	ControlCodec *protocol.Codec
	ControlConn  net.Conn
	Tunnels      map[string]*Tunnel
	TunnelsMu    sync.RWMutex
	Connected    time.Time
	lastPing     atomic.Int64

	// Database integration
	UserID     int64              // 0 if legacy token
	APITokenID int64              // 0 if legacy token
	DBToken    *database.APIToken // nil if legacy token
	IsAdmin    bool               // true if user is admin

	server    *Server
	conn      net.Conn
	log       zerolog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex // for writing to control stream
	closeOnce sync.Once

	// Stream pool: pre-opened yamux streams for low-latency connection handling
	streamPool chan net.Conn
}

// Tunnel represents an active tunnel
type Tunnel struct {
	ID         string
	ClientID   string
	Type       protocol.TunnelType
	Name       string
	Subdomain  string // For HTTP
	RemotePort int    // For TCP/UDP
	LocalPort  int
	Created    time.Time

	// For TCP/UDP
	listener net.Listener
	udpConn  *net.UDPConn
}

// New creates a new server
func New(cfg *config.ServerConfig, log zerolog.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		cfg:           cfg,
		log:           log.With().Str("component", "server").Logger(),
		clientMgr:     NewClientManager(log.With().Str("component", "server").Logger()),
		customDomains: make(map[string]*database.CustomDomain),
		ctx:           ctx,
		cancel:        cancel,
	}

	s.httpRouter = NewHTTPRouter(s, log)
	s.tcpManager = NewTCPManager(s, log)
	s.udpManager = NewUDPManager(s, log)

	capacity := 0
	if cfg.Inspect.Enabled {
		capacity = cfg.Inspect.MaxEntries
		if capacity == 0 {
			capacity = 1000
		}
	}
	maxBody := cfg.Inspect.MaxBodySize
	if maxBody == 0 {
		maxBody = inspect.MaxBodySize
	}
	s.inspectMgr = inspect.NewManager(capacity, maxBody)

	return s
}

// SetDatabase sets the database for the server
func (s *Server) SetDatabase(db *database.Database) {
	s.db = db
}

// SetAuthService sets the auth service for JWT validation
func (s *Server) SetAuthService(authService *auth.Service) {
	s.authService = authService
}

// GetDatabase returns the database
func (s *Server) GetDatabase() *database.Database {
	return s.db
}

// GetConfig returns the server configuration
func (s *Server) GetConfig() *config.ServerConfig {
	return s.cfg
}

// InspectManager returns the inspect manager
func (s *Server) InspectManager() *inspect.Manager {
	return s.inspectMgr
}

// CertManager returns the TLS certificate manager (may be nil).
func (s *Server) CertManager() *fxtls.CertManager {
	return s.certManager
}

// LoadCustomDomains loads verified custom domains from DB into memory.
func (s *Server) LoadCustomDomains() error {
	if s.db == nil {
		return nil
	}
	domains, err := s.db.CustomDomains.GetAllVerified()
	if err != nil {
		return err
	}
	s.customDomainMu.Lock()
	defer s.customDomainMu.Unlock()
	for _, d := range domains {
		s.customDomains[strings.ToLower(d.Domain)] = d
	}
	s.log.Info().Int("count", len(domains)).Msg("Loaded custom domains")
	return nil
}

// LookupCustomDomain looks up a custom domain by host.
func (s *Server) LookupCustomDomain(host string) *database.CustomDomain {
	host = strings.ToLower(host)
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	s.customDomainMu.RLock()
	defer s.customDomainMu.RUnlock()
	return s.customDomains[host]
}

// AddCustomDomain adds a custom domain to the in-memory cache.
func (s *Server) AddCustomDomain(d *database.CustomDomain) {
	s.customDomainMu.Lock()
	defer s.customDomainMu.Unlock()
	s.customDomains[strings.ToLower(d.Domain)] = d
}

// RemoveCustomDomain removes a custom domain from the in-memory cache.
func (s *Server) RemoveCustomDomain(domain string) {
	s.customDomainMu.Lock()
	defer s.customDomainMu.Unlock()
	delete(s.customDomains, strings.ToLower(domain))
}

// InitCustomDomains initializes custom domains and TLS cert manager.
func (s *Server) InitCustomDomains() error {
	if s.db == nil || !s.cfg.CustomDomains.Enabled {
		return nil
	}

	if err := s.LoadCustomDomains(); err != nil {
		return fmt.Errorf("load custom domains: %w", err)
	}

	s.certManager = fxtls.NewCertManager(s.cfg.TLS, s.db, s.log)
	if err := s.certManager.LoadFromDB(); err != nil {
		s.log.Warn().Err(err).Msg("Failed to load TLS certs from DB")
	}
	s.certManager.StartRenewal()

	return nil
}

// Start starts the server
func (s *Server) Start() error {
	// Start control plane listener
	controlAddr := fmt.Sprintf(":%d", s.cfg.Server.ControlPort)
	var err error

	if s.cfg.TLS.Enabled {
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(s.cfg.TLS.CertFile, s.cfg.TLS.KeyFile)
		if err != nil {
			return fmt.Errorf("load TLS certificate: %w", err)
		}
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
		s.controlListener, err = tls.Listen("tcp", controlAddr, tlsCfg)
	} else {
		s.controlListener, err = newReusePortListener(s.ctx, controlAddr)
	}
	if err != nil {
		return fmt.Errorf("listen control: %w", err)
	}
	s.log.Info().Str("addr", controlAddr).Msg("Control plane listening")

	// Start HTTP listener
	httpAddr := fmt.Sprintf(":%d", s.cfg.Server.HTTPPort)
	s.httpListener, err = newReusePortListener(s.ctx, httpAddr)
	if err != nil {
		s.controlListener.Close()
		return fmt.Errorf("listen http: %w", err)
	}
	s.log.Info().Str("addr", httpAddr).Msg("HTTP listener started")

	// Accept control connections
	s.wg.Add(1)
	go s.acceptControlConnections()

	// Start HTTP server with keep-alive support
	s.httpServer = &http.Server{
		Handler: s.httpRouter,
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.httpServer.Serve(s.httpListener); err != nil && err != http.ErrServerClosed {
			s.log.Error().Err(err).Msg("HTTP server error")
		}
	}()

	return nil
}

// Stop stops the server gracefully
func (s *Server) Stop() error {
	s.log.Info().Msg("Shutting down server...")

	// Phase 1: stop accepting new connections
	if s.controlListener != nil {
		s.controlListener.Close()
	}
	if s.httpListener != nil {
		s.httpListener.Close()
	}

	// Phase 2: drain in-flight connections (max 10s)
	s.log.Info().Msg("Draining active connections...")
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer drainCancel()

	// Gracefully shutdown HTTP server (drains keep-alive connections)
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(drainCtx); err != nil {
			s.log.Warn().Err(err).Msg("HTTP server shutdown error")
		}
	}

	drainDone := make(chan struct{})
	go func() {
		s.activeConns.Wait()
		close(drainDone)
	}()
	select {
	case <-drainDone:
		s.log.Info().Msg("All connections drained")
	case <-drainCtx.Done():
		s.log.Warn().Msg("Drain timeout, forcing shutdown")
	}

	// Phase 3: cancel context and close all clients
	s.cancel()

	for _, c := range s.clientMgr.allClients() {
		c.Close()
	}

	// Stop managers
	s.tcpManager.Stop()
	s.udpManager.Stop()

	s.inspectMgr.Close()

	if s.certManager != nil {
		s.certManager.Stop()
	}

	s.wg.Wait()
	s.log.Info().Msg("Server stopped")
	return nil
}

func (s *Server) acceptControlConnections() {
	defer s.wg.Done()

	for {
		conn, err := s.controlListener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				s.log.Error().Err(err).Msg("Accept control connection failed")
				continue
			}
		}

		s.wg.Add(1)
		go s.handleControlConnection(conn)
	}
}


func (s *Server) handleControlConnection(conn net.Conn) {
	defer s.wg.Done()

	tuneTCPConn(conn)

	remoteAddr := conn.RemoteAddr().String()
	log := s.log.With().Str("remote", remoteAddr).Logger()
	log.Debug().Msg("New control connection")

	// Negotiate compression before yamux
	rwc, compressed, err := protocol.NegotiateCompression(conn, s.cfg.Server.CompressionEnabled, true)
	if err != nil {
		log.Error().Err(err).Msg("Compression negotiation failed")
		conn.Close()
		return
	}
	if compressed {
		log.Debug().Msg("Compression enabled (zstd)")
	}

	// Create yamux session FIRST (server mode) with optimized config
	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = true
	yamuxCfg.KeepAliveInterval = 10 * time.Second
	yamuxCfg.MaxStreamWindowSize = 4 * 1024 * 1024 // 4MB window for high throughput
	yamuxCfg.ConnectionWriteTimeout = 30 * time.Second
	session, err := yamux.Server(rwc, yamuxCfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create yamux session")
		conn.Close()
		return
	}

	// Accept the control stream (first stream from client)
	controlStream, err := session.Accept()
	if err != nil {
		log.Error().Err(err).Msg("Failed to accept control stream")
		session.Close()
		return
	}

	// Create codec for the control stream
	codec := protocol.NewCodec(controlStream, controlStream)

	// Wait for authentication with timeout
	controlStream.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read auth message
	data, baseMsg, err := codec.DecodeRaw()
	if err != nil {
		log.Error().Err(err).Msg("Failed to read auth message")
		session.Close()
		return
	}

	if baseMsg.Type != protocol.MsgAuth {
		log.Error().Str("type", string(baseMsg.Type)).Msg("Expected auth message")
		s.sendError(codec, protocol.ErrCodeProtocolError, "expected auth message", true)
		session.Close()
		return
	}

	parsed, err := protocol.ParseMessage(data, protocol.MsgAuth)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse auth message")
		session.Close()
		return
	}

	authMsg := parsed.(*protocol.AuthMessage)
	controlStream.SetReadDeadline(time.Time{}) // Clear deadline

	// Authenticate
	client, err := s.authenticate(conn, session, controlStream, codec, authMsg, log)
	if err != nil {
		log.Warn().Err(err).Msg("Authentication failed")
		session.Close()
		return
	}

	log = log.With().Str("client_id", client.ID).Logger()
	log.Info().Msg("Client authenticated")

	// Handle client messages
	client.handle()
}

func (s *Server) removeClient(clientID string) {
	s.clientMgr.removeClient(clientID)
}

func (s *Server) sendError(codec *protocol.Codec, code, message string, fatal bool) {
	msg := &protocol.ErrorMessage{
		Message: protocol.NewMessage(protocol.MsgError),
		Error:   message,
		Code:    code,
		Fatal:   fatal,
	}
	codec.Encode(msg)
}

func (s *Server) GetClient(clientID string) *Client {
	return s.clientMgr.GetClient(clientID)
}

// Client methods

func (c *Client) handle() {
	defer c.Close()

	// Pre-open yamux streams for low-latency connection handling
	c.startStreamPool()

	// Start keepalive
	go c.keepalive()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		data, baseMsg, err := c.ControlCodec.DecodeRaw()
		if err != nil {
			c.log.Debug().Err(err).Msg("Read error, closing client")
			return
		}

		c.lastPing.Store(time.Now().UnixNano())

		switch baseMsg.Type {
		case protocol.MsgTunnelRequest:
			c.handleTunnelRequest(data)
		case protocol.MsgTunnelClose:
			c.handleTunnelClose(data)
		case protocol.MsgConnectionAccept:
			c.handleConnectionAccept(data)
		case protocol.MsgPing:
			c.handlePing()
		case protocol.MsgPong:
			// Keepalive response, just update LastPing (already done above)
		default:
			c.log.Warn().Str("type", string(baseMsg.Type)).Msg("Unknown message type")
		}
	}
}

func (c *Client) handleTunnelRequest(data []byte) {
	parsed, err := protocol.ParseMessage(data, protocol.MsgTunnelRequest)
	if err != nil {
		c.log.Error().Err(err).Msg("Failed to parse tunnel request")
		return
	}
	req := parsed.(*protocol.TunnelRequestMessage)

	// Check tunnel limit
	maxTunnels := 10
	if c.Token != nil && c.Token.MaxTunnels > 0 {
		maxTunnels = c.Token.MaxTunnels
	}
	if c.DBToken != nil && c.DBToken.MaxTunnels > 0 {
		maxTunnels = c.DBToken.MaxTunnels
	}

	c.TunnelsMu.RLock()
	tunnelCount := len(c.Tunnels)
	c.TunnelsMu.RUnlock()

	if tunnelCount >= maxTunnels {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodeTunnelLimit, "tunnel limit reached")
		return
	}

	switch req.TunnelType {
	case protocol.TunnelHTTP:
		c.createHTTPTunnel(req)
	case protocol.TunnelTCP:
		c.createTCPTunnel(req)
	case protocol.TunnelUDP:
		c.createUDPTunnel(req)
	default:
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodeProtocolError, "unknown tunnel type")
	}
}

func (c *Client) createHTTPTunnel(req *protocol.TunnelRequestMessage) {
	subdomain := req.Subdomain
	subdomain = strings.ToLower(subdomain)
	if subdomain == "" {
		subdomain = generateShortID()
	}

	// Check subdomain permission
	if c.Token != nil && !c.Token.CanUseSubdomain(subdomain) {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodePermissionDenied, "subdomain not allowed")
		return
	}

	// Register with HTTP router
	tunnelID := generateID()
	tunnel := &Tunnel{
		ID:        tunnelID,
		ClientID:  c.ID,
		Type:      protocol.TunnelHTTP,
		Name:      req.Name,
		Subdomain: subdomain,
		LocalPort: req.LocalPort,
		Created:   time.Now(),
	}

	if err := c.server.httpRouter.RegisterTunnel(subdomain, tunnel); err != nil {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodeSubdomainTaken, err.Error())
		return
	}

	c.server.inspectMgr.GetOrCreate(tunnelID)

	c.TunnelsMu.Lock()
	c.Tunnels[tunnelID] = tunnel
	c.TunnelsMu.Unlock()

	url := fmt.Sprintf("http://%s.%s", subdomain, c.server.cfg.Domain.Base)

	resp := &protocol.TunnelCreatedMessage{
		Message:    protocol.NewMessage(protocol.MsgTunnelCreated),
		TunnelID:   tunnelID,
		TunnelType: protocol.TunnelHTTP,
		Name:       req.Name,
		URL:        url,
		Subdomain:  subdomain,
	}
	resp.RequestID = req.RequestID

	c.sendControl(resp)
	c.log.Info().Str("tunnel_id", tunnelID).Str("url", url).Msg("HTTP tunnel created")
}

func (c *Client) createTCPTunnel(req *protocol.TunnelRequestMessage) {
	port, listener, err := c.server.tcpManager.AllocatePort(req.RemotePort)
	if err != nil {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodePortUnavailable, err.Error())
		return
	}

	tunnelID := generateID()
	tunnel := &Tunnel{
		ID:         tunnelID,
		ClientID:   c.ID,
		Type:       protocol.TunnelTCP,
		Name:       req.Name,
		RemotePort: port,
		LocalPort:  req.LocalPort,
		Created:    time.Now(),
		listener:   listener,
	}

	c.TunnelsMu.Lock()
	c.Tunnels[tunnelID] = tunnel
	c.TunnelsMu.Unlock()

	// Start accepting connections
	go c.server.tcpManager.AcceptConnections(tunnel, c)

	remoteAddr := fmt.Sprintf("%s:%d", c.server.cfg.Domain.Base, port)

	resp := &protocol.TunnelCreatedMessage{
		Message:    protocol.NewMessage(protocol.MsgTunnelCreated),
		TunnelID:   tunnelID,
		TunnelType: protocol.TunnelTCP,
		Name:       req.Name,
		RemotePort: port,
		RemoteAddr: remoteAddr,
	}
	resp.RequestID = req.RequestID

	c.sendControl(resp)
	c.log.Info().Str("tunnel_id", tunnelID).Int("port", port).Msg("TCP tunnel created")
}

func (c *Client) createUDPTunnel(req *protocol.TunnelRequestMessage) {
	port, udpConn, err := c.server.udpManager.AllocatePort(req.RemotePort)
	if err != nil {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodePortUnavailable, err.Error())
		return
	}

	tunnelID := generateID()
	tunnel := &Tunnel{
		ID:         tunnelID,
		ClientID:   c.ID,
		Type:       protocol.TunnelUDP,
		Name:       req.Name,
		RemotePort: port,
		LocalPort:  req.LocalPort,
		Created:    time.Now(),
		udpConn:    udpConn,
	}

	c.TunnelsMu.Lock()
	c.Tunnels[tunnelID] = tunnel
	c.TunnelsMu.Unlock()

	// Start handling UDP packets
	go c.server.udpManager.HandlePackets(tunnel, c)

	remoteAddr := fmt.Sprintf("%s:%d", c.server.cfg.Domain.Base, port)

	resp := &protocol.TunnelCreatedMessage{
		Message:    protocol.NewMessage(protocol.MsgTunnelCreated),
		TunnelID:   tunnelID,
		TunnelType: protocol.TunnelUDP,
		Name:       req.Name,
		RemotePort: port,
		RemoteAddr: remoteAddr,
	}
	resp.RequestID = req.RequestID

	c.sendControl(resp)
	c.log.Info().Str("tunnel_id", tunnelID).Int("port", port).Msg("UDP tunnel created")
}

func (c *Client) handleTunnelClose(data []byte) {
	parsed, err := protocol.ParseMessage(data, protocol.MsgTunnelClose)
	if err != nil {
		c.log.Error().Err(err).Msg("Failed to parse tunnel close")
		return
	}
	closeMsg := parsed.(*protocol.TunnelCloseMessage)

	c.closeTunnel(closeMsg.TunnelID)
}

func (c *Client) closeTunnel(tunnelID string) {
	c.TunnelsMu.Lock()
	tunnel, exists := c.Tunnels[tunnelID]
	if exists {
		delete(c.Tunnels, tunnelID)
	}
	c.TunnelsMu.Unlock()

	if !exists {
		return
	}

	switch tunnel.Type {
	case protocol.TunnelHTTP:
		c.server.httpRouter.UnregisterTunnel(tunnel.Subdomain)
		c.server.inspectMgr.Remove(tunnelID)
	case protocol.TunnelTCP:
		if tunnel.listener != nil {
			tunnel.listener.Close()
		}
	case protocol.TunnelUDP:
		if tunnel.udpConn != nil {
			tunnel.udpConn.Close()
		}
	}

	resp := &protocol.TunnelClosedMessage{
		Message:  protocol.NewMessage(protocol.MsgTunnelClosed),
		TunnelID: tunnelID,
	}
	c.sendControl(resp)

	c.log.Info().Str("tunnel_id", tunnelID).Msg("Tunnel closed")
}

func (c *Client) handleConnectionAccept(data []byte) {
	// This is handled via yamux streams directly
}

func (c *Client) handlePing() {
	pong := &protocol.PongMessage{
		Message: protocol.NewMessage(protocol.MsgPong),
	}
	c.sendControl(pong)
}

func (c *Client) keepalive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if time.Since(time.Unix(0, c.lastPing.Load())) > 90*time.Second {
				c.log.Warn().Msg("Client timeout, closing")
				c.Close()
				return
			}
		}
	}
}

func (c *Client) sendControl(msg any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ControlCodec.Encode(msg)
}

func (c *Client) sendTunnelError(requestID, tunnelID, code, message string) {
	msg := &protocol.TunnelErrorMessage{
		Message:  protocol.NewMessage(protocol.MsgTunnelError),
		TunnelID: tunnelID,
		Error:    message,
		Code:     code,
	}
	msg.RequestID = requestID
	c.sendControl(msg)
}

// Close closes the client connection
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.cancel()

		// Close all tunnels
		c.TunnelsMu.Lock()
		for tunnelID, tunnel := range c.Tunnels {
			switch tunnel.Type {
			case protocol.TunnelHTTP:
				c.server.httpRouter.UnregisterTunnel(tunnel.Subdomain)
				c.server.inspectMgr.Remove(tunnelID)
			case protocol.TunnelTCP:
				if tunnel.listener != nil {
					tunnel.listener.Close()
				}
			case protocol.TunnelUDP:
				if tunnel.udpConn != nil {
					tunnel.udpConn.Close()
				}
			}
			delete(c.Tunnels, tunnelID)
		}
		c.TunnelsMu.Unlock()

		if c.ControlConn != nil {
			c.ControlConn.Close()
		}
		if c.Session != nil {
			c.Session.Close()
		}
		if c.conn != nil {
			c.conn.Close()
		}

		// Unlink user from client
		c.server.clientMgr.unlinkUserClient(c.UserID, c.ID)

		c.server.removeClient(c.ID)
		c.log.Info().Msg("Client disconnected")
	})
}

// Helper functions

var connIDCounter atomic.Uint64

func generateID() string {
	id := connIDCounter.Add(1)
	return strconv.FormatUint(id, 36)
}

func generateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// API Integration methods

// TunnelInfo represents tunnel information for the API
type TunnelInfo struct {
	ID         string
	Type       string
	Name       string
	Subdomain  string
	RemotePort int
	LocalPort  int
	ClientID   string
	UserID     int64
	CreatedAt  time.Time
}

// Stats represents server statistics
type Stats struct {
	ActiveClients int
	ActiveTunnels int
	HTTPTunnels   int
	TCPTunnels    int
	UDPTunnels    int
}

// GetTunnelsByUserID returns all tunnels for a user
func (s *Server) GetTunnelsByUserID(userID int64) []TunnelInfo {
	return s.clientMgr.GetTunnelsByUserID(userID)
}

// GetAllTunnels returns all tunnels from all clients (for admin)
func (s *Server) GetAllTunnels() []TunnelInfo {
	return s.clientMgr.GetAllTunnels()
}

// AdminCloseTunnel closes any tunnel by ID (admin only, no user check)
func (s *Server) AdminCloseTunnel(tunnelID string) error {
	return s.clientMgr.AdminCloseTunnel(tunnelID)
}

// CloseTunnelByID closes a tunnel by ID for a specific user
func (s *Server) CloseTunnelByID(tunnelID string, userID int64) error {
	return s.clientMgr.CloseTunnelByID(tunnelID, userID)
}

// GetStats returns server statistics
func (s *Server) GetStats() Stats {
	return s.clientMgr.GetStats()
}

