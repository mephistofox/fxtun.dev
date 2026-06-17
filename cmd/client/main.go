package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	client "github.com/mephistofox/fxtun.dev/internal/client/core"
	"github.com/mephistofox/fxtun.dev/internal/client/keyring"
	"github.com/mephistofox/fxtun.dev/internal/config"
)

const defaultControlPort = "4443"

var (
	Version          = "dev"
	BuildTime        = "unknown"
	DefaultServerURL = "https://fxtun.dev"
)

var (
	configFile string
	serverAddr string
	token      string
	logLevel   string
	logFormat  string

	// Quick tunnel flags
	remotePort   int
	domain       string
	authFlag     string
	allowIPsFlag []string

	// Auto-close flags
	autoCloseFlag   string
	maxLifetimeFlag string

	// Preset flag
	presetFlag string

	// Inspector flags
	inspectAddr string
	noInspect   bool

	// TLS flags
	insecureFlag bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "fxtunnel",
		Short: "fxTunnel Client - Expose local services to the internet",
		Long: `fxTunnel Client connects to a fxTunnel Server and creates tunnels
to expose local services to the internet via HTTP subdomains, TCP ports, or UDP ports.

Quick start:
  fxtunnel login                       Save your API token
  fxtunnel http 3000                   Expose local HTTP server
  fxtunnel tcp 22                      Expose local TCP service
  fxtunnel udp 53                      Expose local UDP service

Tunneling options:
  fxtunnel http 3000 --domain myapp    Use a custom subdomain
  fxtunnel tcp 22 --remote-port 2222   Use a specific remote port
  fxtunnel --config client.yaml        Use config file for multiple tunnels

Security options (HTTP tunnels):
  fxtunnel http 3000 --auth user:pass           Require HTTP Basic Auth
  fxtunnel http 3000 --allow-ip 203.0.113.0/24  Restrict by IP/CIDR
  fxtunnel http 3000 --auto-close 30m           Auto-close after idle period
  fxtunnel http 3000 --max-lifetime 8h          Set maximum tunnel lifetime
  fxtunnel http 3000 --preset openclaw          Apply security preset

Daemon mode (background):
  fxtunnel up                          Start daemon from config file
  fxtunnel status                      Show daemon status and tunnels
  fxtunnel down                        Stop daemon gracefully

Domain management:
  fxtunnel domains list                List reserved subdomains
  fxtunnel domains add myapp           Reserve a subdomain

Project setup:
  fxtunnel init                        Create fxtunnel.yaml interactively
  fxtunnel presets                     List available security presets

Authentication:
  fxtunnel login                       Save API token (interactive or -t)
  fxtunnel logout                      Remove saved credentials

Configuration:
  -c, --config <path>                  Use config file for multiple tunnels
  -s, --server <host:port>             Server address (default port: 4443)
  -t, --token <token>                  API token (or use 'fxtunnel login')
  --log-level debug|info|warn|error    Log verbosity (default: warn)
  --inspect-addr <addr>                Inspector address (default 127.0.0.1:4040)
  --no-inspect                         Disable traffic inspector

For GUI mode, use fxtunnel-gui binary.`,
		RunE: runConfig,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().StringVarP(&serverAddr, "server", "s", "", "Server address (host or host:port, default port: 4443)")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "Authentication token")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "console", "Log format (console, json)")
	rootCmd.PersistentFlags().StringVar(&inspectAddr, "inspect-addr", "", "Inspector listen address (default 127.0.0.1:4040)")
	rootCmd.PersistentFlags().BoolVar(&noInspect, "no-inspect", false, "Disable local traffic inspector")
	rootCmd.PersistentFlags().BoolVar(&insecureFlag, "insecure", false, "Connect without TLS (for servers without TLS enabled)")

	// HTTP tunnel command
	httpCmd := &cobra.Command{
		Use:   "http <local_port>",
		Short: "Create an HTTP tunnel",
		Long: `Create an HTTP tunnel to expose a local web service.

Security options:
  --auth user:pass         Require HTTP Basic Auth for tunnel access
  --allow-ip 1.2.3.4      Restrict access to specific IPs/CIDRs (repeatable)
  --auto-close 30m         Auto-close tunnel after idle period (1m-24h)
  --max-lifetime 8h        Maximum tunnel lifetime (1m-7d)
  --preset openclaw        Apply security preset (random Basic Auth)

Presets provide a convenient shorthand for common security configurations.
Explicit flags override preset values.`,
		Args: cobra.ExactArgs(1),
		RunE: runHTTP,
	}
	httpCmd.Flags().StringVarP(&domain, "domain", "d", "", "Subdomain to use (auto-generated if not set)")
	httpCmd.Flags().StringVar(&domain, "subdomain", "", "Alias for --domain")
	httpCmd.Flags().StringVar(&authFlag, "auth", "", "HTTP Basic Auth credentials (format: user:password, min 8 char password)")
	httpCmd.Flags().StringSliceVar(&allowIPsFlag, "allow-ip", nil, "Allowed IP/CIDR (repeatable, e.g. 203.0.113.10,10.0.0.0/8)")
	httpCmd.Flags().StringVar(&autoCloseFlag, "auto-close", "", "Auto-close tunnel after idle duration (e.g. 5m, 30m, 2h)")
	httpCmd.Flags().StringVar(&maxLifetimeFlag, "max-lifetime", "", "Maximum tunnel lifetime (e.g. 1h, 8h, 7d)")
	httpCmd.Flags().StringVar(&presetFlag, "preset", "", "Apply a named preset (available: openclaw)")
	rootCmd.AddCommand(httpCmd)

	// TCP tunnel command
	tcpCmd := &cobra.Command{
		Use:   "tcp <local_port>",
		Short: "Create a TCP tunnel",
		Long: `Create a TCP tunnel to expose a local TCP service.

Security options:
  --allow-ip 1.2.3.4      Restrict access to specific IPs/CIDRs (repeatable)
  --auto-close 30m         Auto-close tunnel after idle period (1m-24h)
  --max-lifetime 8h        Maximum tunnel lifetime (1m-7d)`,
		Args: cobra.ExactArgs(1),
		RunE: runTCP,
	}
	tcpCmd.Flags().IntVarP(&remotePort, "remote-port", "r", 0, "Remote port (auto-assigned if 0)")
	tcpCmd.Flags().StringSliceVar(&allowIPsFlag, "allow-ip", nil, "Allowed IP/CIDR (repeatable, e.g. 203.0.113.10,10.0.0.0/8)")
	tcpCmd.Flags().StringVar(&autoCloseFlag, "auto-close", "", "Auto-close tunnel after idle duration (e.g. 5m, 30m, 2h)")
	tcpCmd.Flags().StringVar(&maxLifetimeFlag, "max-lifetime", "", "Maximum tunnel lifetime (e.g. 1h, 8h, 7d)")
	rootCmd.AddCommand(tcpCmd)

	// UDP tunnel command
	udpCmd := &cobra.Command{
		Use:   "udp <local_port>",
		Short: "Create a UDP tunnel",
		Long: `Create a UDP tunnel to expose a local UDP service.

Security options:
  --allow-ip 1.2.3.4      Restrict access to specific IPs/CIDRs (repeatable)
  --auto-close 30m         Auto-close tunnel after idle period (1m-24h)
  --max-lifetime 8h        Maximum tunnel lifetime (1m-7d)`,
		Args: cobra.ExactArgs(1),
		RunE: runUDP,
	}
	udpCmd.Flags().IntVarP(&remotePort, "remote-port", "r", 0, "Remote port (auto-assigned if 0)")
	udpCmd.Flags().StringSliceVar(&allowIPsFlag, "allow-ip", nil, "Allowed IP/CIDR (repeatable, e.g. 203.0.113.10,10.0.0.0/8)")
	udpCmd.Flags().StringVar(&autoCloseFlag, "auto-close", "", "Auto-close tunnel after idle duration (e.g. 5m, 30m, 2h)")
	udpCmd.Flags().StringVar(&maxLifetimeFlag, "max-lifetime", "", "Maximum tunnel lifetime (e.g. 1h, 8h, 7d)")
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
		Long: `Remove the saved API token and server address from the system keyring.
After logout, you will need to run 'fxtunnel login' or use --token for all commands.`,
		RunE: runLogout,
	}
	rootCmd.AddCommand(logoutCmd)

	// Init command
	rootCmd.AddCommand(newInitCmd())

	// Domains command
	rootCmd.AddCommand(newDomainsCmd())

	// Presets command
	presetsCmd := &cobra.Command{
		Use:   "presets",
		Short: "List available security presets",
		Long:  `Show all available presets and the flags they apply when used with --preset.`,
		Run: func(cmd *cobra.Command, args []string) {
			printPresets()
		},
	}
	rootCmd.AddCommand(presetsCmd)

	// Daemon commands
	rootCmd.AddCommand(newUpCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newDownCmd())

	// Update command
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Check for updates and self-update",
		Long: `Check the server for a newer client version and self-update if available.

The server address is taken from --server flag or saved credentials.`,
		RunE: runUpdate,
	}
	rootCmd.AddCommand(updateCmd)

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print the client version, build timestamp, and project URLs.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("fxTunnel Client %s (built %s)\n", Version, BuildTime)
			fmt.Println("GitHub: https://github.com/mephistofox/fxtun.dev")
			fmt.Printf("Website: %s\n", getInstalledWebsite())
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
	if noInspect {
		cfg.Inspect.Enabled = false
	}
	if inspectAddr != "" {
		cfg.Inspect.Addr = inspectAddr
	}
	if insecureFlag {
		cfg.Server.Insecure = true
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

	// Apply preset (explicit flags override preset values)
	if presetFlag != "" {
		preset, err := resolvePreset(presetFlag)
		if err != nil {
			return err
		}
		if !cmd.Flags().Changed("auth") && authFlag == "" {
			authFlag = preset.AuthUser + ":" + preset.AuthPass
			fmt.Printf("Preset '%s' credentials:\n", presetFlag)
			fmt.Printf("  Username: %s\n", preset.AuthUser)
			fmt.Printf("  Password: %s\n", preset.AuthPass)
			fmt.Println()
		}
		if !cmd.Flags().Changed("auto-close") && autoCloseFlag == "" && preset.AutoClose != "" {
			autoCloseFlag = preset.AutoClose
		}
		if !cmd.Flags().Changed("max-lifetime") && maxLifetimeFlag == "" && preset.MaxLifetime != "" {
			maxLifetimeFlag = preset.MaxLifetime
		}
		if !cmd.Flags().Changed("allow-ip") && len(allowIPsFlag) == 0 && len(preset.AllowIPs) > 0 {
			allowIPsFlag = preset.AllowIPs
		}
	}

	// Validate and hash --auth flag
	var basicAuthHash string
	if authFlag != "" {
		parts := strings.SplitN(authFlag, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --auth format: must be user:password")
		}
		username, password := parts[0], parts[1]
		if len(username) < 1 {
			return fmt.Errorf("invalid --auth: username must be at least 1 character")
		}
		if strings.Contains(username, ":") {
			return fmt.Errorf("invalid --auth: username must not contain ':'")
		}
		if len(password) < 8 {
			return fmt.Errorf("invalid --auth: password must be at least 8 characters")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(username+":"+password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash auth credentials: %w", err)
		}
		basicAuthHash = string(hash)
	}

	// Validate --allow-ip entries
	if err := validateAllowIPs(allowIPsFlag); err != nil {
		return err
	}

	// Validate --auto-close
	if err := client.ValidateAutoClose(autoCloseFlag); err != nil {
		return err
	}

	// Validate --max-lifetime
	if err := client.ValidateMaxLifetime(maxLifetimeFlag); err != nil {
		return err
	}

	tunnelCfg := config.TunnelConfig{
		Name:          fmt.Sprintf("http-%d", port),
		Type:          "http",
		LocalPort:     port,
		Subdomain:     domain,
		BasicAuthHash: basicAuthHash,
		AllowIPs:      allowIPsFlag,
		AutoClose:     autoCloseFlag,
		MaxLifetime:   maxLifetimeFlag,
	}
	if addTunnelToDaemon(tunnelCfg) {
		return nil
	}

	cfg := buildConfig(tunnelCfg)
	return runClient(cfg, log)
}

func runTCP(cmd *cobra.Command, args []string) error {
	resolveCredentials()
	log := setupLogging(logLevel, logFormat)

	port, err := parsePort(args[0])
	if err != nil {
		return err
	}

	// Validate --allow-ip entries
	if err := validateAllowIPs(allowIPsFlag); err != nil {
		return err
	}

	// Validate --auto-close
	if err := client.ValidateAutoClose(autoCloseFlag); err != nil {
		return err
	}

	// Validate --max-lifetime
	if err := client.ValidateMaxLifetime(maxLifetimeFlag); err != nil {
		return err
	}

	tunnelCfg := config.TunnelConfig{
		Name:        fmt.Sprintf("tcp-%d", port),
		Type:        "tcp",
		LocalPort:   port,
		RemotePort:  remotePort,
		AllowIPs:    allowIPsFlag,
		AutoClose:   autoCloseFlag,
		MaxLifetime: maxLifetimeFlag,
	}
	if addTunnelToDaemon(tunnelCfg) {
		return nil
	}

	cfg := buildConfig(tunnelCfg)
	return runClient(cfg, log)
}

func runUDP(cmd *cobra.Command, args []string) error {
	resolveCredentials()
	log := setupLogging(logLevel, logFormat)

	port, err := parsePort(args[0])
	if err != nil {
		return err
	}

	// Validate --allow-ip entries
	if err := validateAllowIPs(allowIPsFlag); err != nil {
		return err
	}

	// Validate --auto-close
	if err := client.ValidateAutoClose(autoCloseFlag); err != nil {
		return err
	}

	// Validate --max-lifetime
	if err := client.ValidateMaxLifetime(maxLifetimeFlag); err != nil {
		return err
	}

	tunnelCfg := config.TunnelConfig{
		Name:        fmt.Sprintf("udp-%d", port),
		Type:        "udp",
		LocalPort:   port,
		RemotePort:  remotePort,
		AllowIPs:    allowIPsFlag,
		AutoClose:   autoCloseFlag,
		MaxLifetime: maxLifetimeFlag,
	}
	if addTunnelToDaemon(tunnelCfg) {
		return nil
	}

	cfg := buildConfig(tunnelCfg)
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
		return client.WebBaseURL(addr)
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
	if err := client.SelfUpdate(info.DownloadURL, info.ServerHost); err != nil {
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
		if err := client.SelfUpdateAndRestart(info.DownloadURL, info.ServerHost); err != nil {
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
			Address:  normalizeServerAddr(serverAddr),
			Token:    token,
			Insecure: insecureFlag,
		},
		Tunnels: []config.TunnelConfig{tunnel},
		Reconnect: config.ReconnectSettings{
			Enabled:     true,
			Interval:    5 * time.Second,
			MaxAttempts: 0,
		},
		Inspect: config.InspectSettings{
			Enabled:     true,
			Addr:        "127.0.0.1:4040",
			MaxBodySize: 262144,
			MaxEntries:  1000,
		},
	}

	if noInspect {
		cfg.Inspect.Enabled = false
	}
	if inspectAddr != "" {
		cfg.Inspect.Addr = inspectAddr
	}

	return cfg
}

// getInstalledWebsite returns the website URL saved by the install script.
// Falls back to DefaultServerURL if not found.
func getInstalledWebsite() string {
	exe, err := os.Executable()
	if err == nil {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(exe), ".fxtunnel-website"))
		if err == nil {
			if url := strings.TrimSpace(string(data)); url != "" {
				return url
			}
		}
	}
	return DefaultServerURL
}

// normalizeServerAddr adds default port if not specified
func normalizeServerAddr(addr string) string {
	if addr == "" {
		return "tunnel.fxtun.dev:443"
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

// validateAllowIPs validates each --allow-ip entry as either a valid IP or CIDR.
func validateAllowIPs(entries []string) error {
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			if _, _, err := net.ParseCIDR(entry); err != nil {
				return fmt.Errorf("invalid --allow-ip CIDR %q: %w", entry, err)
			}
		} else {
			if net.ParseIP(entry) == nil {
				return fmt.Errorf("invalid --allow-ip address %q", entry)
			}
		}
	}
	return nil
}

func runClient(cfg *config.ClientConfig, log zerolog.Logger) error {
	log.Debug().
		Str("version", Version).
		Str("server", cfg.Server.Address).
		Msg("Starting fxTunnel Client")

	// Create client
	c := client.New(cfg, log)
	c.SetVersion(Version)

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
			fmt.Printf("  HTTP:  %s\n", t.URL)
			httpsURL := t.HTTPSURL
			if httpsURL == "" && strings.HasPrefix(t.URL, "http://") {
				httpsURL = "https://" + strings.TrimPrefix(t.URL, "http://")
			}
			if httpsURL != "" {
				fmt.Printf("  HTTPS: %s\n", httpsURL)
			}
		} else {
			fmt.Printf("  %s: %s\n", strings.ToUpper(t.Config.Type), t.RemoteAddr)
		}
		fmt.Printf("  Forwarding to localhost:%d\n", t.Config.LocalPort)
		if t.BasicAuthEnabled {
			fmt.Println("  Basic Auth: enabled")
		}
		if t.AllowIPsCount > 0 {
			fmt.Printf("  IP Allowlist: %d %s\n", t.AllowIPsCount, pluralize(t.AllowIPsCount, "entry", "entries"))
		}
		if t.AutoClose != "" {
			fmt.Printf("  Auto-close: %s (idle timeout)\n", t.AutoClose)
		}
		if t.MaxLifetime != "" {
			fmt.Printf("  Max lifetime: %s\n", t.MaxLifetime)
		}
	}
	if addr := c.InspectorAddr(); addr != "" {
		fmt.Printf("  Inspector: http://%s\n", addr)
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

// pluralize returns singular if count == 1, otherwise plural.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
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
