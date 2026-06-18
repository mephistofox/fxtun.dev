package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestDaemonAPILifecycle(t *testing.T) {
	mgr := &mockTunnelManager{}
	api := NewAPI(mgr, "test-server:4443", testToken)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: api, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	base := fmt.Sprintf("http://%s", ln.Addr().String())
	client := &http.Client{Timeout: 2 * time.Second}

	do := func(method, path string, body io.Reader) *http.Response {
		t.Helper()
		req, err := http.NewRequest(method, base+path, body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+testToken)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	// 1. GET /status — running, 0 tunnels
	resp := do(http.MethodGet, "/status", nil)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Running {
		t.Fatal("expected running=true")
	}
	if len(status.Tunnels) != 0 {
		t.Fatalf("expected 0 tunnels, got %d", len(status.Tunnels))
	}

	// 2. POST /tunnels — add a tunnel
	body, _ := json.Marshal(AddTunnelRequest{
		Type:      "http",
		LocalPort: 3000,
	})
	resp2 := do(http.MethodPost, "/tunnels", bytes.NewReader(body))
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
	var info TunnelInfo
	if err := json.NewDecoder(resp2.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.Type != "http" {
		t.Fatalf("expected type http, got %s", info.Type)
	}
	if info.LocalPort != 3000 {
		t.Fatalf("expected local_port 3000, got %d", info.LocalPort)
	}

	// 3. GET /status — 1 tunnel
	resp3 := do(http.MethodGet, "/status", nil)
	defer resp3.Body.Close()
	var status2 StatusResponse
	if err := json.NewDecoder(resp3.Body).Decode(&status2); err != nil {
		t.Fatal(err)
	}
	if len(status2.Tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(status2.Tunnels))
	}

	// 4. POST /shutdown
	resp4 := do(http.MethodPost, "/shutdown", nil)
	defer resp4.Body.Close()
	if resp4.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp4.StatusCode)
	}

	// 5. Wait on Done()
	select {
	case <-api.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("expected done channel to close after shutdown")
	}
}
