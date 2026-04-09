package core

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/server/auth"
	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/server/database"
	"github.com/mephistofox/fxtunnel/internal/inspect"
	"github.com/mephistofox/fxtunnel/internal/server/monitor"
	"github.com/mephistofox/fxtunnel/internal/protocol"
	"github.com/mephistofox/fxtunnel/internal/server/store"
	fxtls "github.com/mephistofox/fxtunnel/internal/server/tls"
)

var (
	// subdomainRegex validates subdomain format
	subdomainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,30}[a-z0-9])?$`)

	// reservedSubdomains are subdomains that cannot be claimed by tunnel clients
	reservedSubdomains = map[string]bool{
		"api": true, "www": true, "admin": true, "mail": true,
		"smtp": true, "imap": true, "pop": true, "ftp": true,
		"ns1": true, "ns2": true, "ns3": true, "ns4": true,
		"autoconfig": true, "autodiscover": true, "_dmarc": true,
		"status": true, "metrics": true, "grafana": true,
	}
)

const (
	// yamuxMaxStreamWindowSize is the yamux stream window size for high throughput.
	yamuxMaxStreamWindowSize = 16 * 1024 * 1024 // 16MB

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
)

// blockedTCPPorts prevents SSRF via TCP tunnels to sensitive local services.
// Admin users bypass this check.
var blockedTCPPorts = map[int]bool{
	22:    true, // SSH
	25:    true, // SMTP
	53:    true, // DNS
	135:   true, // MSRPC
	139:   true, // NetBIOS
	445:   true, // SMB
	3306:  true, // MySQL
	5432:  true, // PostgreSQL
	6379:  true, // Redis
	11211: true, // Memcached
	27017: true, // MongoDB
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

	// Traffic monitor
	monitor *monitor.Monitor

	// Database integration
	db          *database.Database
	authService *auth.Service

	// Telegram admin notifications
	telegramNotifier interface {
		NotifyFirstTunnel(userID int64, displayName, tunnelType, address string, registeredAt time.Time)
	}

	// Cross-server tunnel registry (optional)
	tunnelRegistry store.TunnelRegistry

	// Custom domains
	certManager    *fxtls.CertManager
	customDomains  map[string]*database.CustomDomain // domain -> entry
	customDomainMu sync.RWMutex

	// Auth rate limiting per IP
	authLimiters sync.Map // remoteIP -> *monitor.SlidingWindow

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

	// Multi-session pool: additional data connections for parallelism
	DataSessions  []*yamux.Session
	DataConns     []net.Conn // underlying TCP connections for data sessions
	DataMu        sync.RWMutex
	sessionIdx    atomic.Uint32 // round-robin counter
	SessionSecret       string    // secret for joining additional connections
	SessionSecretExpiry time.Time // secret valid until this time

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

	// Security features
	BasicAuthHash string       // bcrypt hash
	AllowedNets   []*net.IPNet // parsed CIDRs
	AllowedIPs    []net.IP     // exact IPs (no CIDR)
	AutoClose     time.Duration // idle timeout
	MaxLifetime   time.Duration // max tunnel lifetime
	LastActivity  atomic.Int64  // UnixNano timestamp

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

	monCfg := monitor.Config{
		Enabled:           cfg.Server.Monitor.Enabled,
		DetectionInterval: cfg.Server.Monitor.DetectionInterval,
		Detection: monitor.DetectionConfig{
			UniqueIPsThreshold:     cfg.Server.Monitor.UniqueIPsThreshold,
			ShortConnRatio:         cfg.Server.Monitor.ShortConnRatio,
			UDPAmplificationFactor: cfg.Server.Monitor.UDPAmplificationFactor,
		},
	}
	s.monitor = monitor.New(monCfg, s.handleMonitorAlert)

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

func (s *Server) handleMonitorAlert(alert monitor.Alert) {
	if alert.Severity == monitor.SeverityCritical {
		s.log.Error().
			Str("tunnel", alert.TunnelID).
			Str("alert", string(alert.Type)).
			Msg("critical security alert: " + alert.Message)
	}
}

// SetDatabase sets the database for the server
func (s *Server) SetDatabase(db *database.Database) {
	s.db = db
}

// SetAuthService sets the auth service for JWT validation
func (s *Server) SetAuthService(authService *auth.Service) {
	s.authService = authService
}

// SetTunnelRegistry sets the cross-server tunnel discovery registry.
func (s *Server) SetTunnelRegistry(r store.TunnelRegistry) {
	s.tunnelRegistry = r
}

// TunnelRegistry returns the tunnel registry (may be nil).
func (s *Server) TunnelRegistry() store.TunnelRegistry {
	return s.tunnelRegistry
}

// SetTelegramNotifier sets the Telegram admin notifier for first-tunnel notifications.
func (s *Server) SetTelegramNotifier(n interface {
	NotifyFirstTunnel(userID int64, displayName, tunnelType, address string, registeredAt time.Time)
}) {
	s.telegramNotifier = n
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
				ReadTimeout:       30 * time.Second,
				WriteTimeout:      60 * time.Second,
				IdleTimeout:       120 * time.Second,
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

	// Start HTTP server with keep-alive support
	s.httpServer = &http.Server{
		Handler:           s.httpRouter,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
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
	s.monitor.Stop()

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

	// Auth rate limiting per IP (10 attempts/minute)
	if !s.allowAuth(remoteAddr) {
		log.Warn().Msg("Auth rate limited")
		conn.Close()
		return
	}

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
	yamuxCfg.MaxStreamWindowSize = yamuxMaxStreamWindowSize
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

		// Check client version against minimum required
		if s.cfg.Server.MinVersion != "" && authMsg.Version != "" {
			if authMsg.Version < s.cfg.Server.MinVersion {
				log.Warn().Str("client_version", authMsg.Version).Str("min_version", s.cfg.Server.MinVersion).
					Msg("Client version too old")
				result := &protocol.AuthResultMessage{
					Message: protocol.NewMessage(protocol.MsgAuthResult),
					Success: false,
					Error:   fmt.Sprintf("client version %s is below minimum %s, please upgrade", authMsg.Version, s.cfg.Server.MinVersion),
					Code:    protocol.ErrCodeProtocolError,
				}
				_ = codec.Encode(result)
				session.Close()
				return
			}
		}

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
	maxDS := 0 // unlimited by default
	if client.Plan != nil && !IsUnlimited(client.Plan.MaxDataSessions) {
		maxDS = client.Plan.MaxDataSessions
		if maxDS == 0 {
			maxDS = defaultMaxDataSessions
		}
	}
	if maxDS > 0 && len(client.DataSessions) >= maxDS {
		client.DataMu.Unlock()
		log.Warn().Str("client_id", client.ID).Int("current", len(client.DataSessions)).Int("max", maxDS).
			Msg("Data session limit reached")
		result := &protocol.JoinSessionResult{
			Message: protocol.NewMessage(protocol.MsgJoinSessionResult),
			Success: false,
			Error:   "data session limit reached",
		}
		_ = codec.Encode(result)
		session.Close()
		return
	}
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
	// Check session secret TTL
	if !client.SessionSecretExpiry.IsZero() && time.Now().After(client.SessionSecretExpiry) {
		return nil
	}
	return client
}

func (s *Server) removeClient(clientID string) {
	s.clientMgr.removeClient(clientID)
}

const authRateLimitPerMin = 10

func (s *Server) allowAuth(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	v, _ := s.authLimiters.LoadOrStore(host, monitor.NewSlidingWindow(authRateLimitPerMin, time.Minute))
	return v.(*monitor.SlidingWindow).Allow()
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

	// Serialize tunnel creation per user to prevent race condition on count check
	if c.UserID > 0 {
		mu := c.server.clientMgr.GetTunnelCreateMu(c.UserID)
		mu.Lock()
		defer mu.Unlock()
	}

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
		subdomain = c.server.generateUniqueSubdomain()
	}

	// Validate subdomain format
	if !subdomainRegex.MatchString(subdomain) {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodeSubdomainInvalid, "invalid subdomain format")
		return
	}

	// Block reserved subdomains
	if reservedSubdomains[subdomain] {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodeSubdomainInvalid, "subdomain is reserved")
		return
	}

	// Check subdomain permission
	if c.Token != nil && !c.Token.CanUseSubdomain(subdomain) {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodePermissionDenied, "subdomain not allowed")
		return
	}
	if c.DBToken != nil && !c.DBToken.CanUseSubdomain(subdomain) {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodePermissionDenied, "subdomain not allowed by token")
		return
	}

	// Check reserved domains in database
	if c.server.db != nil && c.UserID > 0 {
		owned, _ := c.server.db.Domains.IsOwnedByUser(subdomain, c.UserID)
		available, _ := c.server.db.Domains.IsAvailable(subdomain)
		if !available && !owned {
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeSubdomainTaken, "subdomain is reserved by another user")
			return
		}
	}

	// Register with HTTP router
	tunnelID := generateID()
	tunnel := &Tunnel{
		ID:            tunnelID,
		ClientID:      c.ID,
		Type:          protocol.TunnelHTTP,
		Name:          req.Name,
		Subdomain:     subdomain,
		LocalPort:     req.LocalPort,
		Created:       time.Now(),
		BasicAuthHash: req.BasicAuthHash,
	}

	// Parse IP allowlist
	if len(req.AllowIPs) > 0 {
		ips, nets, err := parseAllowIPs(req.AllowIPs)
		if err != nil {
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeProtocolError, fmt.Sprintf("invalid allow_ips: %v", err))
			return
		}
		tunnel.AllowedIPs = ips
		tunnel.AllowedNets = nets
	}

	// Parse auto-close duration
	if req.AutoClose != "" {
		d, err := parseTunnelDuration(req.AutoClose)
		if err != nil {
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeProtocolError, fmt.Sprintf("invalid auto_close: %v", err))
			return
		}
		tunnel.AutoClose = d
	}

	// Parse max-lifetime duration
	if req.MaxLifetime != "" {
		d, err := parseTunnelDuration(req.MaxLifetime)
		if err != nil {
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeProtocolError, fmt.Sprintf("invalid max_lifetime: %v", err))
			return
		}
		tunnel.MaxLifetime = d
	}

	// Initialize LastActivity to creation time
	tunnel.LastActivity.Store(time.Now().UnixNano())

	c.server.inspectMgr.GetOrCreateWithUser(tunnelID, c.UserID)

	if err := c.server.httpRouter.RegisterTunnel(subdomain, tunnel); err != nil {
		c.server.inspectMgr.Remove(tunnelID)
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodeSubdomainTaken, err.Error())
		return
	}

	c.TunnelsMu.Lock()
	c.Tunnels[tunnelID] = tunnel
	c.TunnelsMu.Unlock()

	c.registerTunnelMonitor(tunnel)

	url := fmt.Sprintf("http://%s.%s", subdomain, c.server.cfg.Domain.Base)

	resp := &protocol.TunnelCreatedMessage{
		Message:          protocol.NewMessage(protocol.MsgTunnelCreated),
		TunnelID:         tunnelID,
		TunnelType:       protocol.TunnelHTTP,
		Name:             req.Name,
		URL:              url,
		Subdomain:        subdomain,
		BasicAuthEnabled: tunnel.BasicAuthHash != "",
		AllowIPsCount:    len(tunnel.AllowedIPs) + len(tunnel.AllowedNets),
		AutoClose:        req.AutoClose,
		MaxLifetime:      req.MaxLifetime,
	}
	resp.RequestID = req.RequestID

	_ = c.sendControl(resp)
	c.log.Info().Str("tunnel_id", tunnelID).Str("url", url).Msg("HTTP tunnel created")
	c.registerTunnelInRegistry(tunnel)
	c.notifyFirstTunnel("HTTP", url)
}

func (c *Client) createTCPTunnel(req *protocol.TunnelRequestMessage) {
	// SSRF prevention: block sensitive ports for non-admin users
	if req.RemotePort > 0 && !c.IsAdmin && blockedTCPPorts[req.RemotePort] {
		c.sendTunnelError(req.RequestID, "", protocol.ErrCodePortUnavailable,
			fmt.Sprintf("port %d is blocked for security reasons", req.RemotePort))
		return
	}

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

	// Parse IP allowlist
	if len(req.AllowIPs) > 0 {
		ips, nets, err := parseAllowIPs(req.AllowIPs)
		if err != nil {
			listener.Close()
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeProtocolError, fmt.Sprintf("invalid allow_ips: %v", err))
			return
		}
		tunnel.AllowedIPs = ips
		tunnel.AllowedNets = nets
	}

	// Parse auto-close duration
	if req.AutoClose != "" {
		d, err := parseTunnelDuration(req.AutoClose)
		if err != nil {
			listener.Close()
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeProtocolError, fmt.Sprintf("invalid auto_close: %v", err))
			return
		}
		tunnel.AutoClose = d
	}

	// Parse max-lifetime duration
	if req.MaxLifetime != "" {
		d, err := parseTunnelDuration(req.MaxLifetime)
		if err != nil {
			listener.Close()
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeProtocolError, fmt.Sprintf("invalid max_lifetime: %v", err))
			return
		}
		tunnel.MaxLifetime = d
	}

	// Initialize LastActivity to creation time
	tunnel.LastActivity.Store(time.Now().UnixNano())

	c.TunnelsMu.Lock()
	c.Tunnels[tunnelID] = tunnel
	c.TunnelsMu.Unlock()

	c.registerTunnelMonitor(tunnel)

	// Start accepting connections
	go c.server.tcpManager.AcceptConnections(tunnel, c)

	remoteAddr := fmt.Sprintf("%s:%d", c.server.cfg.Domain.Base, port)

	resp := &protocol.TunnelCreatedMessage{
		Message:       protocol.NewMessage(protocol.MsgTunnelCreated),
		TunnelID:      tunnelID,
		TunnelType:    protocol.TunnelTCP,
		Name:          req.Name,
		RemotePort:    port,
		RemoteAddr:    remoteAddr,
		AllowIPsCount: len(tunnel.AllowedIPs) + len(tunnel.AllowedNets),
		AutoClose:     req.AutoClose,
		MaxLifetime:   req.MaxLifetime,
	}
	resp.RequestID = req.RequestID

	_ = c.sendControl(resp)
	c.log.Info().Str("tunnel_id", tunnelID).Int("port", port).Msg("TCP tunnel created")
	c.registerTunnelInRegistry(tunnel)
	c.notifyFirstTunnel("TCP", remoteAddr)
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

	// Parse IP allowlist
	if len(req.AllowIPs) > 0 {
		ips, nets, err := parseAllowIPs(req.AllowIPs)
		if err != nil {
			udpConn.Close()
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeProtocolError, fmt.Sprintf("invalid allow_ips: %v", err))
			return
		}
		tunnel.AllowedIPs = ips
		tunnel.AllowedNets = nets
	}

	// Parse auto-close duration
	if req.AutoClose != "" {
		d, err := parseTunnelDuration(req.AutoClose)
		if err != nil {
			udpConn.Close()
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeProtocolError, fmt.Sprintf("invalid auto_close: %v", err))
			return
		}
		tunnel.AutoClose = d
	}

	// Parse max-lifetime duration
	if req.MaxLifetime != "" {
		d, err := parseTunnelDuration(req.MaxLifetime)
		if err != nil {
			udpConn.Close()
			c.sendTunnelError(req.RequestID, "", protocol.ErrCodeProtocolError, fmt.Sprintf("invalid max_lifetime: %v", err))
			return
		}
		tunnel.MaxLifetime = d
	}

	// Initialize LastActivity to creation time
	tunnel.LastActivity.Store(time.Now().UnixNano())

	c.TunnelsMu.Lock()
	c.Tunnels[tunnelID] = tunnel
	c.TunnelsMu.Unlock()

	c.registerTunnelMonitor(tunnel)

	// Start handling UDP packets
	go c.server.udpManager.HandlePackets(tunnel, c)

	remoteAddr := fmt.Sprintf("%s:%d", c.server.cfg.Domain.Base, port)

	resp := &protocol.TunnelCreatedMessage{
		Message:       protocol.NewMessage(protocol.MsgTunnelCreated),
		TunnelID:      tunnelID,
		TunnelType:    protocol.TunnelUDP,
		Name:          req.Name,
		RemotePort:    port,
		RemoteAddr:    remoteAddr,
		AllowIPsCount: len(tunnel.AllowedIPs) + len(tunnel.AllowedNets),
		AutoClose:     req.AutoClose,
		MaxLifetime:   req.MaxLifetime,
	}
	resp.RequestID = req.RequestID

	_ = c.sendControl(resp)
	c.log.Info().Str("tunnel_id", tunnelID).Int("port", port).Msg("UDP tunnel created")
	c.registerTunnelInRegistry(tunnel)
	c.notifyFirstTunnel("UDP", remoteAddr)
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

func (c *Client) registerTunnelMonitor(tunnel *Tunnel) {
	var limits monitor.TunnelLimits
	if c.Plan != nil {
		limits = monitor.TunnelLimits{
			TCPConnPerMin:    c.Plan.RateLimitTCP,
			UDPPacketsPerSec: c.Plan.RateLimitUDP,
			HTTPReqPerMin:    c.Plan.RateLimitHTTP,
		}
	}
	c.server.monitor.RegisterTunnel(tunnel.ID, string(tunnel.Type), limits)
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

	c.server.monitor.RemoveTunnel(tunnelID)

	// Remove from cross-server registry
	if c.server.tunnelRegistry != nil {
		_ = c.server.tunnelRegistry.Unregister(tunnelID)
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

// registerTunnelInRegistry registers the tunnel in the cross-server Redis registry
// and starts a heartbeat goroutine that refreshes the TTL every 30 seconds.
func (c *Client) registerTunnelInRegistry(tunnel *Tunnel) {
	reg := c.server.tunnelRegistry
	if reg == nil {
		return
	}

	entry := store.TunnelEntry{
		TunnelID:   tunnel.ID,
		Type:       string(tunnel.Type),
		Name:       tunnel.Name,
		Subdomain:  tunnel.Subdomain,
		RemotePort: tunnel.RemotePort,
		LocalPort:  tunnel.LocalPort,
		ClientID:   c.ID,
		UserID:     c.UserID,
		ServerID:   "", // set by registry
		CreatedAt:  tunnel.Created,
	}

	if err := reg.Register(entry); err != nil {
		c.log.Warn().Err(err).Str("tunnel_id", tunnel.ID).Msg("Failed to register tunnel in Redis")
		return
	}

	// Heartbeat goroutine — refreshes TTL every 30s, stops when client context is done
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				// Check if tunnel still exists
				c.TunnelsMu.RLock()
				_, exists := c.Tunnels[tunnel.ID]
				c.TunnelsMu.RUnlock()
				if !exists {
					return
				}
				_ = reg.Heartbeat(tunnel.ID)
			}
		}
	}()
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

	tickCount := 0
	const tokenCheckInterval = 10 // every 10 ticks (~5 min at 30s interval)

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

			// Periodic token revocation check
			tickCount++
			if tickCount%tokenCheckInterval == 0 && c.APITokenID > 0 && c.server.db != nil {
				if _, err := c.server.db.Tokens.GetByID(c.APITokenID); err != nil {
					c.log.Warn().Int64("token_id", c.APITokenID).Msg("Token revoked or deleted, closing connection")
					c.Close()
					return
				}
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

// notifyFirstTunnel checks if this is the user's first-ever tunnel and notifies admin.
func (c *Client) notifyFirstTunnel(tunnelType, address string) {
	if c.server.telegramNotifier == nil || c.server.db == nil || c.UserID <= 0 {
		return
	}

	isFirst, err := c.server.db.Users.SetFirstTunnelAt(c.UserID)
	if err != nil {
		c.log.Error().Err(err).Msg("Failed to check first tunnel")
		return
	}
	if !isFirst {
		return
	}

	user, err := c.server.db.Users.GetByID(c.UserID)
	if err != nil {
		return
	}
	c.server.telegramNotifier.NotifyFirstTunnel(c.UserID, user.DisplayName, tunnelType, address, user.CreatedAt)
}

// Close closes the client connection
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.cancel()

		// Close all tunnels
		c.TunnelsMu.Lock()
		for tunnelID, tunnel := range c.Tunnels {
			c.server.monitor.RemoveTunnel(tunnelID)
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

func generateID() string {
	b := make([]byte, 9) // 9 bytes = 12 base64url chars
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	// Base36 encode: lowercase alphanumeric, URL-safe, short
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	out := make([]byte, 12)
	for i := range out {
		out[i] = alphabet[int(b[i%len(b)])%len(alphabet)]
	}
	return string(out)
}

func generateShortID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
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

