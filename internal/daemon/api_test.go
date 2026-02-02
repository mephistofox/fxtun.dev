package daemon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mephistofox/fxtunnel/internal/config"
)

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
	api := NewAPI(mgr, "example.com:4443")

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

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
	api := NewAPI(mgr, "example.com:4443")

	body, _ := json.Marshal(AddTunnelRequest{
		Type:      "http",
		LocalPort: 8080,
		Subdomain: "myapp",
	})
	req := httptest.NewRequest(http.MethodPost, "/tunnels", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

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
	api := NewAPI(mgr, "example.com:4443")

	req := httptest.NewRequest(http.MethodDelete, "/tunnels/t1", nil)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(mgr.tunnels) != 0 {
		t.Fatalf("expected 0 tunnels after removal, got %d", len(mgr.tunnels))
	}
}

func TestAPIShutdown(t *testing.T) {
	mgr := &mockTunnelManager{}
	api := NewAPI(mgr, "example.com:4443")

	req := httptest.NewRequest(http.MethodPost, "/shutdown", nil)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	select {
	case <-api.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("expected done channel to close after shutdown")
	}
}
