package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ServerMode defines the operating mode of the server.
type ServerMode string

const (
	ModeStandalone ServerMode = "standalone" // default: single server, current behavior
	ModeHub        ServerMode = "hub"        // central coordinator with DB, API, admin
	ModeNode       ServerMode = "node"       // edge server handling tunnel connections
)

// NodeSettings contains edge node configuration (used when mode=node).
type NodeSettings struct {
	HubURL     string `mapstructure:"hub_url"`     // hub API URL, e.g. "https://hub.fxtun.dev"
	HubToken   string `mapstructure:"hub_token"`    // pre-shared secret for node authentication
	Name       string `mapstructure:"name"`         // human-readable node name, e.g. "moscow-1"
	Region     string `mapstructure:"region"`        // geographic region, e.g. "ru-msk"
	PublicAddr string `mapstructure:"public_addr"`   // public address for client connections (host:port)
	HTTPAddr   string `mapstructure:"http_addr"`     // public address for inter-node HTTP proxy (host:port)
}

// GeoIPSettings contains GeoIP database configuration for region-based node selection.
type GeoIPSettings struct {
	Enabled  bool   `mapstructure:"enabled"`
	Database string `mapstructure:"database"` // path to .mmdb file
}

// ServerConfig holds all server configuration
type ServerConfig struct {
	Mode          ServerMode           `mapstructure:"mode"`
	Node          NodeSettings         `mapstructure:"node"`
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
	Creem         CreemSettings         `mapstructure:"creem"`
	Payments      PaymentsSettings     `mapstructure:"payments"`
	SMTP          SMTPSettings         `mapstructure:"smtp"`
	Telegram      TelegramSettings     `mapstructure:"telegram"`
	ExchangeRate  float64              `mapstructure:"exchange_rate"`
	Redis         RedisSettings        `mapstructure:"redis"`
	GeoIP         GeoIPSettings        `mapstructure:"geoip"`
	DNS           DNSSettings          `mapstructure:"dns"`
}

// DNSSettings contains authoritative DNS server configuration.
type DNSSettings struct {
	Enabled  bool   `mapstructure:"enabled"`
	Listen   string `mapstructure:"listen"`    // ":53"
	ZoneFile string `mapstructure:"zone_file"` // path to YAML zone file
}

// RedisSettings contains Redis cache configuration
type RedisSettings struct {
	Enabled        bool     `mapstructure:"enabled"`
	Addr           string   `mapstructure:"addr"`
	Password       string   `mapstructure:"password"`
	DB             int      `mapstructure:"db"`
	KeyPrefix      string   `mapstructure:"key_prefix"`
	SentinelEnabled bool    `mapstructure:"sentinel_enabled"`
	SentinelMaster string   `mapstructure:"sentinel_master"`
	SentinelAddrs  []string `mapstructure:"sentinel_addrs"`
}

// ServerSettings contains network settings
type ServerSettings struct {
	ControlPort        int           `mapstructure:"control_port"`
	HTTPPort           int           `mapstructure:"http_port"`
	// HTTPBind is the address the HTTP tunnel proxy listens on. Empty = all
	// interfaces (legacy). Set to "127.0.0.1" in production to force traffic
	// through nginx (which terminates TLS and sets X-Real-IP).
	HTTPBind           string        `mapstructure:"http_bind"`
	TCPPortRange       PortRange     `mapstructure:"tcp_port_range"`
	UDPPortRange       PortRange     `mapstructure:"udp_port_range"`
	CompressionEnabled bool          `mapstructure:"compression_enabled"`
	MinVersion         string        `mapstructure:"min_version"`
	Monitor            MonitorConfig `mapstructure:"monitor"`
}

// MonitorConfig contains abuse detection settings.
// Rate limits are not configured here — they come from the plans table in the database.
type MonitorConfig struct {
	Enabled                bool          `mapstructure:"enabled"`
	DetectionInterval      time.Duration `mapstructure:"detection_interval"`
	UniqueIPsThreshold     int           `mapstructure:"unique_ips_threshold"`
	ShortConnRatio         float64       `mapstructure:"short_conn_ratio"`
	UDPAmplificationFactor int           `mapstructure:"udp_amplification_factor"`
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
	Enabled                  bool          `mapstructure:"enabled"`
	Tokens                   []TokenConfig `mapstructure:"tokens"`
	JWTSecret                string        `mapstructure:"jwt_secret"`
	AccessTokenTTL           string        `mapstructure:"access_token_ttl"`
	RefreshTokenTTL          string        `mapstructure:"refresh_token_ttl"`
	MaxDomains               int           `mapstructure:"max_domains_per_user"`
	PhoneRegistrationEnabled bool          `mapstructure:"phone_registration_enabled"`
	// PhoneRegistrationTarpit: when phone_registration_enabled=false and this is true,
	// the /api/auth/register endpoint returns a plausible 201 with fake (unusable)
	// tokens instead of 403 — so bots waste time on accounts they can't log into.
	PhoneRegistrationTarpit  bool          `mapstructure:"phone_registration_tarpit"`
	// TrustedProxies lists IP addresses whose X-Real-IP / X-Forwarded-For
	// headers may be trusted to determine the real client IP. Anything outside
	// this list is treated as a potentially-malicious direct connection and
	// the TCP source is used. Default: ["127.0.0.1", "::1"] (loopback only).
	TrustedProxies           []string      `mapstructure:"trusted_proxies"`
}

// WebSettings contains web panel configuration
type WebSettings struct {
	Enabled     bool            `mapstructure:"enabled"`
	// Bind is the address the API listens on. Empty = all interfaces
	// (legacy). Set to "127.0.0.1" in production so only nginx can reach it.
	Bind        string          `mapstructure:"bind"`
	Port        int             `mapstructure:"port"`
	CORSOrigins []string        `mapstructure:"cors_origins"`
	RateLimit   RateLimitConfig `mapstructure:"rate_limit"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	AuthPerMin     int  `mapstructure:"auth_per_min"`
	GlobalPerMin   int  `mapstructure:"global_per_min"`
	RegisterPerMin int  `mapstructure:"register_per_min"`
}

// DatabaseSettings contains database configuration
type DatabaseSettings struct {
	DSN string `mapstructure:"dsn"`
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

// CreemSettings contains Creem.io payment configuration
type CreemSettings struct {
	Enabled       bool   `mapstructure:"enabled"`
	APIKey        string `mapstructure:"api_key"`
	WebhookSecret string `mapstructure:"webhook_secret"`
	TestMode      bool   `mapstructure:"test_mode"`
	SuccessURL    string `mapstructure:"success_url"`
	CancelURL     string `mapstructure:"cancel_url"`
}

// PaymentDomainSettings contains per-domain payment settings
type PaymentDomainSettings struct {
	Enabled  bool   `mapstructure:"enabled" yaml:"enabled"`
	Provider string `mapstructure:"provider" yaml:"provider"`
	Message  string `mapstructure:"message" yaml:"message"`
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
	BaseURL  string `mapstructure:"base_url"`    // Base URL for email links (e.g. https://fxtun.ru)
	BaseURLEN string `mapstructure:"base_url_en"` // Base URL for English emails (e.g. https://fxtun.dev)
}

// TelegramSettings contains Telegram bot notification configuration
type TelegramSettings struct {
	Enabled  bool   `mapstructure:"enabled"`
	BotToken string `mapstructure:"bot_token"`
	ChatID   string `mapstructure:"chat_id"`
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
	v.SetDefault("server.monitor.enabled", true)
	v.SetDefault("server.monitor.detection_interval", "30s")
	v.SetDefault("server.monitor.unique_ips_threshold", 200)
	v.SetDefault("server.monitor.short_conn_ratio", 0.8)
	v.SetDefault("server.monitor.udp_amplification_factor", 10)
	v.SetDefault("domain.base", "localhost")
	v.SetDefault("domain.wildcard", true)
	v.SetDefault("auth.enabled", true)
	v.SetDefault("auth.jwt_secret", "")
	v.SetDefault("auth.access_token_ttl", "15m")
	v.SetDefault("auth.refresh_token_ttl", "168h")
	v.SetDefault("auth.max_domains_per_user", 3)
	v.SetDefault("auth.phone_registration_enabled", false)
	v.SetDefault("auth.phone_registration_tarpit", true)
	v.SetDefault("auth.trusted_proxies", []string{"127.0.0.1", "::1"})
	v.SetDefault("server.http_bind", "")
	v.SetDefault("web.bind", "")
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
	v.SetDefault("database.dsn", "postgres://fxtunnel:fxtunnel@localhost:5432/fxtunnel?sslmode=disable")
	v.SetDefault("totp.enabled", true)
	v.SetDefault("totp.issuer", "fxTunnel")
	v.SetDefault("totp.encryption_key", "")
	v.SetDefault("web.rate_limit.enabled", true)
	v.SetDefault("web.rate_limit.auth_per_min", 5)
	v.SetDefault("web.rate_limit.global_per_min", 100)
	v.SetDefault("web.rate_limit.register_per_min", 1)
	v.SetDefault("downloads.enabled", true)
	v.SetDefault("downloads.path", "./downloads")
	v.SetDefault("inspect.enabled", true)
	v.SetDefault("inspect.max_entries", 1000)
	v.SetDefault("inspect.max_body_size", 262144)
	v.SetDefault("yookassa.enabled", false)
	v.SetDefault("yookassa.test_mode", false)
	v.SetDefault("creem.enabled", false)
	v.SetDefault("creem.test_mode", false)
	v.SetDefault("smtp.enabled", false)
	v.SetDefault("smtp.port", 587)
	v.SetDefault("smtp.ssl_port", 465)
	v.SetDefault("smtp.from_name", "fxTunnel")
	v.SetDefault("telegram.enabled", false)
	v.SetDefault("exchange_rate", 80.0)
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.key_prefix", "fxt:")
	v.SetDefault("redis.sentinel_enabled", false)
	v.SetDefault("redis.sentinel_master", "fxtunnel-master")
	v.SetDefault("mode", "standalone")
	v.SetDefault("geoip.enabled", false)
	v.SetDefault("geoip.database", "")
	v.SetDefault("dns.enabled", false)
	v.SetDefault("dns.listen", ":53")
	v.SetDefault("dns.zone_file", "")

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

	// Viper splits dots in map keys (e.g., "fxtun.ru" becomes "fxtun" -> "ru").
	// Re-parse payments.domains directly from YAML to preserve domain names with dots.
	if cfgFile := v.ConfigFileUsed(); cfgFile != "" {
		if domains, err := parsePaymentDomains(cfgFile); err == nil && len(domains) > 0 {
			cfg.Payments.Domains = domains
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// parsePaymentDomains reads payments.domains from YAML file directly,
// bypassing Viper which mangles dots in map keys.
func parsePaymentDomains(configPath string) (map[string]PaymentDomainSettings, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Payments struct {
			Domains map[string]PaymentDomainSettings `yaml:"domains"`
		} `yaml:"payments"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	return raw.Payments.Domains, nil
}

// EffectiveMode returns the server mode, defaulting to standalone.
func (c *ServerConfig) EffectiveMode() ServerMode {
	if c.Mode == "" {
		return ModeStandalone
	}
	return c.Mode
}

// Validate checks the configuration for errors
func (c *ServerConfig) Validate() error {
	switch c.EffectiveMode() {
	case ModeStandalone, ModeHub, ModeNode:
		// valid
	default:
		return fmt.Errorf("invalid mode %q: must be standalone, hub, or node", c.Mode)
	}

	if c.EffectiveMode() == ModeNode {
		if c.Node.HubURL == "" {
			return fmt.Errorf("node.hub_url is required in node mode")
		}
		if c.Node.HubToken == "" {
			return fmt.Errorf("node.hub_token is required in node mode")
		}
		if c.Node.Name == "" {
			return fmt.Errorf("node.name is required in node mode")
		}
		if c.Node.PublicAddr == "" {
			return fmt.Errorf("node.public_addr is required in node mode")
		}
		if !c.Redis.Enabled {
			return fmt.Errorf("redis.enabled must be true in node mode")
		}
	}

	if c.EffectiveMode() == ModeHub {
		if !c.Redis.Enabled {
			return fmt.Errorf("redis.enabled must be true in hub mode")
		}
		if c.Node.HubToken == "" {
			return fmt.Errorf("node.hub_token is required in hub mode (used to authenticate edge nodes)")
		}
	}

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
