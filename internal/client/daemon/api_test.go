package daemon

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/config"
)

const testToken = "test-session-token"

// authedReq builds a request that passes the daemon guard: loopback Host and a
// valid bearer token.
func authedReq(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Host = "127.0.0.1:7070"
	req.Header.Set("Authorization", "Bearer "+testToken)
	return req
}

type mockTunnelManager struct {
	tunnels []TunnelInfo
}

func (m *mockTunnelManager) GetTunnels() []TunnelInfo {
	return m.tunnels
}

func (m *mockTunnelManager) RequestTunnel(cfg config.TunnelConfig) (TunnelInfo, error) {
	info := TunnelInfo{
		ID:        cfg.Name,
		Type:      cfg.Type,
		LocalPort: cfg.LocalPort,
		Subdomain: cfg.Subdomain,
	}
	m.tunnels = append(m.tunnels, info)
	return info, nil
}

func (m *mockTunnelManager) CloseTunnel(id string) error {
	for i, t := range m.tunnels {
		if t.ID == id {
			m.tunnels = append(m.tunnels[:i], m.tunnels[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockTunnelManager) Shutdown() {}

func TestAPIStatus(t *testing.T) {
	mgr := &mockTunnelManager{
		tunnels: []TunnelInfo{
			{ID: "t1", Type: "http", LocalPort: 3000},
			{ID: "t2", Type: "tcp", LocalPort: 22},
		},
	}
	api := NewAPI(mgr, "example.com:4443", testToken)

	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, authedReq(http.MethodGet, "/status", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp StatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Running {
		t.Fatal("expected running=true")
	}
	if len(resp.Tunnels) != 2 {
		t.Fatalf("expected 2 tunnels, got %d", len(resp.Tunnels))
	}
	if resp.Server != "example.com:4443" {
		t.Fatalf("expected server example.com:4443, got %s", resp.Server)
	}
}

func TestAPIAddTunnel(t *testing.T) {
	mgr := &mockTunnelManager{}
	api := NewAPI(mgr, "example.com:4443", testToken)

	body, _ := json.Marshal(AddTunnelRequest{
		Type:      "http",
		LocalPort: 8080,
		Subdomain: "myapp",
	})
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, authedReq(http.MethodPost, "/tunnels", bytes.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var info TunnelInfo
	if err := json.NewDecoder(rec.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.ID != "http-8080" {
		t.Fatalf("expected auto-generated name http-8080, got %s", info.ID)
	}
	if info.Type != "http" {
		t.Fatalf("expected type http, got %s", info.Type)
	}
}

func TestAPIRemoveTunnel(t *testing.T) {
	mgr := &mockTunnelManager{
		tunnels: []TunnelInfo{
			{ID: "t1", Type: "http", LocalPort: 3000},
		},
	}
	api := NewAPI(mgr, "example.com:4443", testToken)

	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, authedReq(http.MethodDelete, "/tunnels/t1", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(mgr.tunnels) != 0 {
		t.Fatalf("expected 0 tunnels after removal, got %d", len(mgr.tunnels))
	}
}

func TestAPIShutdown(t *testing.T) {
	mgr := &mockTunnelManager{}
	api := NewAPI(mgr, "example.com:4443", testToken)

	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, authedReq(http.MethodPost, "/shutdown", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	select {
	case <-api.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("expected done channel to close after shutdown")
	}
}

// TestAPIGuard verifies the daemon rejects unauthenticated, cross-site, and
// non-loopback (DNS-rebinding) requests before reaching a handler.
func TestAPIGuard(t *testing.T) {
	newReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/status", nil)
		r.Host = "127.0.0.1:7070"
		r.Header.Set("Authorization", "Bearer "+testToken)
		return r
	}

	cases := []struct {
		name   string
		mutate func(*http.Request)
		want   int
	}{
		{"valid", func(*http.Request) {}, http.StatusOK},
		{"missing token", func(r *http.Request) { r.Header.Del("Authorization") }, http.StatusUnauthorized},
		{"wrong token", func(r *http.Request) { r.Header.Set("Authorization", "Bearer nope") }, http.StatusUnauthorized},
		{"empty bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer ") }, http.StatusUnauthorized},
		{"origin header (CSRF)", func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") }, http.StatusForbidden},
		{"referer header", func(r *http.Request) { r.Header.Set("Referer", "http://evil.example/") }, http.StatusForbidden},
		{"non-loopback host (DNS rebinding)", func(r *http.Request) { r.Host = "evil.example:7070" }, http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := NewAPI(&mockTunnelManager{}, "example.com:4443", testToken)
			req := newReq()
			tc.mutate(req)
			rec := httptest.NewRecorder()
			api.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, rec.Code)
			}
		})
	}
}

// An empty server-side token must fail closed (reject everything).
func TestAPIGuardEmptyTokenFailsClosed(t *testing.T) {
	api := NewAPI(&mockTunnelManager{}, "example.com:4443", "")
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Host = "127.0.0.1:7070"
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for empty server token, got %d", rec.Code)
	}
}
