package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/mephistofox/fxtunnel/internal/client"
	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/keyring"
)

const defaultControlPort = "4443"

var (
	Version          = "dev"
	BuildTime        = "unknown"
	DefaultServerURL = "https://mfdev.ru"
)

var (
	configFile string
	serverAddr string
	token      string
	logLevel   string
	logFormat  string

	// Quick tunnel flags
	remotePort int
	domain     string
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

  # Expose local HTTP server with custom domain
  fxtunnel http 3000 --domain myapp

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
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "console", "Log format (console, json)")

	// HTTP tunnel command
	httpCmd := &cobra.Command{
		Use:   "http <local_port>",
		Short: "Create an HTTP tunnel",
		Args:  cobra.ExactArgs(1),
		RunE:  runHTTP,
	}
	httpCmd.Flags().StringVarP(&domain, "domain", "d", "", "Subdomain to use (auto-generated if not set)")
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

	// Login command
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Save authentication token",
		Long: `Save your API token to the system keyring for future use.
Use -t to provide token directly, or enter it interactively.`,
		RunE: runLogin,
	}
	rootCmd.AddCommand(loginCmd)

	// Logout command
	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove saved credentials",
		RunE:  runLogout,
	}
	rootCmd.AddCommand(logoutCmd)

	// Init command
	rootCmd.AddCommand(newInitCmd())

	// Domains command
	rootCmd.AddCommand(newDomainsCmd())

	// Update command
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Check for updates and self-update",
		RunE:  runUpdate,
	}
	rootCmd.AddCommand(updateCmd)

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("fxTunnel Client %s (built %s)\n", Version, BuildTime)
			fmt.Println("GitHub: https://github.com/mephistofox/fxtunnel")
			fmt.Println("Website: https://mfdev.ru")
		},
	}
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runConfig(cmd *cobra.Command, args []string) error {
	resolveCredentials()
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
		_ = cmd.Help()
		return nil
	}

	return runClient(cfg, log)
}

func runHTTP(cmd *cobra.Command, args []string) error {
	resolveCredentials()
	log := setupLogging(logLevel, logFormat)

	port, err := parsePort(args[0])
	if err != nil {
		return err
	}

	cfg := buildConfig(config.TunnelConfig{
		Name:      fmt.Sprintf("http-%d", port),
		Type:      "http",
		LocalPort: port,
		Subdomain: domain,
	})

	return runClient(cfg, log)
}

func runTCP(cmd *cobra.Command, args []string) error {
	resolveCredentials()
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
	resolveCredentials()
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

func resolveCredentials() {
	if token == "" || serverAddr == "" {
		kr := keyring.New()
		if creds, err := kr.LoadCredentials(); err == nil {
			if token == "" && creds.Token != "" {
				token = creds.Token
			}
			if serverAddr == "" && creds.ServerAddress != "" {
				serverAddr = creds.ServerAddress
			}
		}
	}
}

func runLogin(cmd *cobra.Command, args []string) error {
	if token != "" {
		return saveToken(token)
	}

	fmt.Println("How would you like to authenticate?")
	fmt.Println("  1) Enter API token manually")
	fmt.Println("  2) Log in with browser")
	fmt.Print("Choice [1/2] (default: 2): ")

	scanner := bufio.NewScanner(os.Stdin)
	choice := ""
	if scanner.Scan() {
		choice = strings.TrimSpace(scanner.Text())
	}

	switch choice {
	case "1":
		return loginWithToken(scanner)
	default:
		return loginWithBrowser()
	}
}

func loginWithToken(scanner *bufio.Scanner) error {
	fmt.Print("Enter your API token: ")
	t := ""
	if scanner.Scan() {
		t = strings.TrimSpace(scanner.Text())
	}
	if t == "" {
		return fmt.Errorf("token cannot be empty")
	}
	return saveToken(t)
}

func loginWithBrowser() error {
	webURL := resolveWebURL()

	// Request device code
	apiURL := webURL + "/api/auth/device/code"
	resp, err := http.Post(apiURL, "application/json", nil) //nolint:gosec // URL built from user-configured server
	if err != nil {
		return fmt.Errorf("failed to start device flow: %w\nTry: fxtunnel login -t <token>", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned %d — device flow may not be supported\nTry: fxtunnel login -t <token>", resp.StatusCode)
	}

	var deviceResp struct {
		SessionID string `json:"session_id"`
		AuthURL   string `json:"auth_url"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return fmt.Errorf("invalid server response: %w", err)
	}

	fmt.Printf("\nOpen this URL in your browser to authenticate:\n\n  %s\n\n", deviceResp.AuthURL)
	fmt.Println("Waiting for authorization...")

	_ = openBrowser(deviceResp.AuthURL)

	// Poll for token
	pollURL := webURL + "/api/auth/device/token?session=" + deviceResp.SessionID
	deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)

		pollResp, err := http.Get(pollURL) //nolint:gosec // URL built from user-configured server
		if err != nil {
			continue
		}

		var result struct {
			Status string `json:"status"`
			Token  string `json:"token"`
		}
		_ = json.NewDecoder(pollResp.Body).Decode(&result)
		pollResp.Body.Close()

		switch result.Status {
		case "authorized":
			fmt.Println("Authorized!")
			kr := keyring.New()
			creds := keyring.Credentials{
				Token:         result.Token,
				AuthMethod:    "token",
				ServerAddress: serverAddr,
			}
			if err := kr.SaveCredentials(creds); err != nil {
				return fmt.Errorf("failed to save credentials: %w", err)
			}
			fmt.Println("Token saved. You can now use fxtunnel without --token flag.")
			return nil
		case "expired":
			return fmt.Errorf("session expired — please try again")
		}
	}

	return fmt.Errorf("authorization timed out — please try again")
}

func resolveWebURL() string {
	addr := serverAddr
	if addr == "" {
		kr := keyring.New()
		if creds, err := kr.LoadCredentials(); err == nil && creds.ServerAddress != "" {
			addr = creds.ServerAddress
		}
	}

	if addr != "" {
		host := addr
		if idx := strings.Index(addr, ":"); idx != -1 {
			host = addr[:idx]
		}
		return "https://" + host
	}

	return DefaultServerURL
}

func saveToken(t string) error {
	kr := keyring.New()
	creds := keyring.Credentials{
		Token:      t,
		AuthMethod: "token",
	}
	if serverAddr != "" {
		creds.ServerAddress = serverAddr
	}
	if err := kr.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	fmt.Println("Token saved. You can now use fxtunnel without --token flag.")
	return nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func runLogout(cmd *cobra.Command, args []string) error {
	kr := keyring.New()
	if err := kr.Clear(); err != nil {
		return fmt.Errorf("failed to remove credentials: %w", err)
	}
	fmt.Println("Credentials removed.")
	return nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	resolveCredentials()
	addr := normalizeServerAddr(serverAddr)

	fmt.Printf("  \033[90mChecking for updates from %s...\033[0m\n", addr)

	info, err := client.CheckUpdate(addr, Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  \033[31mUpdate check failed: %v\033[0m\n", err)
		return err
	}
	if info == nil {
		fmt.Println("  \033[32mAlready up to date.\033[0m")
		return nil
	}

	if client.IsVersionIncompatible(info.MinVersion, Version) {
		fmt.Printf("  \033[33mIncompatible version %s (minimum: %s)\033[0m\n", Version, info.MinVersion)
	} else {
		fmt.Printf("  \033[33mNew version available: %s (current: %s)\033[0m\n", info.ClientVersion, Version)
	}

	if info.DownloadURL == "" {
		fmt.Println("  \033[31mNo download available for your platform.\033[0m")
		return nil
	}

	fmt.Printf("  \033[90mDownloading...\033[0m\n")
	if err := client.SelfUpdate(info.DownloadURL); err != nil {
		fmt.Fprintf(os.Stderr, "  \033[31mUpdate failed: %v\033[0m\n", err)
		return err
	}

	fmt.Printf("  \033[32mUpdated to %s. Please restart the client.\033[0m\n", info.ClientVersion)
	return nil
}

func checkAndAutoUpdate(addr string) {
	info, err := client.CheckUpdate(addr, Version)
	if err != nil || info == nil {
		return
	}

	if client.IsVersionIncompatible(info.MinVersion, Version) {
		fmt.Printf("  \033[33mIncompatible version %s (minimum: %s), updating...\033[0m\n", Version, info.MinVersion)
		if info.DownloadURL == "" {
			fmt.Fprintf(os.Stderr, "  \033[31mNo download available for this platform\033[0m\n")
			os.Exit(1)
		}
		if err := client.SelfUpdateAndRestart(info.DownloadURL); err != nil {
			fmt.Fprintf(os.Stderr, "  \033[31mAuto-update failed: %v\033[0m\n", err)
			os.Exit(1)
		}
		return // unreachable after restart
	}

	fmt.Printf("  \033[33mNew version available: %s (current: %s). Run 'fxtunnel update' to upgrade.\033[0m\n", info.ClientVersion, Version)
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
		return "mfdev.ru:" + defaultControlPort
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
	log.Debug().
		Str("version", Version).
		Str("server", cfg.Server.Address).
		Msg("Starting fxTunnel Client")

	// Create client
	c := client.New(cfg, log)

	fmt.Println("  \033[90mConnecting to fxtunnel server...\033[0m")

	// Connect
	if err := c.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "  \033[31mFailed to connect: %v\033[0m\n", err)
		os.Exit(1)
	}

	// Background update check (with forced auto-update if incompatible)
	go checkAndAutoUpdate(cfg.Server.Address)

	fmt.Println("  \033[32mTunnel established!\033[0m")
	for _, t := range c.GetTunnels() {
		if t.URL != "" {
			fmt.Printf("  HTTP: %s\n", t.URL)
		} else {
			fmt.Printf("  %s: %s\n", strings.ToUpper(t.Config.Type), t.RemoteAddr)
		}
		fmt.Printf("  Forwarding to localhost:%d\n", t.Config.LocalPort)
	}
	fmt.Println("  \033[90mReady to receive connections\033[0m")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	done := make(chan struct{})
	go func() { c.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Warn().Msg("Close timeout, exiting")
	}
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
