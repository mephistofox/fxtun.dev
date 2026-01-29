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
	"github.com/mephistofox/fxtunnel/internal/server"
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
similar to ngrok or serveo.net.

It allows clients to expose local services through HTTP subdomains,
TCP ports, or UDP ports.`,
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
		apiServer = api.New(cfg, db, authService, tunnelProvider, log)

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
		apiServer.Shutdown(shutdownCtx)
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
