package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mephistofox/fxtunnel/internal/client"
	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/daemon"
)

var daemonForeground bool

func newUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start tunnel daemon in background",
		RunE:  runUp,
	}
	cmd.Flags().BoolVar(&daemonForeground, "foreground", false, "Run in foreground instead of detaching")
	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon status and active tunnels",
		RunE:  runStatus,
	}
}

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop the tunnel daemon",
		RunE:  runDown,
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

		attr := &os.ProcAttr{
			Files: []*os.File{devNull, devNull, devNull},
		}

		proc, err := os.StartProcess(exe, newArgs, attr)
		if err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}
		_ = proc.Release()

		// Poll for up to 5 seconds
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			if st, ok := daemon.IsDaemonRunning(statePath); ok {
				fmt.Printf("Daemon started (PID %d)\n", st.PID)
				printDaemonStatus(st.APIAddr)
				return nil
			}
		}

		fmt.Println("Daemon started but status not available yet.")
		return nil
	}

	return runDaemonForeground()
}

func runDaemonForeground() error {
	resolveCredentials()
	log := setupLogging(logLevel, logFormat)

	cfg, err := config.LoadClientConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if serverAddr != "" {
		cfg.Server.Address = serverAddr
	}
	if token != "" {
		cfg.Server.Token = token
	}
	cfg.Server.Address = normalizeServerAddr(cfg.Server.Address)
	cfg.Reconnect.Enabled = true

	c := client.New(cfg, log)
	if err := c.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	mgr := daemon.NewClientManager(c)
	api := daemon.NewAPI(mgr, cfg.Server.Address)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to listen: %w", err)
	}

	srv := &http.Server{Handler: api}
	go func() { _ = srv.Serve(listener) }()

	statePath := daemon.DefaultStatePath()
	if err := daemon.SaveState(statePath, &daemon.State{
		PID:       os.Getpid(),
		APIAddr:   listener.Addr().String(),
		Server:    cfg.Server.Address,
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
			fmt.Printf("  HTTP: %s\n", t.URL)
		} else {
			fmt.Printf("  %s: %s\n", strings.ToUpper(t.Config.Type), t.RemoteAddr)
		}
		fmt.Printf("  Forwarding to localhost:%d\n", t.Config.LocalPort)
	}

	// Wait for signal or API shutdown
	select {
	case <-mgr.SigChan():
	case <-api.Done():
	}

	srv.Close()
	c.Close()
	return nil
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
	printDaemonStatus(st.APIAddr)
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
	resp, err := httpClient.Post(fmt.Sprintf("http://%s/shutdown", st.APIAddr), "", nil)
	if err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}
	resp.Body.Close()

	fmt.Println("Daemon stopped.")
	return nil
}

func printDaemonStatus(apiAddr string) {
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Get(fmt.Sprintf("http://%s/status", apiAddr))
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

func addTunnelToDaemon(tunnelCfg config.TunnelConfig) bool {
	statePath := daemon.DefaultStatePath()
	st, running := daemon.IsDaemonRunning(statePath)
	if !running {
		return false
	}

	req := daemon.AddTunnelRequest{
		Type:       tunnelCfg.Type,
		LocalPort:  tunnelCfg.LocalPort,
		RemotePort: tunnelCfg.RemotePort,
		Subdomain:  tunnelCfg.Subdomain,
		Name:       tunnelCfg.Name,
	}

	body, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal request: %v\n", err)
		return true
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Post(
		fmt.Sprintf("http://%s/tunnels", st.APIAddr),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to add tunnel to daemon: %v\n", err)
		return true
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		fmt.Fprintf(os.Stderr, "Failed to add tunnel: %s\n", errResp["error"])
		return true
	}

	var info daemon.TunnelInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		fmt.Fprintf(os.Stderr, "Tunnel added but failed to decode response: %v\n", err)
		return true
	}

	if info.URL != "" {
		fmt.Printf("  Tunnel added: %s -> localhost:%d\n", info.URL, info.LocalPort)
	} else {
		fmt.Printf("  Tunnel added: %s -> localhost:%d\n", info.RemoteAddr, info.LocalPort)
	}
	return true
}
