package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/mephistofox/fxtunnel/internal/api"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/database"
	"github.com/mephistofox/fxtunnel/internal/email"
	"github.com/mephistofox/fxtunnel/internal/exchange"
	"github.com/mephistofox/fxtunnel/internal/scheduler"
	"github.com/mephistofox/fxtunnel/internal/server"
	fxtls "github.com/mephistofox/fxtunnel/internal/tls"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

var (
	configFile string
	logLevel   string
	logFormat  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "fxtunnel-server",
		Short: "fxTunnel Server - Self-hosted reverse tunneling",
		Long: `fxTunnel Server provides a self-hosted reverse tunneling solution
It allows clients to expose local services through HTTP subdomains,
TCP ports, or UDP ports.

GitHub: https://github.com/mephistofox/fxtunnel
Website: https://mfdev.ru`,
		RunE: run,
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.Flags().StringVar(&logFormat, "log-format", "console", "Log format (console, json)")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("fxTunnel Server %s (built %s)\n", Version, BuildTime)
			fmt.Println("GitHub: https://github.com/mephistofox/fxtunnel")
			fmt.Println("Website: https://mfdev.ru")
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

	// Override log settings from config if not set via flags
	if !cmd.Flags().Changed("log-level") && cfg.Logging.Level != "" {
		log = setupLogging(cfg.Logging.Level, cfg.Logging.Format)
	}

	// Initialize database if web panel is enabled
	var db *database.Database
	var authService *auth.Service
	var apiServer *api.Server

	if cfg.Web.Enabled {
		log.Info().Str("path", cfg.Database.Path).Msg("Initializing database")

		db, err = database.New(cfg.Database.Path, log)
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

	// Initialize custom domains
	if cfg.CustomDomains.Enabled && db != nil {
		if err := srv.InitCustomDomains(); err != nil {
			log.Error().Err(err).Msg("Failed to initialize custom domains")
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
		Msg("Server started")

	// Start API server if web panel is enabled
	if cfg.Web.Enabled && authService != nil {
		tunnelProvider := &serverAdapter{srv: srv}
		var cdm api.CustomDomainManager
		if cfg.CustomDomains.Enabled {
			cdm = &customDomainAdapter{srv: srv}
		}
		srv.InspectManager().SetStore(db.Exchanges)

		apiServer = api.New(cfg, db, authService, tunnelProvider, srv.InspectManager(), cdm, log)
		apiServer.SetVersion(Version)
		apiServer.SetMinVersion(cfg.Server.MinVersion)
		apiServer.SetReplayProvider(srv.HTTPRouter())

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

		// Start expired session cleanup
		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if deleted, err := db.Sessions.DeleteExpired(); err != nil {
						log.Error().Err(err).Msg("Failed to cleanup expired sessions")
					} else if deleted > 0 {
						log.Info().Int64("deleted", deleted).Msg("Cleaned up expired sessions")
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

		// Initialize exchange rate service
		exchange.New(log)
		log.Info().Msg("Exchange rate service initialized")

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
			notifier = email.NewNotifier(emailService, db, baseURL, cfg.SMTP.From, log)
			apiServer.SetNotifier(notifier)
			log.Info().Msg("Email service initialized")
		}

		// Start subscription scheduler if payments are enabled
		if cfg.YooKassa.Enabled {
			subscriptionScheduler := scheduler.New(db, cfg, log)

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
