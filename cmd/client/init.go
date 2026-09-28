package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/mephistofox/fxtun.dev/internal/config"
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
	// 1. Check authentication
	_, _, ok := checkAuth()
	if !ok {
		fmt.Println("You are not logged in.")
		fmt.Println("Please run 'fxtunnel login' first to save your API token.")
		return fmt.Errorf("authentication required")
	}
	fmt.Println("✓ Authenticated")

	return runInitInteractive(bufio.NewScanner(os.Stdin))
}

// errInputClosed is returned when stdin hits EOF mid-prompt (B16): init used
// to treat EOF the same as an empty answer, which kept re-prompting forever
// on invalid-input branches.
var errInputClosed = errors.New("input closed")

// runInitInteractive drives the prompts and writes fxtunnel.yaml. Split out
// from runInit so it can be tested without a real authenticated session.
func runInitInteractive(scanner *bufio.Scanner) error {
	// 2. Check existing config. In "add" mode, existingRoot holds the parsed
	// document so every other top-level section and its comments survive the
	// rewrite (B17); only the tunnels key is replaced.
	var existingRoot *yaml.Node
	var tunnels []config.TunnelConfig
	if _, err := os.Stat(projectConfigFile); err == nil {
		fmt.Printf("\n%s already exists.\n", projectConfigFile)
		fmt.Print("Overwrite or add tunnels? [o]verwrite / [a]dd (default: add): ")
		choice, ok := readLine(scanner)
		if !ok {
			return errInputClosed
		}
		if strings.HasPrefix(strings.ToLower(choice), "o") {
			fmt.Printf("Warning: the whole %s will be replaced, including server: and any other sections.\n", projectConfigFile)
		} else {
			data, err := os.ReadFile(projectConfigFile)
			if err != nil {
				return fmt.Errorf("read existing config: %w", err)
			}
			var doc yaml.Node
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parse existing config: %w", err)
			}
			if len(doc.Content) > 0 && doc.Content[0].Kind == yaml.MappingNode {
				existingRoot = doc.Content[0]
			}
			var existing projectConfig
			if err := yaml.Unmarshal(data, &existing); err != nil {
				return fmt.Errorf("parse existing config: %w", err)
			}
			tunnels = existing.Tunnels
			fmt.Printf("\nExisting tunnels: %d\n", len(tunnels))
			for _, t := range tunnels {
				fmt.Printf("  - %s (%s, port %d)\n", t.Name, t.Type, t.LocalPort)
			}
		}
	}

	// 3. Collect tunnels
	for {
		fmt.Println("\n--- Add tunnel ---")

		// Type
		fmt.Print("Type [http/tcp/udp] (default: http): ")
		tunnelType, ok := readLine(scanner)
		if !ok {
			return errInputClosed
		}
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
		name, ok := readLine(scanner)
		if !ok {
			return errInputClosed
		}
		if name == "" {
			name = defaultName
		}

		// Local port
		fmt.Print("Local port: ")
		portStr, ok := readLine(scanner)
		if !ok {
			return errInputClosed
		}
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
			sub, ok := readLine(scanner)
			if !ok {
				return errInputClosed
			}
			if sub != "" {
				tunnel.Subdomain = sub
			}
		} else {
			fmt.Print("Remote port (0 for auto): ")
			rpStr, ok := readLine(scanner)
			if !ok {
				return errInputClosed
			}
			if rpStr != "" {
				rp, err := strconv.Atoi(rpStr)
				if err != nil || validateRemotePort(rp) != nil {
					fmt.Println("Invalid remote port.")
					continue
				}
				tunnel.RemotePort = rp
			}
		}

		tunnels = append(tunnels, tunnel)
		fmt.Printf("✓ Added tunnel '%s' (%s → localhost:%d)\n", tunnel.Name, tunnel.Type, tunnel.LocalPort)

		// More?
		fmt.Print("\nAdd another tunnel? [y/N]: ")
		more, ok := readLine(scanner)
		if !ok {
			return errInputClosed
		}
		if !strings.HasPrefix(strings.ToLower(more), "y") {
			break
		}
	}

	// 4. Write config
	data, err := marshalProjectConfig(existingRoot, tunnels)
	if err != nil {
		return err
	}

	if err := os.WriteFile(projectConfigFile, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Printf("\n✓ Saved %s with %d tunnel(s)\n", projectConfigFile, len(tunnels))
	fmt.Println("Run 'fxtunnel' to start tunnels.")
	return nil
}

// marshalProjectConfig renders the tunnels list to YAML. With existingRoot
// (add mode, B17) it replaces only the "tunnels" key in the parsed document
// node, leaving every other section and comment untouched. Without it
// (overwrite mode, or no prior file) it writes a fresh tunnels-only document.
func marshalProjectConfig(existingRoot *yaml.Node, tunnels []config.TunnelConfig) ([]byte, error) {
	if existingRoot == nil {
		data, err := yaml.Marshal(&projectConfig{Tunnels: tunnels})
		if err != nil {
			return nil, fmt.Errorf("marshal config: %w", err)
		}
		return data, nil
	}

	var tunnelsNode yaml.Node
	if err := tunnelsNode.Encode(tunnels); err != nil {
		return nil, fmt.Errorf("encode tunnels: %w", err)
	}
	setMappingValue(existingRoot, "tunnels", &tunnelsNode)

	data, err := yaml.Marshal(existingRoot)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	return data, nil
}

// setMappingValue replaces the value node for key in a YAML mapping node,
// or appends the key/value pair if the key is not present yet.
func setMappingValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
}

// readLine reads one line from stdin. ok is false on EOF or a scanner error
// (B16): callers must stop prompting rather than loop on an empty answer.
func readLine(scanner *bufio.Scanner) (string, bool) {
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), true
	}
	return "", false
}
