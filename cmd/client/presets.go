package main

import (
	"crypto/rand"
	"fmt"
)

type PresetConfig struct {
	Name        string
	Description string
	AuthUser    string
	AuthPass    string // generated at resolve time
	AutoClose   string
	MaxLifetime string
	AllowIPs    []string
}

// presetRegistry contains all known presets (without generated passwords).
var presetRegistry = []PresetConfig{
	{
		Name:        "openclaw",
		Description: "Quick secure sharing — random Basic Auth credentials",
		AuthUser:    "admin",
	},
}

func resolvePreset(name string) (*PresetConfig, error) {
	for _, p := range presetRegistry {
		if p.Name == name {
			cfg := p // copy
			if cfg.AuthUser != "" {
				pass, err := generateSecurePassword(16)
				if err != nil {
					return nil, fmt.Errorf("generate password: %w", err)
				}
				cfg.AuthPass = pass
			}
			return &cfg, nil
		}
	}

	available := make([]string, len(presetRegistry))
	for i, p := range presetRegistry {
		available[i] = p.Name
	}
	return nil, fmt.Errorf("unknown preset %q (available: %s)", name, joinStrings(available))
}

func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func printPresets() {
	fmt.Println("Available presets:")
	fmt.Println()
	for _, p := range presetRegistry {
		fmt.Printf("  %s\n", p.Name)
		fmt.Printf("    %s\n\n", p.Description)
		fmt.Println("    Flags applied:")
		if p.AuthUser != "" {
			fmt.Printf("      --auth %s:<random-16-char>  (generated on each use)\n", p.AuthUser)
		}
		if p.AutoClose != "" {
			fmt.Printf("      --auto-close %s\n", p.AutoClose)
		}
		if p.MaxLifetime != "" {
			fmt.Printf("      --max-lifetime %s\n", p.MaxLifetime)
		}
		if len(p.AllowIPs) > 0 {
			for _, ip := range p.AllowIPs {
				fmt.Printf("      --allow-ip %s\n", ip)
			}
		}
		fmt.Println()
		fmt.Println("    Explicit flags override preset values.")
		fmt.Println()
	}
}

func generateSecurePassword(n int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}
