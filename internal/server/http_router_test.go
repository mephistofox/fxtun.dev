package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/protocol"
)

func newTestRouter(baseDomain string) (*HTTPRouter, *Server) {
	log := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	cfg := &config.ServerConfig{
		Server: config.ServerSettings{
			ControlPort:  14443,
			HTTPPort:     18080,
			TCPPortRange: config.PortRange{Min: 30000, Max: 31000},
			UDPPortRange: config.PortRange{Min: 31001, Max: 32000},
		},
		Domain: config.DomainSettings{
			Base:     baseDomain,
			Wildcard: true,
		},
	}
	srv := New(cfg, log)
	return srv.httpRouter, srv
}

func TestExtractSubdomain(t *testing.T) {
	router, _ := newTestRouter("example.com")

	tests := []struct {
		host string
		want string
	}{
		{"app.example.com", "app"},
		{"app.example.com:8080", "app"},
		{"APP.example.com", "app"},
		{"deep.sub.example.com", "deep.sub"},
		{"example.com", ""},
		{"example.com:8080", ""},
		{"www.example.com", ""},
		{"other.com", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := router.extractSubdomain(tt.host)
			if got != tt.want {
				t.Errorf("extractSubdomain(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestRegisterAndGetTunnel(t *testing.T) {
	router, _ := newTestRouter("example.com")

	tunnel := &Tunnel{
		ID:       "t1",
		ClientID: "c1",
		Type:     protocol.TunnelHTTP,
	}

	if err := router.RegisterTunnel("myapp", tunnel); err != nil {
		t.Fatalf("RegisterTunnel: %v", err)
	}

	got := router.GetTunnel("myapp")
	if got == nil {
		t.Fatal("expected tunnel, got nil")
	}
	if got.ID != "t1" {
		t.Fatalf("expected tunnel ID t1, got %s", got.ID)
	}

	// Case-insensitive lookup
	got = router.GetTunnel("MYAPP")
	if got == nil {
		t.Fatal("expected case-insensitive lookup to succeed")
	}
}

func TestRegisterDuplicateSubdomain(t *testing.T) {
	router, _ := newTestRouter("example.com")

	tunnel := &Tunnel{ID: "t1", ClientID: "c1"}
	if err := router.RegisterTunnel("dup", tunnel); err != nil {
		t.Fatalf("first register: %v", err)
	}

	tunnel2 := &Tunnel{ID: "t2", ClientID: "c2"}
	err := router.RegisterTunnel("dup", tunnel2)
	if err == nil {
		t.Fatal("expected error for duplicate subdomain")
	}
}

func TestUnregisterTunnel(t *testing.T) {
	router, _ := newTestRouter("example.com")

	tunnel := &Tunnel{ID: "t1", ClientID: "c1"}
	_ = router.RegisterTunnel("gone", tunnel)
	router.UnregisterTunnel("gone")

	if got := router.GetTunnel("gone"); got != nil {
		t.Fatal("expected nil after unregister")
	}
}

func TestServeHTTPUnknownSubdomain(t *testing.T) {
	router, srv := newTestRouter("example.com")
	defer srv.cancel()

	req := httptest.NewRequest(http.MethodGet, "http://unknown.example.com/", nil)
	req.Host = "unknown.example.com"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestServeHTTPNoSubdomain(t *testing.T) {
	router, srv := newTestRouter("example.com")
	defer srv.cancel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestShouldShowInterstitial(t *testing.T) {
	router, _ := newTestRouter("example.com")

	tests := []struct {
		name   string
		method string
		accept string
		cookie string
		header string
		want   bool
	}{
		{"GET html", http.MethodGet, "text/html", "", "", true},
		{"POST skips", http.MethodPost, "text/html", "", "", false},
		{"API request", http.MethodGet, "application/json", "", "", false},
		{"skip header", http.MethodGet, "text/html", "", "1", false},
		{"consent cookie", http.MethodGet, "text/html", "1", "", false},
		{"wildcard accept", http.MethodGet, "*/*", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://app.example.com/", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			if tt.header != "" {
				req.Header.Set("X-FxTunnel-Skip-Warning", tt.header)
			}
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "_fxt_consent_app", Value: tt.cookie})
			}
			got := router.shouldShowInterstitial(req, "app")
			if got != tt.want {
				t.Errorf("shouldShowInterstitial = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		accept string
		want   string
	}{
		{"ru-RU,ru;q=0.9,en;q=0.8", "ru"},
		{"en-US,en;q=0.9", "en"},
		{"", "en"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Language", tt.accept)
		got := detectLanguage(req)
		if got != tt.want {
			t.Errorf("detectLanguage(%q) = %q, want %q", tt.accept, got, tt.want)
		}
	}
}

func TestIsUpgradeRequest(t *testing.T) {
	tests := []struct {
		conn string
		want bool
	}{
		{"Upgrade", true},
		{"keep-alive, upgrade", true},
		{"keep-alive", false},
		{"", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tt.conn != "" {
			req.Header.Set("Connection", tt.conn)
		}
		if got := isUpgradeRequest(req); got != tt.want {
			t.Errorf("isUpgradeRequest(Connection: %q) = %v, want %v", tt.conn, got, tt.want)
		}
	}
}
