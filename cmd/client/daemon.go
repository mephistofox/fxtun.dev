package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	client "github.com/mephistofox/fxtun.dev/internal/client/core"
	"github.com/mephistofox/fxtun.dev/internal/client/daemon"
	"github.com/mephistofox/fxtun.dev/internal/config"
)

var daemonForeground bool

func newUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start tunnel daemon in background",
		Long: `Start the tunnel daemon in the background.

The daemon reads tunnels from the config file (fxtunnel.yaml or ~/.fxtunnel/client.yaml)
and maintains persistent connections with automatic reconnect.

Use --foreground to run in the foreground (useful for systemd or debugging).

Examples:
  fxtunnel up                   Start daemon using config file
  fxtunnel up --foreground      Run in foreground (no detach)`,
		RunE: runUp,
	}
	cmd.Flags().BoolVar(&daemonForeground, "foreground", false, "Run in foreground instead of detaching")
	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon status and active tunnels",
		Long: `Show the running daemon's status including PID, server address, uptime,
and a list of all active tunnels with their public URLs.`,
		RunE: runStatus,
	}
}

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop the tunnel daemon",
		Long: `Stop the tunnel daemon and close all active tunnels.

Sends a graceful shutdown request and waits for the daemon to fully stop.`,
		RunE: runDown,
	}
}

func runUp(cmd *cobra.Command, args []string) error {
	statePath := daemon.DefaultStatePath()
	if _, running := daemon.IsDaemonRunning(statePath); running {
		fmt.Println("Daemon is already running.")
		return nil
	}

	if !daemonForeground {
		// Re-exec self with --foreground
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to find executable: %w", err)
		}

		// Build args: take original os.Args and append --foreground
		newArgs := make([]string, len(os.Args))
		copy(newArgs, os.Args)
		newArgs = append(newArgs, "--foreground")

		devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
		if err != nil {
			return fmt.Errorf("failed to open devnull: %w", err)
		}
		defer devNull.Close()

		// The daemon's output goes to a log next to its state, so a failed start can be explained.
		logPath := filepath.Join(filepath.Dir(statePath), "daemon.log")
		if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
			return fmt.Errorf("failed to create %s: %w", filepath.Dir(logPath), err)
		}
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open daemon log: %w", err)
		}
		defer logFile.Close()

		attr := &os.ProcAttr{
			Files: []*os.File{devNull, logFile, logFile},
			Sys:   daemonSysProcAttr(),
		}

		proc, err := os.StartProcess(exe, newArgs, attr)
		if err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}
		exited := make(chan struct{})
		go func() { _, _ = proc.Wait(); close(exited) }()

		// Wait until the daemon serves its API or dies.
		timeout := time.After(10 * time.Second)
		for {
			select {
			case <-exited:
				out, _ := os.ReadFile(logPath)
				return fmt.Errorf("daemon exited during startup:\n%s", out)
			case <-timeout:
				fmt.Printf("Daemon not ready after 10s; check `fxtunnel status` and %s\n", logPath)
				return nil
			case <-time.After(200 * time.Millisecond):
				if st, ok := daemon.IsDaemonRunning(statePath); ok {
					fmt.Printf("Daemon started (PID %d)\n", st.PID)
					printDaemonStatus(st.APIAddr, st.Token)
					return nil
				}
			}
		}
	}

	return runDaemonForeground(cmd)
}

func runDaemonForeground(cmd *cobra.Command) error {
	cfg, log, err := loadConfig(cmd, nil)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	c := client.New(cfg, log)
	c.SetVersion(Version)
	if err := c.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	apiToken, err := daemon.GenerateToken()
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to generate daemon token: %w", err)
	}

	mgr := daemon.NewClientManager(c)
	api := daemon.NewAPI(mgr, cfg.Server.Address, apiToken)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to listen: %w", err)
	}

	srv := &http.Server{Handler: api, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(listener) }()

	statePath := daemon.DefaultStatePath()
	if err := daemon.SaveState(statePath, &daemon.State{
		PID:       os.Getpid(),
		APIAddr:   listener.Addr().String(),
		Server:    cfg.Server.Address,
		Token:     apiToken,
		StartedAt: time.Now(),
	}); err != nil {
		srv.Close()
		c.Close()
		return fmt.Errorf("failed to save state: %w", err)
	}
	defer daemon.RemoveState(statePath)

	// Print active tunnels
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
	}

	// Wait for signal or API shutdown
	var lost error
	select {
	case <-mgr.SigChan():
	case <-api.Done():
	case <-c.Done():
		// Reconnect disabled or given up: nothing left to serve.
		lost = fmt.Errorf("disconnected from server")
	}

	srv.Close()
	c.Close()
	return lost
}

func runStatus(cmd *cobra.Command, args []string) error {
	statePath := daemon.DefaultStatePath()
	st, running := daemon.IsDaemonRunning(statePath)
	if !running {
		fmt.Println("Daemon is not running.")
		return nil
	}

	fmt.Printf("Daemon running (PID %d)\n", st.PID)
	fmt.Printf("Server: %s\n", st.Server)
	printDaemonStatus(st.APIAddr, st.Token)
	return nil
}

func runDown(cmd *cobra.Command, args []string) error {
	statePath := daemon.DefaultStatePath()
	st, running := daemon.IsDaemonRunning(statePath)
	if !running {
		fmt.Println("Daemon is not running.")
		return nil
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/shutdown", st.APIAddr), nil)
	if err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+st.Token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}
	resp.Body.Close()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(300 * time.Millisecond)
		if _, ok := daemon.IsDaemonRunning(statePath); !ok {
			fmt.Println("Daemon stopped.")
			return nil
		}
	}
	fmt.Fprintln(os.Stderr, "Warning: daemon may still be shutting down.")
	return nil
}

func printDaemonStatus(apiAddr, token string) {
	httpClient := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/status", apiAddr), nil)
	if err != nil {
		fmt.Printf("  Failed to fetch status: %v\n", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("  Failed to fetch status: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var status daemon.StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		fmt.Printf("  Failed to decode status: %v\n", err)
		return
	}

	if len(status.Tunnels) == 0 {
		fmt.Println("  No active tunnels.")
	} else {
		for _, t := range status.Tunnels {
			if t.URL != "" {
				fmt.Printf("  HTTP: %s\n", t.URL)
			} else {
				fmt.Printf("  %s: %s\n", strings.ToUpper(t.Type), t.RemoteAddr)
			}
		}
	}
	fmt.Printf("  Uptime: %s\n", status.Uptime)
}

// addTunnelToDaemon adds the tunnel to a running daemon. It reports whether a
// daemon handled the request and, if so, the error to exit with.
func addTunnelToDaemon(tunnelCfg config.TunnelConfig) (bool, error) {
	statePath := daemon.DefaultStatePath()
	st, running := daemon.IsDaemonRunning(statePath)
	if !running {
		return false, nil
	}

	req := daemon.AddTunnelRequest{
		Type:          tunnelCfg.Type,
		LocalPort:     tunnelCfg.LocalPort,
		RemotePort:    tunnelCfg.RemotePort,
		Subdomain:     tunnelCfg.Subdomain,
		Name:          tunnelCfg.Name,
		BasicAuthHash: tunnelCfg.BasicAuthHash,
		AllowIPs:      tunnelCfg.AllowIPs,
		AutoClose:     tunnelCfg.AutoClose,
		MaxLifetime:   tunnelCfg.MaxLifetime,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return true, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Longer than the daemon's own 30 s wait for the server, so a slow success is not reported as a failure.
	httpClient := &http.Client{Timeout: 35 * time.Second}
	httpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/tunnels", st.APIAddr), bytes.NewReader(body))
	if err != nil {
		return true, fmt.Errorf("failed to add tunnel to daemon: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+st.Token)
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return true, fmt.Errorf("failed to add tunnel to daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return true, fmt.Errorf("failed to add tunnel: %s", errResp["error"])
	}

	var info daemon.TunnelInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return true, fmt.Errorf("tunnel added but failed to decode response: %w", err)
	}

	if info.URL != "" {
		fmt.Printf("  Tunnel added: %s -> localhost:%d\n", info.URL, info.LocalPort)
	} else {
		fmt.Printf("  Tunnel added: %s -> localhost:%d\n", info.RemoteAddr, info.LocalPort)
	}
	return true, nil
}
