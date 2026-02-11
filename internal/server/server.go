package server

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/auth"
	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/database"
	"github.com/mephistofox/fxtun.dev/internal/inspect"
	"github.com/mephistofox/fxtun.dev/internal/protocol"
	fxtls "github.com/mephistofox/fxtun.dev/internal/tls"
)

const (
	// yamuxKeepAliveInterval is the interval between yamux keepalive probes.
	yamuxKeepAliveInterval = 10 * time.Second

	// yamuxConnectionWriteTimeout is the timeout for writing to a yamux connection.
	yamuxConnectionWriteTimeout = 30 * time.Second

	// authTimeout is the maximum time to wait for an authentication message.
	authTimeout = 30 * time.Second

	// keepaliveInterval is the interval between server-side keepalive checks.
	keepaliveInterval = 30 * time.Second

	// clientTimeout is the duration after which a client is considered unresponsive.
	clientTimeout = 90 * time.Second

	// drainTimeout is the maximum time to wait for active connections to drain during shutdown.
	drainTimeout = 10 * time.Second

	// defaultMaxTunnels is the default maximum number of tunnels per client.
	defaultMaxTunnels = 10

	// defaultInspectMaxEntries is the default capacity for the inspect buffer.
	defaultInspectMaxEntries = 1000

	// defaultMaxControlConns is the default global limit on concurrent control connections.
	defaultMaxControlConns = 1000

	// defaultMaxConnsPerIP is the default per-IP connection limit.
	defaultMaxConnsPerIP = 50

	// maxDataSessionsPerClient is the maximum number of data sessions per client.
	maxDataSessionsPerClient = 32
)

// subdomainRegex validates subdomain format: alphanumeric, may contain hyphens, 1-32 chars.
var subdomainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,30}[a-z0-9])?$`)

// reservedSubdomains is a blocklist of subdomains that cannot be used for tunnels.
var reservedSubdomains = map[string]struct{}{
	"www":  {},
	"api":  {},
	"admin": {},
	"mail": {},
	"ftp":  {},
	"smtp": {},
	"imap": {},
	"pop":  {},
	"ns1":  {},
	"ns2":  {},
	"mx":   {},
	"app":  {},
}

// Server is the main tunnel server
type Server struct {
	cfg    *config.ServerConfig
	log    zerolog.Logger

	// Listeners
	controlListener  net.Listener
	httpListener     net.Listener
	httpsListener    net.Listener
	httpsServer      *http.Server

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

	// Connection limits
	connSem    chan struct{} // semaphore for global max connections
	ipConnCount sync.Map     // map[string]*int32 — per-IP connection count

	// DDoS protection
	ipBan        *IPBanManager
	acceptLimiter *acceptRateLimiter

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

	// Multi-session pool: additional data connections for parallelism
	DataSessions  []*yamux.Session
	DataConns     []net.Conn // underlying TCP connections for data sessions
	DataMu        sync.RWMutex
	sessionIdx    atomic.Uint32 // round-robin counter
	SessionSecret string        // secret for joining additional connections

	// Database integration
	UserID     int64              // 0 if legacy token
	APITokenID int64              // 0 if legacy token
	DBToken    *database.APIToken // nil if legacy token
	IsAdmin    bool               // true if user is admin
	Plan       *database.Plan     // user's plan (nil if none)

	server    *Server
	conn      net.Conn
	log       zerolog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex // for writing to control stream
	closeOnce sync.Once

	// Stream pool: pre-opened yamux streams for low-latency connection handling
	streamPool chan net.Conn

	// Bandwidth limiter (per-client, based on plan)
	bwLimiter *BandwidthLimiter
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

	// Per-tunnel HTTP request concurrency limiter
	reqSem chan struct{}
}

// New creates a new server
func New(cfg *config.ServerConfig, log zerolog.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	maxConns := cfg.Server.MaxControlConns
	if maxConns <= 0 {
		maxConns = defaultMaxControlConns
	}

	s := &Server{
		cfg:           cfg,
		log:           log.With().Str("component", "server").Logger(),
		clientMgr:     NewClientManager(log.With().Str("component", "server").Logger()),
		customDomains: make(map[string]*database.CustomDomain),
		connSem:       make(chan struct{}, maxConns),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Apply configurable buffer sizes
	initProxyBufSize(cfg.Server.ProxyBufferSize)
	if cfg.Server.TCPBufferSize > 0 {
		tcpBufSize = cfg.Server.TCPBufferSize
	}

	// Initialize IP ban manager
	if cfg.Server.IPBan.Enabled {
		s.ipBan = NewIPBanManager(cfg.Server.IPBan, log)
	}

	// Initialize accept rate limiter
	s.acceptLimiter = newAcceptRateLimiter(cfg.Server.AcceptRateGlobal, cfg.Server.AcceptRatePerIP)

	s.httpRouter = NewHTTPRouter(s, log)
	s.tcpManager = NewTCPManager(s, log)
	s.udpManager = NewUDPManager(s, log)

	capacity := 0
	if cfg.Inspect.Enabled {
		capacity = cfg.Inspect.MaxEntries
		if capacity == 0 {
			capacity = defaultInspectMaxEntries
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

// HTTPRouter returns the HTTP router for replay support.
func (s *Server) HTTPRouter() *HTTPRouter {
	return s.httpRouter
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
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
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

	// Start HTTPS listener for custom domains (if CertManager is available)
	if s.certManager != nil && s.cfg.TLS.HTTPSPort > 0 {
		httpsAddr := fmt.Sprintf(":%d", s.cfg.TLS.HTTPSPort)
		tlsListener, err := newReusePortListener(s.ctx, httpsAddr)
		if err != nil {
			s.log.Warn().Err(err).Str("addr", httpsAddr).Msg("Failed to start HTTPS listener for custom domains")
		} else {
			s.httpsListener = tls.NewListener(tlsListener, s.certManager.TLSConfig())
			s.httpsServer = &http.Server{
				Handler:           s.httpRouter,
				ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout:       s.httpReadTimeout(),
				WriteTimeout:      s.httpWriteTimeout(),
				IdleTimeout:       s.httpIdleTimeout(),
			}
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				if err := s.httpsServer.Serve(s.httpsListener); err != nil && err != http.ErrServerClosed {
					s.log.Error().Err(err).Msg("HTTPS server error")
				}
			}()
			s.log.Info().Str("addr", httpsAddr).Msg("HTTPS listener started for custom domains")
		}
	}

	// Accept control connections
	s.wg.Add(1)
	go s.acceptControlConnections()

	// Periodic cleanup of accept rate limiter per-IP entries
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.acceptLimiter.Cleanup()
			}
		}
	}()

	// Start HTTP server with keep-alive support
	s.httpServer = &http.Server{
		Handler:           s.httpRouter,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       s.httpReadTimeout(),
		WriteTimeout:      s.httpWriteTimeout(),
		IdleTimeout:       s.httpIdleTimeout(),
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
	if s.httpsListener != nil {
		s.httpsListener.Close()
	}

	// Phase 2: drain in-flight connections (max 10s)
	s.log.Info().Msg("Draining active connections...")
	drainCtx, drainCancel := context.WithTimeout(context.Background(), drainTimeout)
	defer drainCancel()

	// Gracefully shutdown HTTP/HTTPS servers (drains keep-alive connections)
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(drainCtx); err != nil {
			s.log.Warn().Err(err).Msg("HTTP server shutdown error")
		}
	}
	if s.httpsServer != nil {
		if err := s.httpsServer.Shutdown(drainCtx); err != nil {
			s.log.Warn().Err(err).Msg("HTTPS server shutdown error")
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

	// Phase 3: notify clients and gracefully close sessions
	clients := s.clientMgr.allClients()

	// Send shutdown notification and GoAway to all clients
	for _, c := range clients {
		shutdownMsg := &protocol.ServerShutdownMessage{
			Message: protocol.NewMessage(protocol.MsgServerShutdown),
			Reason:  "server shutting down",
		}
		_ = c.sendControl(shutdownMsg)
		if c.Session != nil {
			_ = c.Session.GoAway()
		}
		c.DataMu.RLock()
		for _, ds := range c.DataSessions {
			_ = ds.GoAway()
		}
		c.DataMu.RUnlock()
	}

	// Allow in-flight streams to finish
	if len(clients) > 0 {
		time.Sleep(2 * time.Second)
	}

	s.cancel()

	for _, c := range clients {
		c.Close()
	}

	// Stop managers
	s.tcpManager.Stop()
	s.udpManager.Stop()

	s.inspectMgr.Close()

	if s.certManager != nil {
		s.certManager.Stop()
	}

	if s.ipBan != nil {
		s.ipBan.Stop()
	}

	s.wg.Wait()
	s.log.Info().Msg("Server stopped")
	return nil
}

func (s *Server) acceptControlConnections() {
	defer s.wg.Done()

	maxPerIPCfg := s.cfg.Server.MaxConnsPerIP
	if maxPerIPCfg <= 0 {
		maxPerIPCfg = defaultMaxConnsPerIP
	}
	maxPerIP := int32(maxPerIPCfg) //nolint:gosec // config values are small positive ints, overflow impossible

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

		ip := connIP(conn)

		// Check IP ban
		if s.ipBan != nil && s.ipBan.IsBanned(ip) {
			s.log.Debug().Str("ip", ip).Msg("Rejected banned IP")
			conn.Close()
			continue
		}

		// Accept rate limiting
		if !s.acceptLimiter.Allow(ip) {
			s.log.Warn().Str("ip", ip).Msg("Accept rate limit exceeded, rejecting")
			if s.ipBan != nil {
				s.ipBan.RecordViolation(ip, ViolationFlood)
			}
			conn.Close()
			continue
		}

		// Global connection limit (non-blocking check)
		select {
		case s.connSem <- struct{}{}:
			// acquired
		default:
			s.log.Warn().Str("remote", conn.RemoteAddr().String()).Msg("Global connection limit reached, rejecting")
			conn.Close()
			continue
		}

		// Per-IP connection limit
		if !s.ipConnAcquire(ip, maxPerIP) {
			s.log.Warn().Str("ip", ip).Msg("Per-IP connection limit reached, rejecting")
			<-s.connSem // release global slot
			conn.Close()
			continue
		}

		s.wg.Add(1)
		go func() {
			defer func() {
				s.ipConnRelease(ip)
				<-s.connSem
			}()
			s.handleControlConnection(conn)
		}()
	}
}

// connIP extracts the IP address (without port) from a connection.
func connIP(conn net.Conn) string {
	addr := conn.RemoteAddr().String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// ipConnAcquire increments the per-IP counter and returns false if the limit is reached.
func (s *Server) ipConnAcquire(ip string, max int32) bool {
	val, _ := s.ipConnCount.LoadOrStore(ip, new(int32))
	counter := val.(*int32)
	n := atomic.AddInt32(counter, 1)
	if n > max {
		atomic.AddInt32(counter, -1)
		return false
	}
	return true
}

// ipConnRelease decrements the per-IP counter.
func (s *Server) ipConnRelease(ip string) {
	val, ok := s.ipConnCount.Load(ip)
	if !ok {
		return
	}
	counter := val.(*int32)
	if atomic.AddInt32(counter, -1) <= 0 {
		s.ipConnCount.Delete(ip)
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
	yamuxCfg.KeepAliveInterval = yamuxKeepAliveInterval
	yamuxCfg.MaxStreamWindowSize = s.yamuxWindowSize()
	yamuxCfg.ConnectionWriteTimeout = yamuxConnectionWriteTimeout
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
	_ = controlStream.SetReadDeadline(time.Now().Add(authTimeout))

	// Read auth message
	data, baseMsg, err := codec.DecodeRaw()
	if err != nil {
		log.Error().Err(err).Msg("Failed to read auth message")
		session.Close()
		return
	}

	_ = controlStream.SetReadDeadline(time.Time{}) // Clear deadline

	switch baseMsg.Type {
	case protocol.MsgJoinSession:
		// Additional data connection joining an existing client
		s.handleJoinSession(conn, session, controlStream, codec, data, log)
		return

	case protocol.MsgAuth:
		parsed, err := protocol.ParseMessage(data, protocol.MsgAuth)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse auth message")
			session.Close()
			return
		}

		authMsg := parsed.(*protocol.AuthMessage)

		// Authenticate
		client, err := s.authenticate(conn, session, controlStream, codec, authMsg, log)
		if err != nil {
			log.Warn().Err(err).Msg("Authentication failed")
			if s.ipBan != nil {
				s.ipBan.RecordViolation(connIP(conn), ViolationAuth)
			}
			session.Close()
			return
		}

		log = log.With().Str("client_id", client.ID).Logger()
		log.Info().Msg("Client authenticated")

		// Handle client messages
		client.handle()

	default:
		log.Error().Str("type", string(baseMsg.Type)).Msg("Expected auth or join_session message")
		s.sendError(codec, protocol.ErrCodeProtocolError, "expected auth or join_session message", true)
		session.Close()
	}
}

func (s *Server) handleJoinSession(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, data []byte, log zerolog.Logger) {
	parsed, err := protocol.ParseMessage(data, protocol.MsgJoinSession)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse join_session message")
		session.Close()
		return
	}
	joinMsg := parsed.(*protocol.JoinSessionMessage)

	client := s.findClientBySecret(joinMsg.ClientID, joinMsg.Secret)
	if client == nil {
		log.Warn().Str("client_id", joinMsg.ClientID).Msg("Join session failed: invalid client or secret")
		result := &protocol.JoinSessionResult{
			Message: protocol.NewMessage(protocol.MsgJoinSessionResult),
			Success: false,
			Error:   "invalid client_id or secret",
		}
		_ = codec.Encode(result)
		session.Close()
		return
	}

	// Enforce data session limit
	client.DataMu.Lock()
	if len(client.DataSessions) >= maxDataSessionsPerClient {
		client.DataMu.Unlock()
		log.Warn().Str("client_id", joinMsg.ClientID).Int("count", len(client.DataSessions)).Msg("Data session limit reached")
		result := &protocol.JoinSessionResult{
			Message: protocol.NewMessage(protocol.MsgJoinSessionResult),
			Success: false,
			Error:   "data session limit reached",
		}
		_ = codec.Encode(result)
		session.Close()
		return
	}

	// Add data session to client
	client.DataSessions = append(client.DataSessions, session)
	client.DataConns = append(client.DataConns, conn)
	client.DataMu.Unlock()

	// Send success
	result := &protocol.JoinSessionResult{
		Message: protocol.NewMessage(protocol.MsgJoinSessionResult),
		Success: true,
	}
	_ = codec.Encode(result)

	// Close the control stream — server will use session.Open() for data streams
	controlStream.Close()

	log.Info().Str("client_id", client.ID).Int("data_sessions", len(client.DataSessions)).Msg("Data session joined")
}

func (s *Server) findClientBySecret(clientID, secret string) *Client {
	client := s.clientMgr.GetClient(clientID)
	if client == nil {
		return nil
	}
	if client.SessionSecret == "" || client.SessionSecret != secret {
		return nil
	}
	return client
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
	_ = codec.Encode(msg)
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

	// Global limit from plan
	globalMax := defaultMaxTunnels
	if c.Plan != nil {
		if IsUnlimited(c.Plan.MaxTunnels) {
			globalMax = 0 // no global limit
		} else {
			globalMax = c.Plan.MaxTunnels
		}
	}

	// Per-token limit
	tokenMax := 0
	if c.Token != nil && c.Token.MaxTunnels > 0 {
		tokenMax = c.Token.MaxTunnels
	}
	if c.DBToken != nil && c.DBToken.MaxTunnels > 0 {
		tokenMax = c.DBToken.MaxTunnels
	}

	var tunnelCount int
	if c.UserID > 0 {
		tunnelCount = c.server.clientMgr.CountTunnelsByUserID(c.UserID)
	} else {
		c.TunnelsMu.RLock()
		tunnelCount = len(c.Tunnels)
		c.TunnelsMu.RUnlock()
	}

	if globalMax > 0 && tunnelCount >= globalMax {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodeTunnelLimit, "tunnel limit reached")
		return
	}

	// Also check per-token limit
	if tokenMax > 0 {
		c.TunnelsMu.RLock()
		clientTunnels := len(c.Tunnels)
		c.TunnelsMu.RUnlock()
		if clientTunnels >= tokenMax {
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeTunnelLimit, "token tunnel limit reached")
			return
		}
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

	// Validate subdomain format
	if !subdomainRegex.MatchString(subdomain) {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodeSubdomainInvalid, "invalid subdomain format")
		return
	}

	// Check against reserved subdomain blocklist
	if _, reserved := reservedSubdomains[subdomain]; reserved {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodeSubdomainInvalid, "subdomain is reserved")
		return
	}

	// Check subdomain permission
	if c.Token != nil && !c.Token.CanUseSubdomain(subdomain) {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodePermissionDenied, "subdomain not allowed")
		return
	}

	// Register with HTTP router
	tunnelID := generateID()
	maxConcurrent := c.server.cfg.Server.MaxConcurrentRequestsPerTunnel
	if maxConcurrent <= 0 {
		maxConcurrent = 100
	}
	tunnel := &Tunnel{
		ID:        tunnelID,
		ClientID:  c.ID,
		Type:      protocol.TunnelHTTP,
		Name:      req.Name,
		Subdomain: subdomain,
		LocalPort: req.LocalPort,
		Created:   time.Now(),
		reqSem:    make(chan struct{}, maxConcurrent),
	}

	if err := c.server.httpRouter.RegisterTunnel(subdomain, tunnel); err != nil {
		c.log.Warn().Err(err).Str("subdomain", subdomain).Msg("Failed to register HTTP tunnel")
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodeSubdomainTaken, "subdomain already in use")
		return
	}

	c.server.inspectMgr.GetOrCreateWithUser(tunnelID, c.UserID)

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

	_ = c.sendControl(resp)
	c.log.Info().Str("tunnel_id", tunnelID).Str("url", url).Msg("HTTP tunnel created")
}

func (c *Client) createTCPTunnel(req *protocol.TunnelRequestMessage) {
	port, listener, err := c.server.tcpManager.AllocatePort(req.RemotePort)
	if err != nil {
		c.log.Warn().Err(err).Int("requested_port", req.RemotePort).Msg("Failed to allocate TCP port")
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodePortUnavailable, "resource allocation failed")
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

	_ = c.sendControl(resp)
	c.log.Info().Str("tunnel_id", tunnelID).Int("port", port).Msg("TCP tunnel created")
}

func (c *Client) createUDPTunnel(req *protocol.TunnelRequestMessage) {
	port, udpConn, err := c.server.udpManager.AllocatePort(req.RemotePort)
	if err != nil {
		c.log.Warn().Err(err).Int("requested_port", req.RemotePort).Msg("Failed to allocate UDP port")
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodePortUnavailable, "resource allocation failed")
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

	_ = c.sendControl(resp)
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
	_ = c.sendControl(resp)

	c.log.Info().Str("tunnel_id", tunnelID).Msg("Tunnel closed")
}

func (c *Client) handleConnectionAccept(data []byte) {
	// This is handled via yamux streams directly
}

func (c *Client) handlePing() {
	pong := &protocol.PongMessage{
		Message: protocol.NewMessage(protocol.MsgPong),
	}
	_ = c.sendControl(pong)
}

func (c *Client) keepalive() {
	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if time.Since(time.Unix(0, c.lastPing.Load())) > clientTimeout {
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
	_ = c.sendControl(msg)
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

		// Close all data sessions
		c.DataMu.Lock()
		for _, ds := range c.DataSessions {
			ds.Close()
		}
		for _, dc := range c.DataConns {
			dc.Close()
		}
		c.DataSessions = nil
		c.DataConns = nil
		c.DataMu.Unlock()

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
	_, _ = rand.Read(b)
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

// httpReadTimeout returns the configured HTTP read timeout or a default.
func (s *Server) httpReadTimeout() time.Duration {
	if s.cfg.Server.HTTPReadTimeout > 0 {
		return s.cfg.Server.HTTPReadTimeout
	}
	return 30 * time.Second
}

// httpWriteTimeout returns the configured HTTP write timeout or a default.
func (s *Server) httpWriteTimeout() time.Duration {
	if s.cfg.Server.HTTPWriteTimeout > 0 {
		return s.cfg.Server.HTTPWriteTimeout
	}
	return 120 * time.Second
}

// httpIdleTimeout returns the configured HTTP idle timeout or a default.
func (s *Server) httpIdleTimeout() time.Duration {
	if s.cfg.Server.HTTPIdleTimeout > 0 {
		return s.cfg.Server.HTTPIdleTimeout
	}
	return 120 * time.Second
}

// yamuxWindowSize returns the configured yamux window size or a default.
func (s *Server) yamuxWindowSize() uint32 {
	if s.cfg.Server.YamuxWindowSize > 0 {
		return uint32(s.cfg.Server.YamuxWindowSize) //nolint:gosec // config value validated
	}
	return 4 * 1024 * 1024 // 4MB default
}

// IPBanManager returns the server's IP ban manager (may be nil).
func (s *Server) IPBanManager() *IPBanManager {
	return s.ipBan
}

