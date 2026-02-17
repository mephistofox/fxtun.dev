package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckIPAllowlist_NoRestriction(t *testing.T) {
	tunnel := &Tunnel{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	ok := checkIPAllowlist(w, req, tunnel)

	assert.True(t, ok)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCheckIPAllowlist_AllowedExactIP(t *testing.T) {
	tunnel := &Tunnel{
		AllowedIPs: []net.IP{net.ParseIP("192.168.1.100")},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	ok := checkIPAllowlist(w, req, tunnel)

	assert.True(t, ok)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCheckIPAllowlist_AllowedCIDR(t *testing.T) {
	_, cidr, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)

	tunnel := &Tunnel{
		AllowedNets: []*net.IPNet{cidr},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.42.0.5:9999"
	w := httptest.NewRecorder()

	ok := checkIPAllowlist(w, req, tunnel)

	assert.True(t, ok)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCheckIPAllowlist_Blocked(t *testing.T) {
	tunnel := &Tunnel{
		AllowedIPs: []net.IP{net.ParseIP("192.168.1.100")},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()

	ok := checkIPAllowlist(w, req, tunnel)

	assert.False(t, ok)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCheckIPAllowlist_XRealIP(t *testing.T) {
	tunnel := &Tunnel{
		AllowedIPs: []net.IP{net.ParseIP("203.0.113.50")},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.50")
	w := httptest.NewRecorder()

	ok := checkIPAllowlist(w, req, tunnel)

	assert.True(t, ok)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCheckIPAllowlist_IPv6MappedIPv4(t *testing.T) {
	// Client sends an IPv6-mapped IPv4 address, but the allowlist has plain IPv4
	tunnel := &Tunnel{
		AllowedIPs: []net.IP{net.ParseIP("192.168.1.1")},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[::ffff:192.168.1.1]:12345"
	w := httptest.NewRecorder()

	ok := checkIPAllowlist(w, req, tunnel)

	assert.True(t, ok)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIsIPAllowed_NoRestriction(t *testing.T) {
	tunnel := &Tunnel{}
	assert.True(t, isIPAllowed(net.ParseIP("1.2.3.4"), tunnel))
}

func TestIsIPAllowed_ExactMatch(t *testing.T) {
	tunnel := &Tunnel{
		AllowedIPs: []net.IP{net.ParseIP("10.0.0.1")},
	}
	assert.True(t, isIPAllowed(net.ParseIP("10.0.0.1"), tunnel))
	assert.False(t, isIPAllowed(net.ParseIP("10.0.0.2"), tunnel))
}

func TestIsIPAllowed_CIDRMatch(t *testing.T) {
	_, cidr, _ := net.ParseCIDR("192.168.0.0/16")
	tunnel := &Tunnel{
		AllowedNets: []*net.IPNet{cidr},
	}
	assert.True(t, isIPAllowed(net.ParseIP("192.168.1.1"), tunnel))
	assert.False(t, isIPAllowed(net.ParseIP("10.0.0.1"), tunnel))
}

func TestParseAllowIPs_Valid(t *testing.T) {
	raw := []string{"192.168.1.1", "10.0.0.0/8", "2001:db8::1", "fd00::/64"}

	ips, nets, err := parseAllowIPs(raw)

	require.NoError(t, err)
	assert.Len(t, ips, 2)
	assert.Len(t, nets, 2)

	// Check IPs
	assert.True(t, ips[0].Equal(net.ParseIP("192.168.1.1")))
	assert.True(t, ips[1].Equal(net.ParseIP("2001:db8::1")))

	// Check CIDRs
	assert.Equal(t, "10.0.0.0/8", nets[0].String())
	assert.Equal(t, "fd00::/64", nets[1].String())
}

func TestParseAllowIPs_Invalid(t *testing.T) {
	tests := []struct {
		name string
		raw  []string
	}{
		{"invalid IP", []string{"not-an-ip"}},
		{"invalid CIDR", []string{"10.0.0.0/99"}},
		{"partial IP", []string{"192.168"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseAllowIPs(tt.raw)
			assert.Error(t, err)
		})
	}
}
