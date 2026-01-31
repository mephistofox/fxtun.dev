package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/mephistofox/fxtunnel/internal/config"
)

const projectConfigFile = "fxtunnel.yaml"

type projectConfig struct {
	Tunnels []config.TunnelConfig `yaml:"tunnels"`
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize tunnel configuration for the current project",
		Long: `Interactively create fxtunnel.yaml in the current directory.
Configures tunnels for the project. Requires authentication — if not logged in,
you will be prompted to run 'fxtunnel login' first.`,
		RunE: runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	scanner := bufio.NewScanner(os.Stdin)

	// 1. Check authentication
	_, _, ok := checkAuth()
	if !ok {
		fmt.Println("You are not logged in.")
		fmt.Println("Please run 'fxtunnel login' first to save your API token.")
		return fmt.Errorf("authentication required")
	}
	fmt.Println("✓ Authenticated")

	// 2. Check existing config
	var existing *projectConfig
	if _, err := os.Stat(projectConfigFile); err == nil {
		fmt.Printf("\n%s already exists.\n", projectConfigFile)
		fmt.Print("Overwrite or add tunnels? [o]verwrite / [a]dd (default: add): ")
		choice := readLine(scanner)
		if strings.HasPrefix(strings.ToLower(choice), "o") {
			existing = nil
		} else {
			data, err := os.ReadFile(projectConfigFile)
			if err != nil {
				return fmt.Errorf("read existing config: %w", err)
			}
			existing = &projectConfig{}
			if err := yaml.Unmarshal(data, existing); err != nil {
				return fmt.Errorf("parse existing config: %w", err)
			}
		}
	}

	// 3. Collect tunnels
	var tunnels []config.TunnelConfig
	if existing != nil {
		tunnels = existing.Tunnels
		fmt.Printf("\nExisting tunnels: %d\n", len(tunnels))
		for _, t := range tunnels {
			fmt.Printf("  - %s (%s, port %d)\n", t.Name, t.Type, t.LocalPort)
		}
	}

	for {
		fmt.Println("\n--- Add tunnel ---")

		// Type
		fmt.Print("Type [http/tcp/udp] (default: http): ")
		tunnelType := readLine(scanner)
		if tunnelType == "" {
			tunnelType = "http"
		}
		tunnelType = strings.ToLower(tunnelType)
		if tunnelType != "http" && tunnelType != "tcp" && tunnelType != "udp" {
			fmt.Println("Invalid type. Use http, tcp, or udp.")
			continue
		}

		// Name
		defaultName := tunnelType
		fmt.Printf("Name (default: %s): ", defaultName)
		name := readLine(scanner)
		if name == "" {
			name = defaultName
		}

		// Local port
		fmt.Print("Local port: ")
		portStr := readLine(scanner)
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			fmt.Println("Invalid port number.")
			continue
		}

		tunnel := config.TunnelConfig{
			Name:      name,
			Type:      tunnelType,
			LocalPort: port,
		}

		// Type-specific fields
		if tunnelType == "http" {
			fmt.Print("Subdomain (leave empty for auto): ")
			sub := readLine(scanner)
			if sub != "" {
				tunnel.Subdomain = sub
			}
		} else {
			fmt.Print("Remote port (0 for auto): ")
			rpStr := readLine(scanner)
			if rpStr != "" {
				rp, err := strconv.Atoi(rpStr)
				if err == nil {
					tunnel.RemotePort = rp
				}
			}
		}

		tunnels = append(tunnels, tunnel)
		fmt.Printf("✓ Added tunnel '%s' (%s → localhost:%d)\n", tunnel.Name, tunnel.Type, tunnel.LocalPort)

		// More?
		fmt.Print("\nAdd another tunnel? [y/N]: ")
		more := readLine(scanner)
		if !strings.HasPrefix(strings.ToLower(more), "y") {
			break
		}
	}

	// 4. Write config
	cfg := projectConfig{Tunnels: tunnels}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(projectConfigFile, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Printf("\n✓ Saved %s with %d tunnel(s)\n", projectConfigFile, len(tunnels))
	fmt.Println("Run 'fxtunnel' to start tunnels.")
	return nil
}

func readLine(scanner *bufio.Scanner) string {
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}
