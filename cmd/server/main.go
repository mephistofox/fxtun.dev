package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/fxcode/fxtunnel/internal/config"
	"github.com/fxcode/fxtunnel/internal/server"
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

	// Create server
	srv := server.New(cfg, log)

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

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	// Graceful shutdown
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
