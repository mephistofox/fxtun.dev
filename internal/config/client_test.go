package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validClientConfig() *ClientConfig {
	return &ClientConfig{
		Server: ClientServerSettings{Address: "127.0.0.1:4443"},
		Tunnels: []TunnelConfig{
			{Type: "http", LocalPort: 3000},
		},
	}
}

func TestClientConfigValidate_Valid(t *testing.T) {
	assert.NoError(t, validClientConfig().Validate())
}

func TestClientConfigValidate_EmptyAddress(t *testing.T) {
	cfg := validClientConfig()
	cfg.Server.Address = ""
	assert.Error(t, cfg.Validate())
}

func TestClientConfigValidate_InvalidTunnelType(t *testing.T) {
	cfg := validClientConfig()
	cfg.Tunnels = []TunnelConfig{{Type: "invalid", LocalPort: 3000}}
	assert.Error(t, cfg.Validate())
}

func TestClientConfigValidate_InvalidPort(t *testing.T) {
	for _, port := range []int{0, 70000} {
		cfg := validClientConfig()
		cfg.Tunnels = []TunnelConfig{{Type: "http", LocalPort: port}}
		assert.Error(t, cfg.Validate(), "port %d should be invalid", port)
	}
}

func TestClientConfigValidate_MissingType(t *testing.T) {
	cfg := validClientConfig()
	cfg.Tunnels = []TunnelConfig{{LocalPort: 3000}}
	assert.Error(t, cfg.Validate())
}

func TestClientConfigValidate_TCPUDPTunnels(t *testing.T) {
	cfg := validClientConfig()
	cfg.Tunnels = []TunnelConfig{
		{Type: "tcp", LocalPort: 22},
		{Type: "udp", LocalPort: 53},
	}
	assert.NoError(t, cfg.Validate())
}

func TestTunnelConfigGetLocalAddress(t *testing.T) {
	tc := &TunnelConfig{LocalPort: 3000}
	assert.Equal(t, "127.0.0.1:3000", tc.GetLocalAddress())

	tc2 := &TunnelConfig{LocalAddr: "192.168.1.1", LocalPort: 8080}
	assert.Equal(t, "192.168.1.1:8080", tc2.GetLocalAddress())
}

func TestLoadClientConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(orig) }()

	cfg, err := LoadClientConfig("")
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:4443", cfg.Server.Address)
	assert.True(t, cfg.Reconnect.Enabled)
}

func TestLoadClientConfig_FxtunnelYamlPriority(t *testing.T) {
	dir := t.TempDir()

	clientYaml := filepath.Join(dir, "client.yaml")
	err := os.WriteFile(clientYaml, []byte(`
server:
  address: "server1:4443"
tunnels:
  - name: "from-client"
    type: "http"
    local_port: 1111
`), 0600)
	require.NoError(t, err)

	fxtunnelYaml := filepath.Join(dir, "fxtunnel.yaml")
	err = os.WriteFile(fxtunnelYaml, []byte(`
tunnels:
  - name: "from-fxtunnel"
    type: "http"
    local_port: 2222
`), 0600)
	require.NoError(t, err)

	t.Chdir(dir)

	cfg, err := LoadClientConfig("")
	require.NoError(t, err)
	assert.Equal(t, 2222, cfg.Tunnels[0].LocalPort)
	assert.Equal(t, "from-fxtunnel", cfg.Tunnels[0].Name)
}

func TestLoadClientConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "client.yaml")
	yaml := `
server:
  address: "myserver.com:5555"
  token: "sk_test123"
tunnels:
  - name: web
    type: http
    local_port: 8080
    subdomain: myapp
  - name: ssh
    type: tcp
    local_port: 22
reconnect:
  enabled: false
`
	require.NoError(t, os.WriteFile(cfgFile, []byte(yaml), 0600))

	cfg, err := LoadClientConfig(cfgFile)
	require.NoError(t, err)
	assert.Equal(t, "myserver.com:5555", cfg.Server.Address)
	assert.Equal(t, "sk_test123", cfg.Server.Token)
	require.Len(t, cfg.Tunnels, 2)
	assert.Equal(t, "http", cfg.Tunnels[0].Type)
	assert.Equal(t, 8080, cfg.Tunnels[0].LocalPort)
	assert.Equal(t, "myapp", cfg.Tunnels[0].Subdomain)
	assert.Equal(t, "tcp", cfg.Tunnels[1].Type)
	assert.False(t, cfg.Reconnect.Enabled)
}
