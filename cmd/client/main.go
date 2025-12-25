package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/mephistofox/fxtunnel/internal/client"
	"github.com/mephistofox/fxtunnel/internal/config"
)

const defaultControlPort = "4443"

var (
	Version   = "dev"
	BuildTime = "unknown"
)

var (
	configFile string
	serverAddr string
	token      string
	logLevel   string
	logFormat  string

	// Quick tunnel flags
	remotePort int
	subdomain  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "fxtunnel",
		Short: "fxTunnel Client - Expose local services to the internet",
		Long: `fxTunnel Client connects to a fxTunnel Server and creates tunnels
to expose local services to the internet.

Examples:
  # Expose local HTTP server on port 3000
  fxtunnel http 3000

  # Expose local HTTP server with custom subdomain
  fxtunnel http 3000 --subdomain myapp

  # Expose local TCP port
  fxtunnel tcp 22

  # Use config file
  fxtunnel --config client.yaml

For GUI mode, use fxtunnel-gui binary.`,
		RunE: runConfig,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().StringVarP(&serverAddr, "server", "s", "", "Server address (host or host:port, default port: 4443)")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "Authentication token")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "console", "Log format (console, json)")

	// HTTP tunnel command
	httpCmd := &cobra.Command{
		Use:   "http <local_port>",
		Short: "Create an HTTP tunnel",
		Args:  cobra.ExactArgs(1),
		RunE:  runHTTP,
	}
	httpCmd.Flags().StringVarP(&subdomain, "subdomain", "d", "", "Subdomain to use (auto-generated if not set)")
	rootCmd.AddCommand(httpCmd)

	// TCP tunnel command
	tcpCmd := &cobra.Command{
		Use:   "tcp <local_port>",
		Short: "Create a TCP tunnel",
		Args:  cobra.ExactArgs(1),
		RunE:  runTCP,
	}
	tcpCmd.Flags().IntVarP(&remotePort, "remote-port", "r", 0, "Remote port (auto-assigned if 0)")
	rootCmd.AddCommand(tcpCmd)

	// UDP tunnel command
	udpCmd := &cobra.Command{
		Use:   "udp <local_port>",
		Short: "Create a UDP tunnel",
		Args:  cobra.ExactArgs(1),
		RunE:  runUDP,
	}
	udpCmd.Flags().IntVarP(&remotePort, "remote-port", "r", 0, "Remote port (auto-assigned if 0)")
	rootCmd.AddCommand(udpCmd)

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("fxTunnel Client %s (built %s)\n", Version, BuildTime)
		},
	}
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runConfig(cmd *cobra.Command, args []string) error {
	log := setupLogging(logLevel, logFormat)

	// Load config
	cfg, err := config.LoadClientConfig(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Override from flags
	if serverAddr != "" {
		cfg.Server.Address = serverAddr
	}
	if token != "" {
		cfg.Server.Token = token
	}

	// Normalize server address (add default port if missing)
	cfg.Server.Address = normalizeServerAddr(cfg.Server.Address)

	if len(cfg.Tunnels) == 0 {
		cmd.Help()
		return nil
	}

	return runClient(cfg, log)
}

func runHTTP(cmd *cobra.Command, args []string) error {
	log := setupLogging(logLevel, logFormat)

	port, err := parsePort(args[0])
	if err != nil {
		return err
	}

	cfg := buildConfig(config.TunnelConfig{
		Name:      fmt.Sprintf("http-%d", port),
		Type:      "http",
		LocalPort: port,
		Subdomain: subdomain,
	})

	return runClient(cfg, log)
}

func runTCP(cmd *cobra.Command, args []string) error {
	log := setupLogging(logLevel, logFormat)

	port, err := parsePort(args[0])
	if err != nil {
		return err
	}

	cfg := buildConfig(config.TunnelConfig{
		Name:       fmt.Sprintf("tcp-%d", port),
		Type:       "tcp",
		LocalPort:  port,
		RemotePort: remotePort,
	})

	return runClient(cfg, log)
}

func runUDP(cmd *cobra.Command, args []string) error {
	log := setupLogging(logLevel, logFormat)

	port, err := parsePort(args[0])
	if err != nil {
		return err
	}

	cfg := buildConfig(config.TunnelConfig{
		Name:       fmt.Sprintf("udp-%d", port),
		Type:       "udp",
		LocalPort:  port,
		RemotePort: remotePort,
	})

	return runClient(cfg, log)
}

func buildConfig(tunnel config.TunnelConfig) *config.ClientConfig {
	cfg := &config.ClientConfig{
		Server: config.ClientServerSettings{
			Address: normalizeServerAddr(serverAddr),
			Token:   token,
		},
		Tunnels: []config.TunnelConfig{tunnel},
		Reconnect: config.ReconnectSettings{
			Enabled:     true,
			Interval:    5 * time.Second,
			MaxAttempts: 0,
		},
	}

	return cfg
}

// normalizeServerAddr adds default port if not specified
func normalizeServerAddr(addr string) string {
	if addr == "" {
		return "localhost:" + defaultControlPort
	}
	// Check if port is already specified
	if !strings.Contains(addr, ":") {
		return addr + ":" + defaultControlPort
	}
	return addr
}

func parsePort(s string) (int, error) {
	var port int
	_, err := fmt.Sscanf(s, "%d", &port)
	if err != nil {
		return 0, fmt.Errorf("invalid port: %s", s)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port out of range: %d", port)
	}
	return port, nil
}

func runClient(cfg *config.ClientConfig, log zerolog.Logger) error {
	log.Info().
		Str("version", Version).
		Str("server", cfg.Server.Address).
		Msg("Starting fxTunnel Client")

	// Create client
	c := client.New(cfg, log)

	// Connect
	if err := c.Connect(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to server")
	}

	// Print tunnel info
	for _, t := range c.GetTunnels() {
		if t.URL != "" {
			log.Info().
				Str("name", t.Config.Name).
				Str("url", t.URL).
				Int("local_port", t.Config.LocalPort).
				Msg("Tunnel active")
		} else {
			log.Info().
				Str("name", t.Config.Name).
				Str("addr", t.RemoteAddr).
				Int("local_port", t.Config.LocalPort).
				Msg("Tunnel active")
		}
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	c.Close()
	return nil
}

func setupLogging(level, format string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

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
