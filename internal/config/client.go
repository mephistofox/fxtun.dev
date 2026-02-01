package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ClientConfig holds all client configuration
type ClientConfig struct {
	Server    ClientServerSettings `mapstructure:"server"`
	Tunnels   []TunnelConfig       `mapstructure:"tunnels"`
	Reconnect ReconnectSettings    `mapstructure:"reconnect"`
	Logging   LoggingSettings      `mapstructure:"logging"`
}

// ClientServerSettings contains server connection settings
type ClientServerSettings struct {
	Address     string `mapstructure:"address"`
	Token       string `mapstructure:"token"`
	Insecure    bool   `mapstructure:"insecure"`
	TLSVerify   bool   `mapstructure:"tls_verify"`
	Compression bool   `mapstructure:"compression"`
}

// TunnelConfig defines a single tunnel
type TunnelConfig struct {
	Name       string `mapstructure:"name" yaml:"name"`
	Type       string `mapstructure:"type" yaml:"type"`                          // http, tcp, udp
	LocalAddr  string `mapstructure:"local_addr" yaml:"local_addr,omitempty"`
	LocalPort  int    `mapstructure:"local_port" yaml:"local_port"`
	RemotePort int    `mapstructure:"remote_port" yaml:"remote_port,omitempty"` // For TCP/UDP, 0 = auto-assign
	Subdomain  string `mapstructure:"subdomain" yaml:"subdomain,omitempty"`    // For HTTP tunnels
}

// ReconnectSettings contains reconnection configuration
type ReconnectSettings struct {
	Enabled     bool          `mapstructure:"enabled"`
	Interval    time.Duration `mapstructure:"interval"`
	MaxAttempts int           `mapstructure:"max_attempts"` // 0 = infinite
}

// LoadClientConfig loads client configuration from file
func LoadClientConfig(configPath string) (*ClientConfig, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.address", "mfdev.ru:4443")
	v.SetDefault("server.insecure", false)
	v.SetDefault("server.tls_verify", true)
	v.SetDefault("server.compression", true)
	v.SetDefault("reconnect.enabled", true)
	v.SetDefault("reconnect.interval", "5s")
	v.SetDefault("reconnect.max_attempts", 0)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Priority: fxtunnel.yaml in CWD > client.yaml in CWD > configs/ > ~/.fxtunnel/
		if _, err := os.Stat("fxtunnel.yaml"); err == nil {
			v.SetConfigFile("fxtunnel.yaml")
		} else {
			v.SetConfigName("client")
			v.SetConfigType("yaml")
			v.AddConfigPath(".")
			v.AddConfigPath("./configs")

			home, err := os.UserHomeDir()
			if err == nil {
				v.AddConfigPath(filepath.Join(home, ".fxtunnel"))
			}
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
	}

	var cfg ClientConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// Validate checks the configuration for errors
func (c *ClientConfig) Validate() error {
	if c.Server.Address == "" {
		return fmt.Errorf("server address is required")
	}

	for i, t := range c.Tunnels {
		if t.Type == "" {
			return fmt.Errorf("tunnel[%d]: type is required", i)
		}

		switch t.Type {
		case "http":
			if t.LocalPort < 1 || t.LocalPort > 65535 {
				return fmt.Errorf("tunnel[%d]: invalid local_port: %d", i, t.LocalPort)
			}
		case "tcp", "udp":
			if t.LocalPort < 1 || t.LocalPort > 65535 {
				return fmt.Errorf("tunnel[%d]: invalid local_port: %d", i, t.LocalPort)
			}
		default:
			return fmt.Errorf("tunnel[%d]: unknown type: %s", i, t.Type)
		}
	}

	return nil
}

// GetLocalAddress returns the full local address for the tunnel
func (t *TunnelConfig) GetLocalAddress() string {
	addr := t.LocalAddr
	if addr == "" {
		addr = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", addr, t.LocalPort)
}
