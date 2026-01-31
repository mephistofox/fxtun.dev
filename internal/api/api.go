package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/database"
	"github.com/mephistofox/fxtunnel/internal/inspect"
	fxtls "github.com/mephistofox/fxtunnel/internal/tls"
	"github.com/mephistofox/fxtunnel/internal/web"
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
}

// ReplayProvider sends an HTTP request through a tunnel and returns the response.
type ReplayProvider interface {
	ReplayRequest(subdomain string, req *http.Request) (*http.Response, error)
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
	router               chi.Router
	httpServer     *http.Server
	log            zerolog.Logger
	baseDomain     string
	downloadsPath  string
	version        string
	deviceStore    *deviceStore
	shutdownCh     chan struct{}
}

// New creates a new API server
func New(cfg *config.ServerConfig, db *database.Database, authService *auth.Service, tunnelProvider TunnelProvider, inspectProvider InspectProvider, customDomainManager CustomDomainManager, log zerolog.Logger) *Server {
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
		deviceStore:    newDeviceStore(),
		shutdownCh:     make(chan struct{}),
	}

	go s.deviceStore.Cleanup(s.shutdownCh)

	s.setupRoutes()
	return s
}

// SetReplayProvider sets the replay provider for inspect replay feature.
func (s *Server) SetReplayProvider(rp ReplayProvider) {
	s.replayProvider = rp
}

// SetVersion sets the server version string for health endpoint.
func (s *Server) SetVersion(version string) {
	s.version = version
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.loggingMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(30 * time.Second))

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
		corsOrigins = []string{}
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
	r.Group(func(r chi.Router) {
		r.Use(auth.MiddlewareWithDB(s.authService, s.db))
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
		})

		// Downloads (public)
		r.Route("/downloads", func(r chi.Router) {
			r.Get("/", s.handleListDownloads)
			r.Get("/{platform}", s.handleDownload)
		})

		// SSE inspect stream (separate auth to support ?token= for EventSource)
		r.Route("/tunnels/{id}/inspect/stream", func(r chi.Router) {
			r.Use(s.queryTokenAuthMiddleware)
			r.Get("/", s.handleInspectStream)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(auth.MiddlewareWithDB(s.authService, s.db))

			// Auth
			r.Post("/auth/logout", s.handleLogout)
			r.Post("/auth/device/authorize", s.handleDeviceAuthorize)

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

			// Admin routes
			r.Route("/admin", func(r chi.Router) {
				r.Use(auth.AdminMiddleware)

				r.Get("/stats", s.handleGetStats)
				r.Get("/users", s.handleListUsers)
				r.Put("/users/{id}", s.handleUpdateUser)
				r.Delete("/users/{id}", s.handleDeleteUser)
				r.Get("/audit-logs", s.handleListAuditLogs)
				r.Get("/tunnels", s.handleListAllTunnels)
				r.Delete("/tunnels/{id}", s.handleAdminCloseTunnel)

				r.Get("/custom-domains", s.handleAdminListCustomDomains)
				r.Delete("/custom-domains/{id}", s.handleAdminDeleteCustomDomain)

				r.Route("/invite-codes", func(r chi.Router) {
					r.Get("/", s.handleListInviteCodes)
					r.Post("/", s.handleCreateInviteCode)
					r.Delete("/{id}", s.handleDeleteInviteCode)
				})
			})
		})
	})

	// Serve embedded web UI for all other routes
	r.Get("/*", s.serveWebUI())

	s.router = r
}

func (s *Server) queryTokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			if token := r.URL.Query().Get("token"); token != "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
		}
		// Use the same auth middleware
		auth.MiddlewareWithDB(s.authService, s.db)(next).ServeHTTP(w, r)
	})
}

// serveWebUI returns a handler that serves the embedded web UI with SPA support
func (s *Server) serveWebUI() http.HandlerFunc {
	webFS := web.GetFileSystem()
	fileServer := http.FileServer(webFS)

	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to open the file to check if it exists
		f, err := webFS.Open(path)
		if err != nil {
			// File not found, serve index.html for SPA routing
			r.URL.Path = "/"
		} else {
			stat, _ := f.Stat()
			_ = f.Close()
			if stat != nil && stat.IsDir() {
				r.URL.Path = "/"
			}
		}

		fileServer.ServeHTTP(w, r)
	}
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.cfg.Web.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
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
