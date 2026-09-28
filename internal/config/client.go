package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

// ClientConfig holds all client configuration
type ClientConfig struct {
	Server    ClientServerSettings `mapstructure:"server"`
	Tunnels   []TunnelConfig       `mapstructure:"tunnels"`
	Reconnect ReconnectSettings    `mapstructure:"reconnect"`
	Inspect   InspectSettings      `mapstructure:"inspect"`
	Logging   LoggingSettings      `mapstructure:"logging"`
}

// ClientServerSettings contains server connection settings
type ClientServerSettings struct {
	Address     string `mapstructure:"address"`
	Token       string `mapstructure:"token"`
	Insecure    bool   `mapstructure:"insecure"`
	TLSVerify   bool   `mapstructure:"tls_verify"`
	Compression bool   `mapstructure:"compression"`

	// FallbackAddress is an optional secondary endpoint tried when the primary
	// fails to dial or stalls during the compression handshake (the signature
	// of middlebox interference on the non-standard plaintext port). New
	// configs set Address to the tunnel.*:443 TLS endpoint and
	// FallbackAddress to the legacy host:4443 plaintext endpoint.
	FallbackAddress  string `mapstructure:"fallback_address"`
	FallbackInsecure bool   `mapstructure:"fallback_insecure"`
}

// TunnelConfig defines a single tunnel
type TunnelConfig struct {
	Name       string `mapstructure:"name" yaml:"name"`
	Type       string `mapstructure:"type" yaml:"type"` // http, tcp, udp
	LocalAddr  string `mapstructure:"local_addr" yaml:"local_addr,omitempty"`
	LocalPort  int    `mapstructure:"local_port" yaml:"local_port"`
	RemotePort int    `mapstructure:"remote_port" yaml:"remote_port,omitempty"` // For TCP/UDP, 0 = auto-assign
	Subdomain  string `mapstructure:"subdomain" yaml:"subdomain,omitempty"`     // For HTTP tunnels

	// Security features
	BasicAuth     string   `mapstructure:"basic_auth"      yaml:"basic_auth,omitempty"`   // "user:password"
	BasicAuthHash string   `mapstructure:"basic_auth_hash" yaml:"-"`                      // derived bcrypt hash, never in YAML
	AllowIPs      []string `mapstructure:"allow_ips"       yaml:"allow_ips,omitempty"`    // CIDR list
	AutoClose     string   `mapstructure:"auto_close"      yaml:"auto_close,omitempty"`   // "30m", "2h"
	MaxLifetime   string   `mapstructure:"max_lifetime"    yaml:"max_lifetime,omitempty"` // "8h"
}

// ReconnectSettings contains reconnection configuration
type ReconnectSettings struct {
	Enabled     bool          `mapstructure:"enabled"`
	Interval    time.Duration `mapstructure:"interval"`
	MaxAttempts int           `mapstructure:"max_attempts"` // 0 = infinite
}

const (
	// DefaultServerAddress is the TLS control endpoint.
	DefaultServerAddress = "tunnel.fxtun.ru:443"
	// defaultFallbackAddress is the legacy plaintext control port.
	defaultFallbackAddress = "fxtun.ru:4443"
)

// LoadClientConfig loads client configuration from file
func LoadClientConfig(configPath string) (*ClientConfig, error) {
	return LoadClientConfigLayered(configPath, nil, nil)
}

// LoadClientConfigLayered loads the config with two extra layers around the
// usual env > file > defaults: overrides win over everything (flags the user
// set), fallbacks sit just under the file in place of the defaults (a saved
// login). Keys are viper keys such as "server.token".
func LoadClientConfigLayered(configPath string, overrides, fallbacks map[string]any) (*ClientConfig, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.address", DefaultServerAddress)
	v.SetDefault("server.insecure", false)
	v.SetDefault("server.tls_verify", true)
	v.SetDefault("server.compression", true)
	// fallback_address stays unset here on purpose: defaulting it outright
	// would inject the public fxtun.ru:4443 into self-hosted configs that only
	// set server.address, replaying a self-hoster's token to the public server
	// on a transient primary failure. It is filled in after unmarshal, and only
	// when server.address is still ours (see below).
	v.SetDefault("reconnect.enabled", true)
	v.SetDefault("reconnect.interval", "5s")
	v.SetDefault("reconnect.max_attempts", 0)
	v.SetDefault("inspect.enabled", true)
	v.SetDefault("inspect.addr", "127.0.0.1:4040")
	v.SetDefault("inspect.max_body_size", 262144)
	v.SetDefault("inspect.max_entries", 1000)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")
	for k, val := range fallbacks {
		v.SetDefault(k, val)
	}

	// A caller bringing its own tunnels (a quick http/tcp/udp command) reads
	// only ./fxtunnel.yaml: an unrelated client.yaml on the generic search
	// path must not break or redirect a one-off command.
	_, ownTunnels := overrides["tunnels"]
	readFile := true
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Priority: fxtunnel.yaml in CWD > client.yaml in CWD > configs/ > ~/.fxtunnel/
		if _, err := os.Stat("fxtunnel.yaml"); err == nil {
			v.SetConfigFile("fxtunnel.yaml")
		} else if ownTunnels {
			readFile = false
		} else {
			v.SetConfigName("client")
			// No SetConfigType: viper infers "yaml" from the "client.yaml" it
			// finds. Setting the type explicitly also makes viper match a bare
			// "client" file with no extension (e.g. a stray `go build -o
			// client` binary) and fail trying to parse it as YAML.
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
	// AutomaticEnv only covers keys viper already knows, and server.token has
	// no default. FXTUNNEL_TOKEN is the short name the CLI documents.
	_ = v.BindEnv("server.token", "FXTUNNEL_SERVER_TOKEN", "FXTUNNEL_TOKEN")

	if readFile {
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("read config: %w", err)
			}
		}
	}
	for k, val := range overrides {
		v.Set(k, val)
	}

	var cfg ClientConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	ApplyDefaultFallback(&cfg.Server)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// IsPublicAddress reports whether addr is the public fxTunnel SaaS server:
// either its TLS endpoint or the legacy plaintext fallback.
// Only the public server always requires a token; a self-hosted address may
// run with auth.enabled: false in server.yaml.
func IsPublicAddress(addr string) bool {
	return addr == DefaultServerAddress || addr == defaultFallbackAddress
}

// ApplyDefaultFallback fills in the legacy plaintext control endpoint as the
// fallback, but only for the stock SaaS address. Self-hosted configs (which
// always set server.address themselves) are left alone so a transient primary
// failure can never replay their token to the public server.
func ApplyDefaultFallback(s *ClientServerSettings) {
	if s.Address == DefaultServerAddress && s.FallbackAddress == "" {
		s.FallbackAddress = defaultFallbackAddress
		s.FallbackInsecure = true
	}
}

// Validate checks the configuration for errors
func (c *ClientConfig) Validate() error {
	if c.Server.Address == "" {
		return fmt.Errorf("server address is required")
	}

	for i := range c.Tunnels {
		t := &c.Tunnels[i]
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

		if err := ValidateAutoClose(t.AutoClose); err != nil {
			return fmt.Errorf("tunnel[%d]: %w", i, err)
		}
		if err := ValidateMaxLifetime(t.MaxLifetime); err != nil {
			return fmt.Errorf("tunnel[%d]: %w", i, err)
		}

		if err := t.deriveHashes(); err != nil {
			return fmt.Errorf("tunnel[%d]: %w", i, err)
		}
	}

	return nil
}

// deriveHashes hashes the plaintext basic_auth field into BasicAuthHash if it is set
// and BasicAuthHash has not already been provided. The plaintext is cleared after hashing.
func (t *TunnelConfig) deriveHashes() error {
	if t.BasicAuth != "" && t.BasicAuthHash == "" {
		parts := strings.SplitN(t.BasicAuth, ":", 2)
		if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) < 8 {
			return fmt.Errorf("basic_auth must be in format 'user:password' with password at least 8 characters")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(t.BasicAuth), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("basic_auth: failed to hash: %w", err)
		}
		t.BasicAuthHash = string(hash)
		t.BasicAuth = "" // clear plaintext after hashing
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
