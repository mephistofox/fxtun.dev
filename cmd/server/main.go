package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/mephistofox/fxtunnel/internal/server/api"
	"github.com/mephistofox/fxtunnel/internal/server/auth"
	"github.com/mephistofox/fxtunnel/internal/config"
	server "github.com/mephistofox/fxtunnel/internal/server/core"
	"github.com/mephistofox/fxtunnel/internal/server/database"
	fxdns "github.com/mephistofox/fxtunnel/internal/server/dns"
	"github.com/mephistofox/fxtunnel/internal/server/email"
	"github.com/mephistofox/fxtunnel/internal/server/exchange"
	"github.com/mephistofox/fxtunnel/internal/server/geoip"
	"github.com/mephistofox/fxtunnel/internal/server/hub"
	"github.com/mephistofox/fxtunnel/internal/server/payment"
	fxredis "github.com/mephistofox/fxtunnel/internal/server/redis"
	"github.com/mephistofox/fxtunnel/internal/server/store"
	"github.com/mephistofox/fxtunnel/internal/server/scheduler"
	"github.com/mephistofox/fxtunnel/internal/server/telegram"
	fxtls "github.com/mephistofox/fxtunnel/internal/server/tls"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

var (
	configFile string
	logLevel   string
	logFormat  string
	serverMode string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "fxtunnel-server",
		Short: "fxTunnel Server - Self-hosted reverse tunneling",
		Long: `fxTunnel Server provides a self-hosted reverse tunneling solution
It allows clients to expose local services through HTTP subdomains,
TCP ports, or UDP ports.

GitHub: https://github.com/mephistofox/fxtun.dev
Website: https://fxtun.dev`,
		RunE: run,
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.Flags().StringVar(&logFormat, "log-format", "console", "Log format (console, json)")
	rootCmd.Flags().StringVar(&serverMode, "mode", "", "Server mode: standalone, hub, node")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("fxTunnel Server %s (built %s)\n", Version, BuildTime)
			fmt.Println("GitHub: https://github.com/mephistofox/fxtun.dev")
			fmt.Println("Website: https://fxtun.dev")
		},
	}
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Setup logging
	log := setupLogging(logLevel, logFormat)

	log.Info().
		Str("version", Version).
		Str("build_time", BuildTime).
		Msg("Starting fxTunnel Server")

	// Load configuration
	cfg, err := config.LoadServerConfig(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Override mode from CLI flag
	if serverMode != "" {
		cfg.Mode = config.ServerMode(serverMode)
	}

	// Override log settings from config if not set via flags
	if !cmd.Flags().Changed("log-level") && cfg.Logging.Level != "" {
		log = setupLogging(cfg.Logging.Level, cfg.Logging.Format)
	}

	log.Info().
		Str("mode", string(cfg.EffectiveMode())).
		Msg("Server mode")

	// Initialize database if web panel is enabled
	var db *database.Database
	var authService *auth.Service
	var apiServer *api.Server

	if cfg.Web.Enabled {
		log.Info().Str("dsn", cfg.Database.DSN).Msg("Initializing database")

		db, err = database.New(cfg.Database.DSN, log)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize database")
		}
		defer db.Close()

		// Parse token TTLs
		accessTTL, err := time.ParseDuration(cfg.Auth.AccessTokenTTL)
		if err != nil {
			accessTTL = 15 * time.Minute
		}
		refreshTTL, err := time.ParseDuration(cfg.Auth.RefreshTokenTTL)
		if err != nil {
			refreshTTL = 7 * 24 * time.Hour
		}

		totpKey := []byte(cfg.TOTP.EncryptionKey)

		// Derive TLS encryption key from TOTP key for encrypting TLS private keys at rest
		if len(totpKey) >= 32 {
			tlsEncKey := deriveTLSEncryptionKey(totpKey)
			db.TLSCerts.SetEncryptionKey(tlsEncKey)
		}

		// Create auth service
		authService = auth.NewService(
			db,
			cfg.Auth.JWTSecret,
			accessTTL,
			refreshTTL,
			cfg.TOTP.Issuer,
			totpKey,
			cfg.Auth.MaxDomains,
			log,
		)

		log.Info().Msg("Database and auth service initialized")
	}

	// Initialize Redis if enabled
	var redisClient *fxredis.Client
	if cfg.Redis.Enabled {
		redisClient, err = fxredis.New(cfg.Redis, log)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to connect to Redis")
		}
		defer redisClient.Close()

		// Override session store with Redis
		if authService != nil {
			authService.SetSessionStore(fxredis.NewSessionStore(redisClient))
			log.Info().Msg("Redis session store enabled")
		}
	}

	// Initialize Telegram notifications
	var telegramNotifier *telegram.AdminNotifier
	if cfg.Telegram.Enabled {
		tgBot := telegram.NewBot(cfg.Telegram.BotToken)
		telegramNotifier = telegram.NewAdminNotifier(tgBot, cfg.Telegram.ChatID)
		telegramNotifier.SetLogger(log)
		log.Info().Msg("Telegram admin notifications enabled")
	}

	// Create server
	srv := server.New(cfg, log)

	// Set database if initialized
	if db != nil {
		srv.SetDatabase(db)
	}

	// Set auth service for JWT validation
	if authService != nil {
		srv.SetAuthService(authService)
	}

	// Set tunnel registry and TLS cache if Redis enabled
	var tunnelRegistry *fxredis.TunnelRegistry
	var nodeRegistry *fxredis.NodeRegistry
	if redisClient != nil {
		// ServerID determines how other nodes address this server for HTTP proxy
		hostname, _ := os.Hostname()
		serverID := hostname
		if cfg.EffectiveMode() == config.ModeNode && cfg.Node.HTTPAddr != "" {
			serverID = cfg.Node.HTTPAddr
		}
		tunnelRegistry = fxredis.NewTunnelRegistry(redisClient, serverID)
		srv.SetTunnelRegistry(tunnelRegistry)
		srv.SetLocalNodeID(serverID)
		log.Info().Str("server_id", serverID).Msg("Redis tunnel registry enabled")

		// Set node registry for hub and node modes
		if cfg.EffectiveMode() == config.ModeHub || cfg.EffectiveMode() == config.ModeNode {
			nodeRegistry = fxredis.NewNodeRegistry(redisClient)
			srv.SetNodeRegistry(nodeRegistry)
			log.Info().Msg("Redis node registry enabled")
		}
	}

	// Set server mode
	srv.SetMode(cfg.EffectiveMode())

	// Initialize GeoIP for region-based edge node selection
	if cfg.GeoIP.Enabled && cfg.GeoIP.Database != "" {
		geo, err := geoip.New(cfg.GeoIP.Database)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to load GeoIP database, using least-loaded selection")
		} else {
			srv.SetGeoIP(geo)
			defer geo.Close()
			log.Info().Str("db", cfg.GeoIP.Database).Msg("GeoIP database loaded")
		}
	}

	// Set Telegram notifier on tunnel server
	if telegramNotifier != nil {
		srv.SetTelegramNotifier(telegramNotifier)
	}

	// Initialize custom domains
	if cfg.CustomDomains.Enabled && db != nil {
		if err := srv.InitCustomDomains(); err != nil {
			log.Error().Err(err).Msg("Failed to initialize custom domains")
		}
		// Set Redis TLS cache if available
		if redisClient != nil {
			if cm := srv.CertManager(); cm != nil {
				cm.SetRedisCache(fxredis.NewTLSCache(redisClient))
				log.Info().Msg("Redis TLS certificate cache enabled")
			}
		}
	}

	// Node mode: register with hub and fetch TLS cert BEFORE starting server
	var hubClient *hub.Client
	if cfg.EffectiveMode() == config.ModeNode {
		hubClient = hub.NewClient(cfg.Node.HubURL, cfg.Node.HubToken, log)
		nodeID, err := hubClient.Register(cfg.Node.Name, cfg.Node.Region, cfg.Node.PublicAddr, cfg.Node.HTTPAddr, Version)
		if err != nil {
			log.Error().Err(err).Msg("Failed to register with hub (will retry via heartbeat)")
		} else {
			log.Info().Str("node_id", nodeID).Msg("Registered with hub")
		}

		// Fetch TLS cert from hub if cert files don't exist (blocks until obtained)
		if cfg.TLS.Enabled {
			if _, statErr := os.Stat(cfg.TLS.CertFile); os.IsNotExist(statErr) {
				log.Info().Msg("TLS cert not found locally, requesting from hub...")
				for {
					tlsCert, err := hubClient.FetchTLSCert()
					if err != nil {
						log.Warn().Err(err).Msg("Failed to fetch TLS cert (node may be pending approval, retrying in 1m)")
						time.Sleep(1 * time.Minute)
						continue
					}
					certDir := filepath.Dir(cfg.TLS.CertFile)
					_ = os.MkdirAll(certDir, 0700)
					if err := os.WriteFile(cfg.TLS.CertFile, []byte(tlsCert.CertPEM), 0600); err != nil {
						log.Fatal().Err(err).Msg("Failed to write cert file")
					}
					if err := os.WriteFile(cfg.TLS.KeyFile, []byte(tlsCert.KeyPEM), 0600); err != nil {
						log.Fatal().Err(err).Msg("Failed to write key file")
					}
					log.Info().Msg("TLS cert fetched from hub and saved")
					break
				}
			}
		}
	}

	// Start server
	if err := srv.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}

	log.Info().
		Int("control_port", cfg.Server.ControlPort).
		Int("http_port", cfg.Server.HTTPPort).
		Str("domain", cfg.Domain.Base).
		Bool("auth_enabled", cfg.Auth.Enabled).
		Str("mode", string(cfg.EffectiveMode())).
		Msg("Server started")

	// Authoritative DNS server (optional). Resolves static records from a YAML
	// zone file plus dynamic tunnel subdomains from the Redis tunnel registry.
	var dnsSrv *fxdns.Server
	if cfg.DNS.Enabled && cfg.DNS.ZoneFile != "" {
		var dnsTunnels fxdns.TunnelLookup
		var dnsNodes fxdns.NodeLookup
		if tunnelRegistry != nil {
			dnsTunnels = tunnelRegistry
		}
		if nodeRegistry != nil {
			dnsNodes = nodeRegistry
		}

		dnsSrv, err = fxdns.New(fxdns.Config{
			Enabled:  true,
			Listen:   cfg.DNS.Listen,
			ZoneFile: cfg.DNS.ZoneFile,
		}, dnsTunnels, dnsNodes, log)
		if err != nil {
			log.Error().Err(err).Msg("Failed to init DNS server")
		} else if err := dnsSrv.Start(); err != nil {
			log.Error().Err(err).Msg("Failed to start DNS server")
			dnsSrv = nil
		}
	}

	// Node mode: set hub client and start heartbeat AFTER server started
	if cfg.EffectiveMode() == config.ModeNode && hubClient != nil {
		srv.SetHubClient(&hubAuthAdapter{client: hubClient})

		heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
		defer heartbeatCancel()
		go hubClient.StartHeartbeatLoop(heartbeatCtx, 30*time.Second, func() (int, int) {
			stats := srv.GetStats()
			return stats.ActiveTunnels, stats.ActiveClients
		})
	}

	// Hub mode: register self as a node so hub also serves tunnels.
	// Uses node section config (name/region/public_addr) if set, otherwise defaults.
	if cfg.EffectiveMode() == config.ModeHub && nodeRegistry != nil {
		hubName := cfg.Node.Name
		if hubName == "" {
			hubName = "hub"
		}
		hubRegion := cfg.Node.Region
		if hubRegion == "" {
			hubRegion = "default"
		}
		hubPublicAddr := cfg.Node.PublicAddr
		if hubPublicAddr == "" {
			hubPublicAddr = fmt.Sprintf("%s:%d", srv.NodePublicHost(), cfg.Server.ControlPort)
		}
		hubHTTPAddr := cfg.Node.HTTPAddr
		if hubHTTPAddr == "" {
			hubHTTPAddr = fmt.Sprintf("%s:%d", srv.NodePublicHost(), cfg.Server.HTTPPort)
		}

		hubNodeID, _ := os.Hostname()
		if hubNodeID == "" {
			hubNodeID = "hub"
		}

		hubEntry := store.NodeEntry{
			NodeID:     hubNodeID,
			Name:       hubName,
			Region:     hubRegion,
			PublicAddr: hubPublicAddr,
			HTTPAddr:   hubHTTPAddr,
			Status:     "active",
		}
		if err := nodeRegistry.RegisterNode(hubEntry); err != nil {
			log.Warn().Err(err).Msg("Failed to register hub as node")
		} else {
			srv.SetLocalNodeID(hubNodeID)
			log.Info().
				Str("node_id", hubNodeID).
				Str("name", hubName).
				Str("region", hubRegion).
				Msg("Hub registered as node in active pool")

			// Hub heartbeat to keep itself in active set
			hubHBCtx, hubHBCancel := context.WithCancel(context.Background())
			defer hubHBCancel()
			go func() {
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-hubHBCtx.Done():
						return
					case <-ticker.C:
						stats := srv.GetStats()
						_ = nodeRegistry.HeartbeatNode(hubNodeID, stats.ActiveTunnels, stats.ActiveClients)
					}
				}
			}()
		}
	}

	// Start API server if web panel is enabled
	if cfg.Web.Enabled && authService != nil {
		tunnelProvider := &serverAdapter{srv: srv}
		var cdm api.CustomDomainManager
		if cfg.CustomDomains.Enabled {
			cdm = &customDomainAdapter{srv: srv}
		}
		srv.InspectManager().SetStore(db.Exchanges)

		var apiOpts []api.Option
		if redisClient != nil {
			apiOpts = append(apiOpts,
				api.WithDeviceStore(fxredis.NewDeviceStore(redisClient)),
				api.WithOAuthStore(fxredis.NewOAuthStore(redisClient)),
				api.WithIPBanStore(fxredis.NewIPBanStore(redisClient)),
			)
			// Add node registry for hub mode admin endpoints
			if cfg.EffectiveMode() == config.ModeHub {
				apiOpts = append(apiOpts, api.WithNodeRegistry(fxredis.NewNodeRegistry(redisClient)))
			}
		}

		apiServer = api.New(cfg, db, authService, tunnelProvider, srv.InspectManager(), cdm, log, apiOpts...)
		apiServer.SetVersion(Version)
		apiServer.SetMinVersion(cfg.Server.MinVersion)
		apiServer.SetReplayProvider(srv.HTTPRouter())

		if telegramNotifier != nil {
			apiServer.SetTelegramNotifier(telegramNotifier)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			if err := apiServer.Start(ctx); err != nil {
				log.Error().Err(err).Msg("API server error")
			}
		}()

		log.Info().
			Int("port", cfg.Web.Port).
			Msg("Web panel API started")

		// Start periodic cleanup
		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Session cleanup only needed when Redis is not handling TTL
					if redisClient == nil {
						if deleted, err := db.Sessions.DeleteExpired(); err != nil {
							log.Error().Err(err).Msg("Failed to cleanup expired sessions")
						} else if deleted > 0 {
							log.Info().Int64("deleted", deleted).Msg("Cleaned up expired sessions")
						}
					}
					// Cleanup old inspect exchanges (24h TTL)
					if deleted, err := db.Exchanges.DeleteOlderThan(time.Now().Add(-24 * time.Hour)); err != nil {
						log.Error().Err(err).Msg("Failed to cleanup old inspect exchanges")
					} else if deleted > 0 {
						log.Info().Int64("deleted", deleted).Msg("Cleaned up old inspect exchanges")
					}
				}
			}
		}()

		// Start stale-node cleanup for hub mode
		if cfg.EffectiveMode() == config.ModeHub && redisClient != nil {
			nodeReg := fxredis.NewNodeRegistry(redisClient)
			go func() {
				ticker := time.NewTicker(60 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						stale, err := db.EdgeNodes.ListStaleNodes(3 * time.Minute)
						if err != nil {
							log.Error().Err(err).Msg("Failed to list stale nodes")
							continue
						}
						for _, node := range stale {
							log.Warn().Str("node_id", node.NodeID).Str("name", node.Name).Msg("Disabling stale edge node")
							_ = db.EdgeNodes.UpdateStatus(node.ID, "disabled", 0)
							_ = nodeReg.UnregisterNode(node.NodeID)
						}
					}
				}
			}()
		}

		// Initialize exchange rate from config
		exchange.Init(cfg.ExchangeRate)
		log.Info().Float64("rate", cfg.ExchangeRate).Msg("Exchange rate initialized")

		// Initialize email service
		var emailService *email.Service
		var notifier *email.Notifier
		if cfg.SMTP.Enabled {
			emailService = email.New(&cfg.SMTP, log)
			baseURL := cfg.SMTP.BaseURL
			if baseURL == "" {
				baseURL = fmt.Sprintf("https://%s", cfg.Domain.Base)
				if cfg.Web.Port != 443 && cfg.Web.Port != 80 {
					baseURL = fmt.Sprintf("http://%s:%d", cfg.Domain.Base, cfg.Web.Port)
				}
			}
			notifier = email.NewNotifier(emailService, db, baseURL, cfg.SMTP.BaseURLEN, cfg.SMTP.From, log)
			apiServer.SetNotifier(notifier)
			log.Info().Msg("Email service initialized")
		}

		// Setup payment providers
		providers := payment.NewRegistry()
		if cfg.YooKassa.Enabled {
			yookassa := payment.NewYooKassa(payment.YooKassaConfig{
				ShopID:    cfg.YooKassa.ShopID,
				SecretKey: cfg.YooKassa.SecretKey,
				TestMode:  cfg.YooKassa.TestMode,
				ReturnURL: cfg.YooKassa.ReturnURL,
			})
			providers.Register(yookassa)
		}
		if cfg.Creem.Enabled {
			creemProvider := payment.NewCreem(payment.CreemConfig{
				APIKey:        cfg.Creem.APIKey,
				WebhookSecret: cfg.Creem.WebhookSecret,
				TestMode:      cfg.Creem.TestMode,
				SuccessURL:    cfg.Creem.SuccessURL,
				CancelURL:     cfg.Creem.CancelURL,
			})
			providers.Register(creemProvider)
		}
		apiServer.SetPaymentProviders(providers)

		// Start subscription scheduler if payments are enabled
		if cfg.YooKassa.Enabled || cfg.Creem.Enabled {
			subscriptionScheduler := scheduler.New(db, cfg, providers, log)

			// Register event handler for logging
			subscriptionScheduler.OnEvent(func(event scheduler.Event) {
				switch event.Type {
				case scheduler.EventSubscriptionExpiring:
					log.Info().
						Int64("user_id", event.UserID).
						Int("days_left", event.DaysLeft).
						Msg("Subscription expiring soon")
				case scheduler.EventSubscriptionExpired:
					log.Info().
						Int64("user_id", event.UserID).
						Msg("Subscription expired")
				case scheduler.EventSubscriptionRenewed:
					log.Info().
						Int64("user_id", event.UserID).
						Msg("Subscription renewed")
				case scheduler.EventSubscriptionRenewFailed:
					log.Error().
						Int64("user_id", event.UserID).
						Err(event.Error).
						Msg("Subscription renewal failed")
				case scheduler.EventPlanChanged:
					log.Info().
						Int64("user_id", event.UserID).
						Int64("plan_id", event.Plan.ID).
						Msg("Plan changed")
				}
			})

			// Register email notifier if available
			if notifier != nil {
				subscriptionScheduler.OnEvent(notifier.HandleSchedulerEvent)
				log.Info().Msg("Email notifications enabled for scheduler")
			}

			go subscriptionScheduler.Start(ctx)
			log.Info().Msg("Subscription scheduler started")
		}
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	// Graceful shutdown
	exchange.Stop()

	if dnsSrv != nil {
		dnsSrv.Stop()
	}

	if apiServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = apiServer.Shutdown(shutdownCtx)
	}

	if err := srv.Stop(); err != nil {
		log.Error().Err(err).Msg("Error during shutdown")
		return err
	}

	return nil
}

func setupLogging(level, format string) zerolog.Logger {
	// Parse level
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	// Setup output
	var log zerolog.Logger
	if format == "json" {
		log = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		log = zerolog.New(output).With().Timestamp().Logger()
	}

	return log
}

// serverAdapter wraps *server.Server to implement api.TunnelProvider
type serverAdapter struct {
	srv *server.Server
}

func (a *serverAdapter) GetTunnelsByUserID(userID int64) []api.TunnelInfo {
	serverTunnels := a.srv.GetTunnelsByUserID(userID)
	result := make([]api.TunnelInfo, len(serverTunnels))
	for i, t := range serverTunnels {
		result[i] = api.TunnelInfo{
			ID:         t.ID,
			Type:       t.Type,
			Name:       t.Name,
			Subdomain:  t.Subdomain,
			RemotePort: t.RemotePort,
			LocalPort:  t.LocalPort,
			ClientID:   t.ClientID,
			UserID:     t.UserID,
			CreatedAt:  t.CreatedAt,
		}
	}
	return result
}

func (a *serverAdapter) CloseTunnelByID(tunnelID string, userID int64) error {
	return a.srv.CloseTunnelByID(tunnelID, userID)
}

func (a *serverAdapter) GetStats() api.Stats {
	s := a.srv.GetStats()
	return api.Stats{
		ActiveClients: s.ActiveClients,
		ActiveTunnels: s.ActiveTunnels,
		HTTPTunnels:   s.HTTPTunnels,
		TCPTunnels:    s.TCPTunnels,
		UDPTunnels:    s.UDPTunnels,
	}
}

func (a *serverAdapter) GetAllTunnels() []api.TunnelInfo {
	serverTunnels := a.srv.GetAllTunnels()
	result := make([]api.TunnelInfo, len(serverTunnels))
	for i, t := range serverTunnels {
		result[i] = api.TunnelInfo{
			ID:         t.ID,
			Type:       t.Type,
			Name:       t.Name,
			Subdomain:  t.Subdomain,
			RemotePort: t.RemotePort,
			LocalPort:  t.LocalPort,
			ClientID:   t.ClientID,
			UserID:     t.UserID,
			CreatedAt:  t.CreatedAt,
		}
	}
	return result
}

func (a *serverAdapter) AdminCloseTunnel(tunnelID string) error {
	return a.srv.AdminCloseTunnel(tunnelID)
}

// customDomainAdapter wraps *server.Server to implement api.CustomDomainManager
type customDomainAdapter struct {
	srv *server.Server
}

func (a *customDomainAdapter) AddCustomDomain(d *database.CustomDomain) {
	a.srv.AddCustomDomain(d)
}

func (a *customDomainAdapter) RemoveCustomDomain(domain string) {
	a.srv.RemoveCustomDomain(domain)
}

func (a *customDomainAdapter) CertManager() *fxtls.CertManager {
	return a.srv.CertManager()
}

// deriveTLSEncryptionKey derives a separate AES-256 key for TLS private key encryption
// from the TOTP encryption key using SHA-256 with a domain separation prefix.
func deriveTLSEncryptionKey(totpKey []byte) []byte {
	h := sha256.New()
	h.Write([]byte("fxtunnel-tls-key-encryption:"))
	h.Write(totpKey)
	return h.Sum(nil) // 32 bytes
}

// hubAuthAdapter adapts hub.Client to server.HubAuthVerifier interface.
type hubAuthAdapter struct {
	client *hub.Client
}

func (a *hubAuthAdapter) VerifyClientToken(token string) (*server.HubAuthInfo, error) {
	info, err := a.client.VerifyClientToken(token)
	if err != nil {
		return nil, err
	}
	return &server.HubAuthInfo{
		Valid:            info.Valid,
		UserID:           info.UserID,
		MaxTunnels:       info.MaxTunnels,
		MaxDataSessions:  info.MaxDataSessions,
		IsAdmin:          info.IsAdmin,
		InspectorEnabled: info.InspectorEnabled,
	}, nil
}
