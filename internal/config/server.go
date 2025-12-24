package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

// ServerConfig holds all server configuration
type ServerConfig struct {
	Server   ServerSettings   `mapstructure:"server"`
	Domain   DomainSettings   `mapstructure:"domain"`
	Auth     AuthSettings     `mapstructure:"auth"`
	TLS      TLSSettings      `mapstructure:"tls"`
	Logging  LoggingSettings  `mapstructure:"logging"`
}

// ServerSettings contains network settings
type ServerSettings struct {
	ControlPort  int       `mapstructure:"control_port"`
	HTTPPort     int       `mapstructure:"http_port"`
	TCPPortRange PortRange `mapstructure:"tcp_port_range"`
	UDPPortRange PortRange `mapstructure:"udp_port_range"`
}

// PortRange defines a range of ports
type PortRange struct {
	Min int `mapstructure:"min"`
	Max int `mapstructure:"max"`
}

// DomainSettings contains domain configuration
type DomainSettings struct {
	Base     string `mapstructure:"base"`
	Wildcard bool   `mapstructure:"wildcard"`
}

// AuthSettings contains authentication configuration
type AuthSettings struct {
	Enabled bool          `mapstructure:"enabled"`
	Tokens  []TokenConfig `mapstructure:"tokens"`
}

// TokenConfig defines a single auth token
type TokenConfig struct {
	Name              string   `mapstructure:"name"`
	Token             string   `mapstructure:"token"`
	AllowedSubdomains []string `mapstructure:"allowed_subdomains"`
	MaxTunnels        int      `mapstructure:"max_tunnels"`
}

// TLSSettings contains TLS configuration
type TLSSettings struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// LoggingSettings contains logging configuration
type LoggingSettings struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// LoadServerConfig loads server configuration from file
func LoadServerConfig(configPath string) (*ServerConfig, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.control_port", 4443)
	v.SetDefault("server.http_port", 8080)
	v.SetDefault("server.tcp_port_range.min", 10000)
	v.SetDefault("server.tcp_port_range.max", 20000)
	v.SetDefault("server.udp_port_range.min", 20001)
	v.SetDefault("server.udp_port_range.max", 30000)
	v.SetDefault("domain.base", "localhost")
	v.SetDefault("domain.wildcard", true)
	v.SetDefault("auth.enabled", false)
	v.SetDefault("tls.enabled", false)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Look for config in standard locations
		v.SetConfigName("server")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/fxtunnel")

		home, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(filepath.Join(home, ".fxtunnel"))
		}
	}

	// Environment variables
	v.SetEnvPrefix("FXTUNNEL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// Config file not found, use defaults
	}

	var cfg ServerConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// Validate checks the configuration for errors
func (c *ServerConfig) Validate() error {
	if c.Server.ControlPort < 1 || c.Server.ControlPort > 65535 {
		return fmt.Errorf("invalid control port: %d", c.Server.ControlPort)
	}

	if c.Server.HTTPPort < 1 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTPPort)
	}

	if c.Server.TCPPortRange.Min > c.Server.TCPPortRange.Max {
		return fmt.Errorf("invalid TCP port range: %d > %d",
			c.Server.TCPPortRange.Min, c.Server.TCPPortRange.Max)
	}

	if c.Server.UDPPortRange.Min > c.Server.UDPPortRange.Max {
		return fmt.Errorf("invalid UDP port range: %d > %d",
			c.Server.UDPPortRange.Min, c.Server.UDPPortRange.Max)
	}

	if c.TLS.Enabled {
		if c.TLS.CertFile == "" || c.TLS.KeyFile == "" {
			return fmt.Errorf("TLS enabled but cert_file or key_file not set")
		}
	}

	return nil
}

// FindToken finds a token configuration by token string
func (c *ServerConfig) FindToken(token string) *TokenConfig {
	for i := range c.Auth.Tokens {
		if c.Auth.Tokens[i].Token == token {
			return &c.Auth.Tokens[i]
		}
	}
	return nil
}

// CanUseSubdomain checks if the token can use the given subdomain
func (t *TokenConfig) CanUseSubdomain(subdomain string) bool {
	for _, pattern := range t.AllowedSubdomains {
		if pattern == "*" {
			return true
		}
		// Support wildcard patterns like "user1-*"
		if strings.Contains(pattern, "*") {
			regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*") + "$"
			if matched, _ := regexp.MatchString(regexPattern, subdomain); matched {
				return true
			}
		} else if pattern == subdomain {
			return true
		}
	}
	return false
}
