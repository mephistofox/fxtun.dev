package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/database"
	"github.com/mephistofox/fxtunnel/internal/protocol"
)

// Server is the main tunnel server
type Server struct {
	cfg    *config.ServerConfig
	log    zerolog.Logger

	// Listeners
	controlListener net.Listener
	httpListener    net.Listener

	// Connected clients
	clients   map[string]*Client
	clientsMu sync.RWMutex

	// Tunnel managers
	httpRouter  *HTTPRouter
	tcpManager  *TCPManager
	udpManager  *UDPManager

	// Database integration
	db            *database.Database
	authService   *auth.Service
	userClients   map[int64][]string // userID -> clientIDs
	userClientsMu sync.RWMutex

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

	server *Server
	conn   net.Conn
	log    zerolog.Logger
	ctx    context.Context
	cancel context.CancelFunc
	mu        sync.Mutex // for writing to control stream
	closeOnce sync.Once
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
		cfg:         cfg,
		log:         log.With().Str("component", "server").Logger(),
		clients:     make(map[string]*Client),
		userClients: make(map[int64][]string),
		ctx:         ctx,
		cancel:      cancel,
	}

	s.httpRouter = NewHTTPRouter(s, log)
	s.tcpManager = NewTCPManager(s, log)
	s.udpManager = NewUDPManager(s, log)

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
		s.controlListener, err = net.Listen("tcp", controlAddr)
	}
	if err != nil {
		return fmt.Errorf("listen control: %w", err)
	}
	s.log.Info().Str("addr", controlAddr).Msg("Control plane listening")

	// Start HTTP listener
	httpAddr := fmt.Sprintf(":%d", s.cfg.Server.HTTPPort)
	s.httpListener, err = net.Listen("tcp", httpAddr)
	if err != nil {
		s.controlListener.Close()
		return fmt.Errorf("listen http: %w", err)
	}
	s.log.Info().Str("addr", httpAddr).Msg("HTTP listener started")

	// Accept control connections
	s.wg.Add(1)
	go s.acceptControlConnections()

	// Accept HTTP connections
	s.wg.Add(1)
	go s.acceptHTTPConnections()

	return nil
}

// Stop stops the server gracefully
func (s *Server) Stop() error {
	s.log.Info().Msg("Shutting down server...")
	s.cancel()

	if s.controlListener != nil {
		s.controlListener.Close()
	}
	if s.httpListener != nil {
		s.httpListener.Close()
	}

	// Close all clients
	s.clientsMu.Lock()
	clients := make([]*Client, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}
	s.clientsMu.Unlock()
	for _, c := range clients {
		c.Close()
	}

	// Stop managers
	s.tcpManager.Stop()
	s.udpManager.Stop()

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

func (s *Server) acceptHTTPConnections() {
	defer s.wg.Done()

	for {
		conn, err := s.httpListener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				s.log.Error().Err(err).Msg("Accept HTTP connection failed")
				continue
			}
		}

		go s.httpRouter.HandleConnection(conn)
	}
}

func (s *Server) handleControlConnection(conn net.Conn) {
	defer s.wg.Done()

	remoteAddr := conn.RemoteAddr().String()
	log := s.log.With().Str("remote", remoteAddr).Logger()
	log.Debug().Msg("New control connection")

	// Create yamux session FIRST (server mode) with optimized config
	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = true
	yamuxCfg.KeepAliveInterval = 10 * time.Second
	yamuxCfg.MaxStreamWindowSize = 1024 * 1024 // 1MB window for better throughput
	session, err := yamux.Server(conn, yamuxCfg)
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

func (s *Server) authenticate(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, authMsg *protocol.AuthMessage, log zerolog.Logger) (*Client, error) {
	// First, try to authenticate with database token (new system)
	if s.db != nil {
		tokenHash := hashToken(authMsg.Token)
		apiToken, err := s.db.Tokens.GetByTokenHash(tokenHash)
		if err == nil && apiToken != nil {
			// Check IP whitelist
			if !apiToken.IsIPAllowed(conn.RemoteAddr().String()) {
				result := &protocol.AuthResultMessage{
					Message: protocol.NewMessage(protocol.MsgAuthResult),
					Success: false,
					Error:   "IP not allowed",
					Code:    protocol.ErrCodePermissionDenied,
				}
				codec.Encode(result)
				return nil, fmt.Errorf("IP not allowed for token")
			}

			// Valid DB token found
			client := s.createClientFromDBToken(conn, session, controlStream, codec, apiToken, log)

			// Update last used
			s.db.Tokens.UpdateLastUsed(apiToken.ID)

			// Link user to client
			s.linkUserClient(apiToken.UserID, client.ID)

			// Send success
			result := &protocol.AuthResultMessage{
				Message:    protocol.NewMessage(protocol.MsgAuthResult),
				Success:    true,
				ClientID:   client.ID,
				MaxTunnels: apiToken.MaxTunnels,
				ServerName: s.cfg.Domain.Base,
				SessionID:  client.ID,
			}
			if err := codec.Encode(result); err != nil {
				client.Close()
				return nil, fmt.Errorf("send auth result: %w", err)
			}

			log.Info().Int64("user_id", apiToken.UserID).Str("token_name", apiToken.Name).Msg("Authenticated with DB token")
			return client, nil
		}
	}

	// Try JWT authentication (for GUI login with phone/password)
	if s.authService != nil && isJWT(authMsg.Token) {
		claims, err := s.authService.ValidateAccessToken(authMsg.Token)
		if err != nil {
			// Check if token is expired - don't fallback to legacy tokens
			if err == auth.ErrTokenExpired {
				result := &protocol.AuthResultMessage{
					Message: protocol.NewMessage(protocol.MsgAuthResult),
					Success: false,
					Error:   "token expired",
					Code:    protocol.ErrCodeTokenExpired,
				}
				codec.Encode(result)
				return nil, fmt.Errorf("token expired")
			}
			// Other JWT errors - continue to legacy token check
			log.Debug().Err(err).Msg("JWT validation failed, trying legacy tokens")
		} else if claims != nil {
			// Valid JWT - create client for user
			client := s.createClientFromJWT(conn, session, controlStream, codec, claims, log)

			// Link user to client
			s.linkUserClient(claims.UserID, client.ID)

			// Send success
			result := &protocol.AuthResultMessage{
				Message:    protocol.NewMessage(protocol.MsgAuthResult),
				Success:    true,
				ClientID:   client.ID,
				MaxTunnels: 10, // Default for JWT auth
				ServerName: s.cfg.Domain.Base,
				SessionID:  client.ID,
			}
			if err := codec.Encode(result); err != nil {
				client.Close()
				return nil, fmt.Errorf("send auth result: %w", err)
			}

			log.Info().Int64("user_id", claims.UserID).Str("phone", claims.Phone).Msg("Authenticated with JWT")
			return client, nil
		}
	}

	// Fallback: Check YAML config tokens (legacy system)
	if s.cfg.Auth.Enabled {
		tokenCfg := s.cfg.FindToken(authMsg.Token)
		if tokenCfg == nil {
			result := &protocol.AuthResultMessage{
				Message: protocol.NewMessage(protocol.MsgAuthResult),
				Success: false,
				Error:   "invalid token",
			}
			codec.Encode(result)
			return nil, fmt.Errorf("invalid token")
		}

		// Create client with legacy token
		client := s.createClient(conn, session, controlStream, codec, tokenCfg, log)

		// Send success
		result := &protocol.AuthResultMessage{
			Message:    protocol.NewMessage(protocol.MsgAuthResult),
			Success:    true,
			ClientID:   client.ID,
			MaxTunnels: tokenCfg.MaxTunnels,
			ServerName: s.cfg.Domain.Base,
			SessionID:  client.ID,
		}
		if err := codec.Encode(result); err != nil {
			client.Close()
			return nil, fmt.Errorf("send auth result: %w", err)
		}

		return client, nil
	}

	// No auth required - create client without token
	client := s.createClient(conn, session, controlStream, codec, nil, log)

	result := &protocol.AuthResultMessage{
		Message:    protocol.NewMessage(protocol.MsgAuthResult),
		Success:    true,
		ClientID:   client.ID,
		MaxTunnels: 10, // Default limit
		ServerName: s.cfg.Domain.Base,
		SessionID:  client.ID,
	}
	if err := codec.Encode(result); err != nil {
		client.Close()
		return nil, fmt.Errorf("send auth result: %w", err)
	}

	return client, nil
}

// createClientFromDBToken creates a client authenticated with a database token
func (s *Server) createClientFromDBToken(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, apiToken *database.APIToken, log zerolog.Logger) *Client {
	clientID := generateID()
	ctx, cancel := context.WithCancel(s.ctx)

	client := &Client{
		ID:           clientID,
		RemoteAddr:   conn.RemoteAddr().String(),
		Token:        nil, // No legacy token
		Session:      session,
		ControlCodec: codec,
		ControlConn:  controlStream,
		Tunnels:      make(map[string]*Tunnel),
		Connected:    time.Now(),
		UserID:       apiToken.UserID,
		APITokenID:   apiToken.ID,
		DBToken:      apiToken,
		server:       s,
		conn:         conn,
		log:          log.With().Str("client_id", clientID).Int64("user_id", apiToken.UserID).Logger(),
		ctx:          ctx,
		cancel:       cancel,
	}
	client.lastPing.Store(time.Now().UnixNano())

	s.clientsMu.Lock()
	s.clients[clientID] = client
	s.clientsMu.Unlock()

	return client
}

// createClientFromJWT creates a client authenticated with a JWT token
func (s *Server) createClientFromJWT(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, claims *auth.Claims, log zerolog.Logger) *Client {
	clientID := generateID()
	ctx, cancel := context.WithCancel(s.ctx)

	client := &Client{
		ID:           clientID,
		RemoteAddr:   conn.RemoteAddr().String(),
		Token:        nil, // No legacy token
		Session:      session,
		ControlCodec: codec,
		ControlConn:  controlStream,
		Tunnels:      make(map[string]*Tunnel),
		Connected:    time.Now(),
		UserID:       claims.UserID,
		server:       s,
		conn:         conn,
		log:          log.With().Str("client_id", clientID).Int64("user_id", claims.UserID).Logger(),
		ctx:          ctx,
		cancel:       cancel,
	}
	client.lastPing.Store(time.Now().UnixNano())

	s.clientsMu.Lock()
	s.clients[clientID] = client
	s.clientsMu.Unlock()

	return client
}

// isJWT checks if a token looks like a JWT (has 3 dot-separated parts)
func isJWT(token string) bool {
	if strings.HasPrefix(token, "sk_") {
		return false
	}
	parts := 0
	for _, c := range token {
		if c == '.' {
			parts++
		}
	}
	return parts == 2
}

func (s *Server) createClient(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, token *config.TokenConfig, log zerolog.Logger) *Client {
	clientID := generateID()
	ctx, cancel := context.WithCancel(s.ctx)

	client := &Client{
		ID:           clientID,
		RemoteAddr:   conn.RemoteAddr().String(),
		Token:        token,
		Session:      session,
		ControlCodec: codec,
		ControlConn:  controlStream,
		Tunnels:      make(map[string]*Tunnel),
		Connected:    time.Now(),
		server:       s,
		conn:         conn,
		log:          log.With().Str("client_id", clientID).Logger(),
		ctx:          ctx,
		cancel:       cancel,
	}
	client.lastPing.Store(time.Now().UnixNano())

	s.clientsMu.Lock()
	s.clients[clientID] = client
	s.clientsMu.Unlock()

	return client
}

func (s *Server) removeClient(clientID string) {
	s.clientsMu.Lock()
	delete(s.clients, clientID)
	s.clientsMu.Unlock()
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
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return s.clients[clientID]
}

// Client methods

func (c *Client) handle() {
	defer c.Close()

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

// OpenStream opens a new yamux stream to the client
func (c *Client) OpenStream() (net.Conn, error) {
	return c.Session.Open()
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
		c.server.unlinkUserClient(c.UserID, c.ID)

		c.server.removeClient(c.ID)
		c.log.Info().Msg("Client disconnected")
	})
}

// Helper functions

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// hashToken creates a SHA256 hash of a token for database lookup
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
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
	var tunnels []TunnelInfo

	s.userClientsMu.RLock()
	clientIDs := s.userClients[userID]
	s.userClientsMu.RUnlock()

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for _, clientID := range clientIDs {
		client, ok := s.clients[clientID]
		if !ok {
			continue
		}

		client.TunnelsMu.RLock()
		for _, tunnel := range client.Tunnels {
			tunnels = append(tunnels, TunnelInfo{
				ID:         tunnel.ID,
				Type:       string(tunnel.Type),
				Name:       tunnel.Name,
				Subdomain:  tunnel.Subdomain,
				RemotePort: tunnel.RemotePort,
				LocalPort:  tunnel.LocalPort,
				ClientID:   tunnel.ClientID,
				UserID:     client.UserID,
				CreatedAt:  tunnel.Created,
			})
		}
		client.TunnelsMu.RUnlock()
	}

	return tunnels
}

// GetAllTunnels returns all tunnels from all clients (for admin)
func (s *Server) GetAllTunnels() []TunnelInfo {
	var tunnels []TunnelInfo

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for _, client := range s.clients {
		client.TunnelsMu.RLock()
		for _, tunnel := range client.Tunnels {
			tunnels = append(tunnels, TunnelInfo{
				ID:         tunnel.ID,
				Type:       string(tunnel.Type),
				Name:       tunnel.Name,
				Subdomain:  tunnel.Subdomain,
				RemotePort: tunnel.RemotePort,
				LocalPort:  tunnel.LocalPort,
				ClientID:   tunnel.ClientID,
				UserID:     client.UserID,
				CreatedAt:  tunnel.Created,
			})
		}
		client.TunnelsMu.RUnlock()
	}

	return tunnels
}

// AdminCloseTunnel closes any tunnel by ID (admin only, no user check)
func (s *Server) AdminCloseTunnel(tunnelID string) error {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for _, client := range s.clients {
		client.TunnelsMu.RLock()
		_, exists := client.Tunnels[tunnelID]
		client.TunnelsMu.RUnlock()

		if exists {
			client.closeTunnel(tunnelID)
			return nil
		}
	}

	return fmt.Errorf("tunnel not found")
}

// CloseTunnelByID closes a tunnel by ID for a specific user
func (s *Server) CloseTunnelByID(tunnelID string, userID int64) error {
	s.userClientsMu.RLock()
	clientIDs := s.userClients[userID]
	s.userClientsMu.RUnlock()

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for _, clientID := range clientIDs {
		client, ok := s.clients[clientID]
		if !ok {
			continue
		}

		client.TunnelsMu.RLock()
		_, exists := client.Tunnels[tunnelID]
		client.TunnelsMu.RUnlock()

		if exists {
			client.closeTunnel(tunnelID)
			return nil
		}
	}

	return fmt.Errorf("tunnel not found")
}

// GetStats returns server statistics
func (s *Server) GetStats() Stats {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	stats := Stats{
		ActiveClients: len(s.clients),
	}

	for _, client := range s.clients {
		client.TunnelsMu.RLock()
		for _, tunnel := range client.Tunnels {
			stats.ActiveTunnels++
			switch tunnel.Type {
			case protocol.TunnelHTTP:
				stats.HTTPTunnels++
			case protocol.TunnelTCP:
				stats.TCPTunnels++
			case protocol.TunnelUDP:
				stats.UDPTunnels++
			}
		}
		client.TunnelsMu.RUnlock()
	}

	return stats
}

// linkUserClient links a user ID to a client ID
func (s *Server) linkUserClient(userID int64, clientID string) {
	if userID == 0 {
		return
	}

	s.userClientsMu.Lock()
	defer s.userClientsMu.Unlock()

	s.userClients[userID] = append(s.userClients[userID], clientID)
}

// unlinkUserClient removes a client ID from a user's client list
func (s *Server) unlinkUserClient(userID int64, clientID string) {
	if userID == 0 {
		return
	}

	s.userClientsMu.Lock()
	defer s.userClientsMu.Unlock()

	clients := s.userClients[userID]
	for i, id := range clients {
		if id == clientID {
			s.userClients[userID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	if len(s.userClients[userID]) == 0 {
		delete(s.userClients, userID)
	}
}
