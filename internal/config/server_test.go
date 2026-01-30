package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validServerConfig() *ServerConfig {
	return &ServerConfig{
		Server: ServerSettings{
			ControlPort:  4443,
			HTTPPort:     8080,
			TCPPortRange: PortRange{Min: 10000, Max: 20000},
			UDPPortRange: PortRange{Min: 20001, Max: 30000},
		},
		Domain: DomainSettings{Base: "localhost"},
	}
}

func TestServerConfigValidate_Valid(t *testing.T) {
	cfg := validServerConfig()
	assert.NoError(t, cfg.Validate())
}

func TestServerConfigValidate_InvalidControlPort(t *testing.T) {
	for _, port := range []int{0, 70000} {
		cfg := validServerConfig()
		cfg.Server.ControlPort = port
		assert.Error(t, cfg.Validate(), "port %d should be invalid", port)
	}
}

func TestServerConfigValidate_InvalidHTTPPort(t *testing.T) {
	cfg := validServerConfig()
	cfg.Server.HTTPPort = -1
	assert.Error(t, cfg.Validate())
}

func TestServerConfigValidate_InvalidTCPPortRange(t *testing.T) {
	cfg := validServerConfig()
	cfg.Server.TCPPortRange = PortRange{Min: 20000, Max: 10000}
	assert.Error(t, cfg.Validate())
}

func TestServerConfigValidate_InvalidUDPPortRange(t *testing.T) {
	cfg := validServerConfig()
	cfg.Server.UDPPortRange = PortRange{Min: 30000, Max: 20000}
	assert.Error(t, cfg.Validate())
}

func TestServerConfigValidate_TLSWithoutCerts(t *testing.T) {
	cfg := validServerConfig()
	cfg.TLS = TLSSettings{Enabled: true}
	assert.Error(t, cfg.Validate())
}

func TestServerConfigValidate_TLSWithCerts(t *testing.T) {
	cfg := validServerConfig()
	cfg.TLS = TLSSettings{Enabled: true, CertFile: "/tmp/cert.pem", KeyFile: "/tmp/key.pem"}
	assert.NoError(t, cfg.Validate())
}

func TestFindToken(t *testing.T) {
	cfg := validServerConfig()
	cfg.Auth.Tokens = []TokenConfig{
		{Name: "test", Token: "sk_abc"},
		{Name: "other", Token: "sk_xyz"},
	}

	found := cfg.FindToken("sk_abc")
	require.NotNil(t, found)
	assert.Equal(t, "test", found.Name)

	assert.Nil(t, cfg.FindToken("sk_notfound"))
}

func TestTokenCanUseSubdomain(t *testing.T) {
	tests := []struct {
		name       string
		patterns   []string
		subdomain  string
		expected   bool
	}{
		{"wildcard *", []string{"*"}, "anything", true},
		{"exact match", []string{"myapp"}, "myapp", true},
		{"exact no match", []string{"myapp"}, "other", false},
		{"prefix wildcard match", []string{"user-*"}, "user-test", true},
		{"prefix wildcard no match", []string{"user-*"}, "admin-test", false},
		{"empty list", []string{}, "anything", false},
		{"multiple patterns", []string{"app1", "app2"}, "app2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &TokenConfig{AllowedSubdomains: tt.patterns}
			assert.Equal(t, tt.expected, tc.CanUseSubdomain(tt.subdomain))
		})
	}
}

func TestValidate_MissingJWTSecret(t *testing.T) {
	cfg := validServerConfig()
	cfg.Web.Enabled = true
	cfg.Auth.JWTSecret = ""
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jwt_secret")
}

func TestValidate_MissingTOTPKey(t *testing.T) {
	cfg := validServerConfig()
	cfg.Web.Enabled = true
	cfg.Auth.JWTSecret = "this-is-a-very-long-jwt-secret-for-testing-purposes"
	cfg.TOTP.EncryptionKey = ""
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encryption_key")
}

func TestValidate_ShortJWTSecret(t *testing.T) {
	cfg := validServerConfig()
	cfg.Web.Enabled = true
	cfg.Auth.JWTSecret = "too-short"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 32 characters")
}

func TestValidate_ShortTOTPKey(t *testing.T) {
	cfg := validServerConfig()
	cfg.Web.Enabled = true
	cfg.Auth.JWTSecret = "this-is-a-very-long-jwt-secret-for-testing-purposes"
	cfg.TOTP.EncryptionKey = "short"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 16 characters")
}

func TestValidate_SecretsNotRequiredWhenWebDisabled(t *testing.T) {
	cfg := validServerConfig()
	cfg.Web.Enabled = false
	cfg.Auth.JWTSecret = ""
	cfg.TOTP.EncryptionKey = ""
	err := cfg.Validate()
	require.NoError(t, err)
}

func TestLoadServerConfig_Defaults(t *testing.T) {
	// Use a temp dir with no config files so defaults are used
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	cfg, err := LoadServerConfig("")
	require.NoError(t, err)
	assert.Equal(t, 4443, cfg.Server.ControlPort)
	assert.Equal(t, 8080, cfg.Server.HTTPPort)
	assert.Equal(t, 10000, cfg.Server.TCPPortRange.Min)
	assert.Equal(t, 20000, cfg.Server.TCPPortRange.Max)
	assert.Equal(t, 20001, cfg.Server.UDPPortRange.Min)
	assert.Equal(t, 30000, cfg.Server.UDPPortRange.Max)
	assert.Equal(t, "localhost", cfg.Domain.Base)
}

func TestLoadServerConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "server.yaml")
	yaml := `
server:
  control_port: 5555
  http_port: 9090
  tcp_port_range:
    min: 11000
    max: 12000
  udp_port_range:
    min: 21000
    max: 22000
domain:
  base: "example.com"
`
	require.NoError(t, os.WriteFile(cfgFile, []byte(yaml), 0644))

	cfg, err := LoadServerConfig(cfgFile)
	require.NoError(t, err)
	assert.Equal(t, 5555, cfg.Server.ControlPort)
	assert.Equal(t, 9090, cfg.Server.HTTPPort)
	assert.Equal(t, 11000, cfg.Server.TCPPortRange.Min)
	assert.Equal(t, 12000, cfg.Server.TCPPortRange.Max)
	assert.Equal(t, "example.com", cfg.Domain.Base)
}
