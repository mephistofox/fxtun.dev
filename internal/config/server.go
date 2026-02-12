package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// ServerConfig holds all server configuration
type ServerConfig struct {
	Server        ServerSettings       `mapstructure:"server"`
	Domain        DomainSettings       `mapstructure:"domain"`
	Auth          AuthSettings         `mapstructure:"auth"`
	TLS           TLSSettings          `mapstructure:"tls"`
	Logging       LoggingSettings      `mapstructure:"logging"`
	Web           WebSettings          `mapstructure:"web"`
	Database      DatabaseSettings     `mapstructure:"database"`
	TOTP          TOTPSettings         `mapstructure:"totp"`
	Downloads     DownloadsSettings    `mapstructure:"downloads"`
	Inspect       InspectSettings      `mapstructure:"inspect"`
	CustomDomains CustomDomainSettings `mapstructure:"custom_domains"`
	OAuth         OAuthSettings        `mapstructure:"oauth"`
	YooKassa      YooKassaSettings     `mapstructure:"yookassa"`
	Payments      PaymentsSettings     `mapstructure:"payments"`
	SMTP          SMTPSettings         `mapstructure:"smtp"`
}

// ServerSettings contains network settings
type ServerSettings struct {
	ControlPort        int       `mapstructure:"control_port"`
	HTTPPort           int       `mapstructure:"http_port"`
	TCPPortRange       PortRange `mapstructure:"tcp_port_range"`
	UDPPortRange       PortRange `mapstructure:"udp_port_range"`
	CompressionEnabled bool      `mapstructure:"compression_enabled"`
	MinVersion         string    `mapstructure:"min_version"`
}

// PortRange defines a range of ports
type PortRange struct {
	Min int `mapstructure:"min"`
	Max int `mapstructure:"max"`
}

// DomainSettings contains domain configuration
type DomainSettings struct {
	Base     string   `mapstructure:"base"`
	Aliases  []string `mapstructure:"aliases"`
	Wildcard bool     `mapstructure:"wildcard"`
}

// AuthSettings contains authentication configuration
type AuthSettings struct {
	Enabled         bool          `mapstructure:"enabled"`
	Tokens          []TokenConfig `mapstructure:"tokens"`
	JWTSecret       string        `mapstructure:"jwt_secret"`
	AccessTokenTTL  string        `mapstructure:"access_token_ttl"`
	RefreshTokenTTL string        `mapstructure:"refresh_token_ttl"`
	MaxDomains      int           `mapstructure:"max_domains_per_user"`
}

// WebSettings contains web panel configuration
type WebSettings struct {
	Enabled     bool            `mapstructure:"enabled"`
	Port        int             `mapstructure:"port"`
	CORSOrigins []string        `mapstructure:"cors_origins"`
	RateLimit   RateLimitConfig `mapstructure:"rate_limit"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	Enabled      bool `mapstructure:"enabled"`
	AuthPerMin   int  `mapstructure:"auth_per_min"`
	GlobalPerMin int  `mapstructure:"global_per_min"`
}

// DatabaseSettings contains database configuration
type DatabaseSettings struct {
	Path string `mapstructure:"path"`
}

// TOTPSettings contains TOTP 2FA configuration
type TOTPSettings struct {
	Enabled       bool   `mapstructure:"enabled"`
	Issuer        string `mapstructure:"issuer"`
	EncryptionKey string `mapstructure:"encryption_key"`
}

// DownloadsSettings contains client downloads configuration
type DownloadsSettings struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// InspectSettings contains traffic inspection configuration
type InspectSettings struct {
	Enabled     bool   `mapstructure:"enabled"`
	Addr        string `mapstructure:"addr"`
	MaxEntries  int    `mapstructure:"max_entries"`
	MaxBodySize int    `mapstructure:"max_body_size"`
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
	Enabled       bool   `mapstructure:"enabled"`
	CertFile      string `mapstructure:"cert_file"`
	KeyFile       string `mapstructure:"key_file"`
	HTTPSPort     int    `mapstructure:"https_port"`
	ACMEEmail     string `mapstructure:"acme_email"`
	ACMEDirectory string `mapstructure:"acme_directory"`
}

// CustomDomainSettings contains custom domain configuration
type CustomDomainSettings struct {
	Enabled    bool `mapstructure:"enabled"`
	MaxPerUser int  `mapstructure:"max_per_user"`
}

// OAuthSettings contains OAuth provider configuration
type OAuthSettings struct {
	GitHub GitHubOAuthSettings `mapstructure:"github"`
	Google GoogleOAuthSettings `mapstructure:"google"`
}

// GitHubOAuthSettings contains GitHub OAuth configuration with per-domain credentials
type GitHubOAuthSettings struct {
	Domains []GitHubDomainCredentials `mapstructure:"domains"`
}

// GitHubDomainCredentials contains GitHub OAuth credentials for a specific domain
type GitHubDomainCredentials struct {
	Domain       string `mapstructure:"domain"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

// GetCredentials returns GitHub OAuth credentials for the given host
func (g *GitHubOAuthSettings) GetCredentials(host string) *GitHubDomainCredentials {
	domain := extractDomain(host)
	for i := range g.Domains {
		if g.Domains[i].Domain == domain && g.Domains[i].ClientID != "" {
			return &g.Domains[i]
		}
	}
	return nil
}

// GoogleOAuthSettings contains Google OAuth configuration (single app for all domains)
type GoogleOAuthSettings struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

// extractDomain removes port from host if present
func extractDomain(host string) string {
	if idx := strings.Index(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}

// LoggingSettings contains logging configuration
type LoggingSettings struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// YooKassaSettings contains YooKassa payment configuration
type YooKassaSettings struct {
	Enabled   bool   `mapstructure:"enabled"`
	ShopID    string `mapstructure:"shop_id"`
	SecretKey string `mapstructure:"secret_key"`
	TestMode  bool   `mapstructure:"test_mode"`
	ReturnURL string `mapstructure:"return_url"`
}

// PaymentDomainSettings contains per-domain payment settings
type PaymentDomainSettings struct {
	Enabled  bool   `mapstructure:"enabled"`
	Provider string `mapstructure:"provider"` // "yookassa"
	Message  string `mapstructure:"message"`  // Message when disabled
}

// PaymentsSettings contains payment configuration
type PaymentsSettings struct {
	Domains map[string]PaymentDomainSettings `mapstructure:"domains"`
}

// SMTPSettings contains SMTP email configuration
type SMTPSettings struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	SSLPort  int    `mapstructure:"ssl_port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
	BaseURL  string `mapstructure:"base_url"` // Base URL for email links (e.g. https://fxtun.ru)
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
	v.SetDefault("server.compression_enabled", true)
	v.SetDefault("domain.base", "localhost")
	v.SetDefault("domain.wildcard", true)
	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.jwt_secret", "")
	v.SetDefault("auth.access_token_ttl", "15m")
	v.SetDefault("auth.refresh_token_ttl", "168h")
	v.SetDefault("auth.max_domains_per_user", 3)
	v.SetDefault("tls.enabled", false)
	v.SetDefault("tls.https_port", 443)
	v.SetDefault("tls.acme_email", "")
	v.SetDefault("tls.acme_directory", "")
	v.SetDefault("custom_domains.enabled", false)
	v.SetDefault("custom_domains.max_per_user", 3)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")
	v.SetDefault("web.enabled", false)
	v.SetDefault("web.port", 8081)
	v.SetDefault("database.path", "./data/fxtunnel.db")
	v.SetDefault("totp.enabled", true)
	v.SetDefault("totp.issuer", "fxTunnel")
	v.SetDefault("totp.encryption_key", "")
	v.SetDefault("web.rate_limit.enabled", true)
	v.SetDefault("web.rate_limit.auth_per_min", 5)
	v.SetDefault("web.rate_limit.global_per_min", 100)
	v.SetDefault("downloads.enabled", true)
	v.SetDefault("downloads.path", "./downloads")
	v.SetDefault("inspect.enabled", true)
	v.SetDefault("inspect.max_entries", 1000)
	v.SetDefault("inspect.max_body_size", 262144)
	v.SetDefault("yookassa.enabled", false)
	v.SetDefault("yookassa.test_mode", true)
	v.SetDefault("smtp.enabled", false)
	v.SetDefault("smtp.port", 587)
	v.SetDefault("smtp.ssl_port", 465)
	v.SetDefault("smtp.from_name", "fxTunnel")

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
		hasStaticCerts := c.TLS.CertFile != "" && c.TLS.KeyFile != ""
		hasACME := c.CustomDomains.Enabled
		if !hasStaticCerts && !hasACME {
			return fmt.Errorf("TLS enabled but neither cert_file/key_file nor custom_domains.enabled is set")
		}
	}

	if c.Web.Enabled {
		if c.Auth.JWTSecret == "" {
			return fmt.Errorf("auth.jwt_secret is required when web panel is enabled")
		}
		if len(c.Auth.JWTSecret) < 32 {
			return fmt.Errorf("auth.jwt_secret must be at least 32 characters")
		}
		if c.TOTP.EncryptionKey == "" {
			return fmt.Errorf("totp.encryption_key is required when web panel is enabled")
		}
		if len(c.TOTP.EncryptionKey) < 16 {
			return fmt.Errorf("totp.encryption_key must be at least 16 characters")
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
		if strings.Contains(pattern, "*") {
			// Split on * and check prefix/suffix
			parts := strings.SplitN(pattern, "*", 2)
			if len(parts) == 2 {
				if strings.HasPrefix(subdomain, parts[0]) && strings.HasSuffix(subdomain, parts[1]) {
					if len(subdomain) >= len(parts[0])+len(parts[1]) {
						return true
					}
				}
			}
		} else if pattern == subdomain {
			return true
		}
	}
	return false
}
