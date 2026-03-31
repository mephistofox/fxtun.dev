package core

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test 1: Basic Auth blocks unauthenticated requests
func TestHTTPRouter_BasicAuth_Blocks(t *testing.T) {
	hash := hashCredentials(t, "user", "pass")
	tunnel := &Tunnel{BasicAuthHash: hash}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	ok := checkBasicAuth(w, req, tunnel)

	assert.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, `Basic realm="fxTunnel"`, w.Header().Get("WWW-Authenticate"))
}

// Test 2: Basic Auth passes with correct credentials
func TestHTTPRouter_BasicAuth_Passes(t *testing.T) {
	hash := hashCredentials(t, "user", "pass")
	tunnel := &Tunnel{BasicAuthHash: hash}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("user", "pass")
	w := httptest.NewRecorder()

	ok := checkBasicAuth(w, req, tunnel)

	assert.True(t, ok)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("WWW-Authenticate"))
}

// Test 3: IP Allowlist blocks non-allowed IPs
func TestHTTPRouter_IPAllowlist_Blocks(t *testing.T) {
	tunnel := &Tunnel{
		AllowedIPs: []net.IP{net.ParseIP("10.0.0.1")},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	w := httptest.NewRecorder()

	ok := checkIPAllowlist(w, req, tunnel)

	assert.False(t, ok)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Test 4: IP Allowlist + Basic Auth ordering
// The IP check must run BEFORE the auth check: a blocked IP gets 403, not 401,
// even if valid credentials are provided.
func TestHTTPRouter_SecurityChainOrder(t *testing.T) {
	hash := hashCredentials(t, "user", "pass")
	tunnel := &Tunnel{
		BasicAuthHash: hash,
		AllowedIPs:   []net.IP{net.ParseIP("10.0.0.1")},
	}

	// Request comes from a non-allowlisted IP but carries valid auth credentials
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	req.SetBasicAuth("user", "pass")
	w := httptest.NewRecorder()

	// Simulate the middleware chain as done in ServeHTTP: IP check first
	if !checkIPAllowlist(w, req, tunnel) {
		// IP was blocked — must be 403, not 401
		assert.Equal(t, http.StatusForbidden, w.Code)
		// Auth check must never have been reached
		assert.Empty(t, w.Header().Get("WWW-Authenticate"), "WWW-Authenticate must not be set when IP is blocked first")
		return
	}

	// If we reach here, IP was allowed — run auth check
	_ = checkBasicAuth(w, req, tunnel)
	t.Fatal("expected IP check to block the request before auth check was reached")
}

// Test 5: IP Allowlist with CIDR range
func TestHTTPRouter_IPAllowlist_CIDR(t *testing.T) {
	_, cidr, err := net.ParseCIDR("192.168.0.0/16")
	require.NoError(t, err)

	tunnel := &Tunnel{
		AllowedNets: []*net.IPNet{cidr},
	}

	tests := []struct {
		name       string
		remoteAddr string
		wantOk     bool
		wantStatus int
	}{
		{
			name:       "IP inside CIDR passes",
			remoteAddr: "192.168.1.50:9999",
			wantOk:     true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "IP outside CIDR blocked",
			remoteAddr: "10.0.0.1:9999",
			wantOk:     false,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			ok := checkIPAllowlist(w, req, tunnel)

			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// Test 6: Auto-close timer duration parsing
func TestParseTunnelDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
		err   bool
	}{
		{"30m", 30 * time.Minute, false},
		{"2h", 2 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"0.5d", 12 * time.Hour, false},
		{"", 0, false},
		{"invalid", 0, true},
		{"-1h", 0, true},
		{"0h", 0, true},
		{"0d", 0, true},
		{"7d", 7 * 24 * time.Hour, false},
		{"90s", 90 * time.Second, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseTunnelDuration(tt.input)
			if tt.err {
				assert.Error(t, err, "expected error for input %q", tt.input)
			} else {
				require.NoError(t, err, "unexpected error for input %q", tt.input)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// Test 7: All security features together — verify correct middleware chain order
func TestHTTPRouter_AllSecurityFeatures(t *testing.T) {
	hash := hashCredentials(t, "admin", "secret")
	_, cidr, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)

	tunnel := &Tunnel{
		BasicAuthHash: hash,
		AllowedIPs:   []net.IP{net.ParseIP("10.1.2.3")},
		AllowedNets:  []*net.IPNet{cidr},
		AutoClose:    30 * time.Minute,
		MaxLifetime:  2 * time.Hour,
	}

	t.Run("blocked IP gets 403 regardless of auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "203.0.113.99:5555"
		req.SetBasicAuth("admin", "secret") // valid credentials
		w := httptest.NewRecorder()

		// IP check runs first
		if !checkIPAllowlist(w, req, tunnel) {
			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Empty(t, w.Header().Get("WWW-Authenticate"))
			return
		}
		t.Fatal("expected IP check to block the request")
	})

	t.Run("allowed IP without auth gets 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.1.2.3:5555" // in AllowedIPs
		// No auth header
		w := httptest.NewRecorder()

		ipOk := checkIPAllowlist(w, req, tunnel)
		require.True(t, ipOk, "IP should be allowed")

		authOk := checkBasicAuth(w, req, tunnel)
		assert.False(t, authOk)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, `Basic realm="fxTunnel"`, w.Header().Get("WWW-Authenticate"))
	})

	t.Run("allowed IP with correct auth passes both checks", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.5.6.7:5555" // in CIDR 10.0.0.0/8
		req.SetBasicAuth("admin", "secret")
		w := httptest.NewRecorder()

		ipOk := checkIPAllowlist(w, req, tunnel)
		assert.True(t, ipOk)

		authOk := checkBasicAuth(w, req, tunnel)
		assert.True(t, authOk)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Test 8: parseAllowIPs integration with real Tunnel allowlist checks
func TestParseAllowIPs_Integration(t *testing.T) {
	raw := []string{"192.168.1.0/24", "10.0.0.1", "2001:db8::/32"}

	ips, nets, err := parseAllowIPs(raw)
	require.NoError(t, err)
	require.Len(t, ips, 1)
	require.Len(t, nets, 2)

	tunnel := &Tunnel{
		AllowedIPs:  ips,
		AllowedNets: nets,
	}

	tests := []struct {
		name       string
		remoteAddr string
		wantOk     bool
	}{
		// IPv4 exact match
		{"exact IPv4 match", "10.0.0.1:1234", true},
		// IPv4 in CIDR
		{"IPv4 in 192.168.1.0/24", "192.168.1.100:1234", true},
		// IPv4 outside all ranges
		{"IPv4 not in allowlist", "172.16.0.1:1234", false},
		// IPv6 in CIDR
		{"IPv6 in 2001:db8::/32", "[2001:db8::1]:1234", true},
		// IPv6 outside all ranges
		{"IPv6 not in allowlist", "[fd00::1]:1234", false},
		// Address at CIDR boundary
		{"first host in /24", "192.168.1.1:1234", true},
		// Address just outside /24
		{"IP in adjacent /24", "192.168.2.1:1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			ok := checkIPAllowlist(w, req, tunnel)

			assert.Equal(t, tt.wantOk, ok, "remoteAddr=%s", tt.remoteAddr)
			if tt.wantOk {
				assert.Equal(t, http.StatusOK, w.Code)
			} else {
				assert.Equal(t, http.StatusForbidden, w.Code)
			}
		})
	}
}
