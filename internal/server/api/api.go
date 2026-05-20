package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/server/api/dto"
	"github.com/mephistofox/fxtunnel/internal/server/auth"
	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/server/database"
	"github.com/mephistofox/fxtunnel/internal/server/email"
	"github.com/mephistofox/fxtunnel/internal/inspect"
	"github.com/mephistofox/fxtunnel/internal/server/payment"
	"github.com/mephistofox/fxtunnel/internal/server/store"
	"github.com/mephistofox/fxtunnel/internal/server/telegram"
	fxtls "github.com/mephistofox/fxtunnel/internal/server/tls"
)

// TunnelInfo represents tunnel information from the server
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

// TunnelProvider is an interface for getting tunnel information
type TunnelProvider interface {
	GetTunnelsByUserID(userID int64) []TunnelInfo
	CloseTunnelByID(tunnelID string, userID int64) error
	GetStats() Stats
	GetAllTunnels() []TunnelInfo
	AdminCloseTunnel(tunnelID string) error
}

// InspectProvider provides access to traffic inspection buffers.
type InspectProvider interface {
	Get(tunnelID string) *inspect.RingBuffer
	Enabled() bool
	AddAndPersist(tunnelID string, ex *inspect.CapturedExchange)
	ListPersisted(tunnelID string, offset, limit int) ([]*inspect.CapturedExchange, int, error)
	ListPersistedByHostAndUser(host string, userID int64, offset, limit int) ([]*inspect.CapturedExchange, int, error)
	GetPersisted(id string) (*inspect.CapturedExchange, error)
}

// ReplayProvider sends an HTTP request through a tunnel and returns the response.
type ReplayProvider interface {
	ReplayRequest(subdomain string, req *http.Request) (*inspect.ReplayResult, error)
}

// CustomDomainManager provides custom domain cache and TLS cert management.
type CustomDomainManager interface {
	AddCustomDomain(d *database.CustomDomain)
	RemoveCustomDomain(domain string)
	CertManager() *fxtls.CertManager
}

// Server represents the API server
type Server struct {
	cfg                  *config.ServerConfig
	db                   *database.Database
	authService          *auth.Service
	tunnelProvider       TunnelProvider
	inspectProvider      InspectProvider
	customDomainManager  CustomDomainManager
	replayProvider       ReplayProvider
	notifier             *email.Notifier
	telegramNotifier     *telegram.AdminNotifier
	paymentProviders     *payment.Registry
	router               chi.Router
	httpServer     *http.Server
	log            zerolog.Logger
	baseDomain     string
	downloadsPath  string
	version        string
	minVersion     string
	deviceStore    store.DeviceStore
	oauthStore     store.OAuthStore
	nodeRegistry   store.NodeRegistry
	shutdownCh     chan struct{}
}

// Option configures the API server.
type Option func(*Server)

// WithDeviceStore overrides the default in-memory device store.
func WithDeviceStore(ds store.DeviceStore) Option {
	return func(s *Server) { s.deviceStore = ds }
}

// WithOAuthStore overrides the default in-memory OAuth store.
func WithOAuthStore(os store.OAuthStore) Option {
	return func(s *Server) { s.oauthStore = os }
}

// WithNodeRegistry sets the node registry for edge node management.
func WithNodeRegistry(nr store.NodeRegistry) Option {
	return func(s *Server) { s.nodeRegistry = nr }
}

// New creates a new API server
func New(cfg *config.ServerConfig, db *database.Database, authService *auth.Service, tunnelProvider TunnelProvider, inspectProvider InspectProvider, customDomainManager CustomDomainManager, log zerolog.Logger, opts ...Option) *Server {
	memDevice := newDeviceStore()
	memOAuth := newOAuthStore()

	s := &Server{
		cfg:                  cfg,
		db:                   db,
		authService:          authService,
		tunnelProvider:       tunnelProvider,
		inspectProvider:      inspectProvider,
		customDomainManager:  customDomainManager,
		log:            log.With().Str("component", "api").Logger(),
		baseDomain:     cfg.Domain.Base,
		downloadsPath:  cfg.Downloads.Path,
		deviceStore:    memDevice,
		oauthStore:     memOAuth,
		shutdownCh:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Start cleanup goroutines only for in-memory stores
	if s.deviceStore == memDevice {
		go memDevice.Cleanup(s.shutdownCh)
	}
	if s.oauthStore == memOAuth {
		go memOAuth.Cleanup(s.shutdownCh)
	}

	s.setupRoutes()
	return s
}

// SetReplayProvider sets the replay provider for inspect replay feature.
func (s *Server) SetReplayProvider(rp ReplayProvider) {
	s.replayProvider = rp
}

// SetNotifier sets the email notifier for payment notifications.
func (s *Server) SetNotifier(n *email.Notifier) {
	s.notifier = n
}

// SetTelegramNotifier sets the Telegram admin notifier.
func (s *Server) SetTelegramNotifier(n *telegram.AdminNotifier) {
	s.telegramNotifier = n
}

// SetPaymentProviders sets the payment provider registry.
func (s *Server) SetPaymentProviders(r *payment.Registry) {
	s.paymentProviders = r
}

// SetVersion sets the server version string for health endpoint.
func (s *Server) SetVersion(version string) {
	s.version = version
}

// SetMinVersion sets the minimum required client version.
func (s *Server) SetMinVersion(v string) {
	s.minVersion = v
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	// Replaces the chi middleware.RealIP — that one trusts X-Forwarded-For
	// from any TCP source, which is unsafe if the API can be reached
	// directly (bypassing nginx). Our version only honours the headers when
	// the immediate source is in auth.trusted_proxies (default: loopback).
	r.Use(trustedRealIPMiddleware(s.cfg.Auth.TrustedProxies))
	r.Use(s.loggingMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	r.Use(securityHeadersMiddleware)

	r.Use(metricsMiddleware)

	// Rate limiting
	if s.cfg.Web.RateLimit.Enabled {
		globalRL := newIPRateLimiter(s.cfg.Web.RateLimit.GlobalPerMin)
		globalRL.cleanup(s.shutdownCh, 5*time.Minute)
		r.Use(rateLimitMiddleware(globalRL))
	}

	// CORS
	corsOrigins := s.cfg.Web.CORSOrigins
	allowCredentials := len(corsOrigins) > 0
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"https://" + s.cfg.Domain.Base}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: allowCredentials,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", s.handleHealth)
	r.Get("/install.sh", s.handleInstallScript)
	r.Get("/install.ps1", s.handleInstallPS1)
	r.Group(func(r chi.Router) {
		r.Use(auth.MiddlewareWithDB(s.authService, s.db))
		r.Use(auth.AdminMiddleware)
		r.Handle("/metrics", metricsHandler())
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Version (public)
		r.Get("/version", s.handleVersion)

		// Public routes
		r.Route("/auth", func(r chi.Router) {
			if s.cfg.Web.RateLimit.Enabled {
				authRL := newIPRateLimiter(s.cfg.Web.RateLimit.AuthPerMin)
				authRL.cleanup(s.shutdownCh, 5*time.Minute)
				r.Use(rateLimitMiddleware(authRL))
			}
			r.Post("/register", s.handleRegister)
			r.Post("/login", s.handleLogin)
			r.Post("/refresh", s.handleRefresh)
			r.Post("/device/code", s.handleDeviceCode)
			r.Get("/device/token", s.handleDevicePoll)
			r.Get("/github", s.handleGitHubAuth)
			r.Get("/github/callback", s.handleGitHubCallback)
			r.Get("/google", s.handleGoogleAuth)
			r.Get("/google/callback", s.handleGoogleCallback)
			r.Post("/exchange", s.handleOAuthExchange)
		})

		// Downloads (public)
		r.Route("/downloads", func(r chi.Router) {
			r.Get("/", s.handleListDownloads)
			r.Get("/{platform}", s.handleDownload)
		})

		// Plans (public)
		r.Get("/plans/public", s.handleListPublicPlans)

		// Exchange rate (public, cached)
		r.Get("/exchange-rate", s.handleExchangeRate)

		// Payment callbacks (public, from YooKassa)
		r.Route("/payments", func(r chi.Router) {
			r.Post("/webhook", s.handlePaymentWebhook)        // YooKassa webhook
			r.Post("/webhook/creem", s.handleCreemWebhook)     // Creem webhook
			r.Get("/success", s.handlePaymentSuccess)          // Return URL redirect
			r.Get("/fail", s.handlePaymentFail)                // Fail redirect
		})

		// Edge node API (authenticated with hub_token)
		if s.cfg.Node.HubToken != "" {
			r.Route("/nodes", func(r chi.Router) {
				r.Use(s.nodeTokenMiddleware)
				r.Post("/register", s.handleNodeRegister)
				r.Post("/heartbeat", s.handleNodeHeartbeat)
			})
			r.Route("/internal", func(r chi.Router) {
				r.Use(s.nodeTokenMiddleware)
				r.Post("/auth/verify", s.handleVerifyClientToken)
				r.Get("/tls-cert", s.handleNodeTLSCert)
			})
		}

		// SSE inspect stream (no timeout, long-lived connection)
		r.Group(func(r chi.Router) {
			r.Use(auth.MiddlewareWithDB(s.authService, s.db))
			r.Get("/tunnels/{id}/inspect/stream", s.handleInspectStream)
		})

		// SSE admin stats stream (no timeout, long-lived connection)
		// Uses OptionalMiddleware: auth via header if present, otherwise handler falls back to ?token= query param
		r.Group(func(r chi.Router) {
			r.Use(auth.OptionalMiddleware(s.authService))
			r.Get("/admin/stats/stream", s.handleAdminStatsStream)
		})

		// Protected routes (with timeout)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(30 * time.Second))
			r.Use(auth.MiddlewareWithDB(s.authService, s.db))

			// Auth
			r.Post("/auth/logout", s.handleLogout)
			r.Post("/auth/device/authorize", s.handleDeviceAuthorize)
			r.Post("/auth/github/link", s.handleGitHubLink)
			r.Post("/auth/google/link", s.handleGoogleLink)

			// TOTP
			r.Route("/auth/totp", func(r chi.Router) {
				r.Post("/enable", s.handleTOTPEnable)
				r.Post("/verify", s.handleTOTPVerify)
				r.Post("/disable", s.handleTOTPDisable)
			})

			// Profile
			r.Route("/profile", func(r chi.Router) {
				r.Get("/", s.handleGetProfile)
				r.Put("/", s.handleUpdateProfile)
				r.Put("/password", s.handleChangePassword)
			})

			// Tokens
			r.Route("/tokens", func(r chi.Router) {
				r.Get("/", s.handleListTokens)
				r.Post("/", s.handleCreateToken)
				r.Delete("/{id}", s.handleDeleteToken)
			})

			// Domains
			r.Route("/domains", func(r chi.Router) {
				r.Get("/", s.handleListDomains)
				r.Post("/", s.handleReserveDomain)
				r.Delete("/{id}", s.handleReleaseDomain)
				r.Get("/check/{subdomain}", s.handleCheckDomain)
			})

			// Custom domains
			r.Route("/custom-domains", func(r chi.Router) {
				r.Get("/", s.handleListCustomDomains)
				r.Post("/", s.handleAddCustomDomain)
				r.Delete("/{id}", s.handleDeleteCustomDomain)
				r.Post("/{id}/verify", s.handleVerifyCustomDomain)
			})

			// Tunnels
			r.Route("/tunnels", func(r chi.Router) {
				r.Get("/", s.handleListTunnels)
				r.Delete("/{id}", s.handleCloseTunnel)
				r.Get("/{id}/inspect", s.handleListExchanges)
				r.Get("/{id}/inspect/status", s.handleInspectStatus)
				r.Get("/{id}/inspect/{exchangeId}", s.handleGetExchange)
				r.Delete("/{id}/inspect", s.handleClearExchanges)
				r.Post("/{id}/inspect/{exchangeId}/replay", s.handleReplayExchange)
			})

			// Sync
			r.Route("/sync", func(r chi.Router) {
				r.Get("/", s.handleGetSyncData)
				r.Post("/", s.handleSync)
				r.Put("/bundles", s.handleSyncBundles)
				r.Put("/settings", s.handleSyncSettings)
				r.Post("/history", s.handleAddHistory)
				r.Delete("/history", s.handleClearHistory)
				r.Get("/history/stats", s.handleGetHistoryStats)
			})

			// Subscription
			r.Route("/subscription", func(r chi.Router) {
				r.Get("/", s.handleGetSubscription)
				r.Post("/checkout", s.handleCheckout)
				r.Post("/cancel", s.handleCancelSubscription)
				r.Post("/change", s.handleChangePlan)
				r.Get("/payments", s.handleGetPayments)
			})

			// Admin routes
			r.Route("/admin", func(r chi.Router) {
				r.Use(auth.AdminMiddleware)

				r.Get("/stats", s.handleGetStats)
				r.Get("/users", s.handleListUsers)
				r.Get("/users/{id}", s.handleGetUserDetail)
				r.Put("/users/{id}", s.handleUpdateUser)
				r.Delete("/users/{id}", s.handleDeleteUser)
				r.Get("/audit-logs", s.handleListAuditLogs)
				r.Get("/tunnels", s.handleListAllTunnels)
				r.Delete("/tunnels/{id}", s.handleAdminCloseTunnel)

				r.Post("/users/merge", s.handleMergeUsers)
				r.Post("/users/{id}/reset-password", s.handleAdminResetPassword)
				r.Post("/users/{id}/grant-subscription", s.handleAdminGrantSubscription)

				r.Get("/custom-domains", s.handleAdminListCustomDomains)
				r.Delete("/custom-domains/{id}", s.handleAdminDeleteCustomDomain)

				r.Get("/certificates", s.handleAdminListCertificates)

				r.Get("/plans", s.handleListPlans)
				r.Post("/plans", s.handleCreatePlan)
				r.Put("/plans/{id}", s.handleUpdatePlan)
				r.Delete("/plans/{id}", s.handleDeletePlan)

				r.Get("/subscriptions", s.handleAdminListSubscriptions)
				r.Post("/subscriptions/{id}/cancel", s.handleAdminCancelSubscription)
				r.Post("/subscriptions/{id}/extend", s.handleAdminExtendSubscription)

				r.Get("/payments", s.handleAdminListPayments)

				// Chart data (Task 1)
				r.Get("/stats/chart", s.handleGetChartData)

				// Bulk operations (Task 3)
				r.Post("/users/bulk", s.handleBulkUsers)
				r.Post("/tunnels/bulk-close", s.handleBulkCloseTunnels)

				// Settings and system info (Task 4)
				r.Get("/settings", s.handleGetSettings)
				r.Get("/settings/system-info", s.handleGetSystemInfo)

				// Invite codes (Task 5)
				r.Get("/invite-codes", s.handleListInviteCodes)
				r.Post("/invite-codes", s.handleCreateInviteCode)
				r.Delete("/invite-codes/{id}", s.handleDeleteInviteCode)

				// Edge node management
				r.Route("/nodes", func(r chi.Router) {
					r.Get("/", s.handleListNodes)
					r.Post("/{id}/approve", s.handleApproveNode)
					r.Post("/{id}/disable", s.handleDisableNode)
					r.Delete("/{id}", s.handleDeleteNode)
				})
			})
		})
	})

	// Frontend is served separately (nginx/CDN); API returns 404 for non-API routes
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	})

	s.router = r
}


// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	// Empty bind = listen on all interfaces (legacy). In production it should
	// be "127.0.0.1" so external clients can only reach the API through nginx.
	addr := fmt.Sprintf("%s:%d", s.cfg.Web.Bind, s.cfg.Web.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.log.Info().Str("addr", addr).Msg("Starting API server")

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	}
}

// Shutdown gracefully stops the API server
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info().Msg("Stopping API server")
	close(s.shutdownCh)
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Router returns the chi router for embedding
func (s *Server) Router() chi.Router {
	return s.router
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			s.log.Debug().
				Str("request_id", middleware.GetReqID(r.Context())).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Dur("duration", time.Since(start)).
				Msg("HTTP request")
		}()

		next.ServeHTTP(ww, r)
	})
}

// Helper functions for JSON responses

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			s.log.Error().Err(err).Msg("failed to encode JSON response")
		}
	}
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, dto.ErrorResponse{Error: message})
}

func (s *Server) respondErrorWithCode(w http.ResponseWriter, status int, code, message string) {
	s.respondJSON(w, status, dto.ErrorResponse{Error: message, Code: code})
}

func (s *Server) decodeJSON(r *http.Request, v interface{}) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB limit
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, dto.HealthResponse{
		Status:    "ok",
		Version:   s.version,
		Timestamp: time.Now().Unix(),
	})
}
