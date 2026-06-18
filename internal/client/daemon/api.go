package daemon

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mephistofox/fxtunnel/internal/config"
)

type TunnelInfo struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
	URL        string `json:"url,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
}

type TunnelManager interface {
	GetTunnels() []TunnelInfo
	RequestTunnel(cfg config.TunnelConfig) (TunnelInfo, error)
	CloseTunnel(id string) error
	Shutdown()
}

type StatusResponse struct {
	Running bool         `json:"running"`
	PID     int          `json:"pid"`
	Server  string       `json:"server"`
	Uptime  string       `json:"uptime"`
	Tunnels []TunnelInfo `json:"tunnels"`
}

type AddTunnelRequest struct {
	Type          string   `json:"type"`
	LocalPort     int      `json:"local_port"`
	RemotePort    int      `json:"remote_port,omitempty"`
	Subdomain     string   `json:"subdomain,omitempty"`
	Name          string   `json:"name,omitempty"`
	BasicAuthHash string   `json:"basic_auth_hash,omitempty"`
	AllowIPs      []string `json:"allow_ips,omitempty"`
	AutoClose     string   `json:"auto_close,omitempty"`
	MaxLifetime   string   `json:"max_lifetime,omitempty"`
}

type API struct {
	mgr     TunnelManager
	server  string
	token   string
	started time.Time
	done    chan struct{}
	mux     *http.ServeMux
}

// NewAPI builds the local daemon API. token is the per-session bearer token
// that every request must present; it is stored in the 0600 daemon state file
// and shared only with same-machine CLI clients.
func NewAPI(mgr TunnelManager, server, token string) *API {
	a := &API{
		mgr:     mgr,
		server:  server,
		token:   token,
		started: time.Now(),
		done:    make(chan struct{}),
		mux:     http.NewServeMux(),
	}
	a.mux.HandleFunc("GET /status", a.handleStatus)
	a.mux.HandleFunc("POST /tunnels", a.handleAddTunnel)
	a.mux.HandleFunc("DELETE /tunnels/{id}", a.handleRemoveTunnel)
	a.mux.HandleFunc("POST /shutdown", a.handleShutdown)
	return a
}

// ServeHTTP guards every request before dispatching. The daemon listens only
// on loopback, but a malicious web page could still reach it via DNS rebinding
// or a cross-site request, so we (1) require the Host header to be loopback,
// (2) reject any request carrying an Origin/Referer (browsers always attach
// these; the CLI never does), and (3) require the per-session bearer token.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackHost(r.Host) {
		http.Error(w, `{"error":"forbidden host"}`, http.StatusForbidden)
		return
	}
	if r.Header.Get("Origin") != "" || r.Header.Get("Referer") != "" {
		http.Error(w, `{"error":"cross-site request rejected"}`, http.StatusForbidden)
		return
	}
	if !a.authorized(r) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	a.mux.ServeHTTP(w, r)
}

// authorized checks the Authorization header against the session token in
// constant time. An empty server token fails closed.
func (a *API) authorized(r *http.Request) bool {
	if a.token == "" {
		return false
	}
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(h[len(prefix):]), []byte(a.token)) == 1
}

// isLoopbackHost reports whether the request Host targets the local machine.
func isLoopbackHost(host string) bool {
	h := host
	if hostOnly, _, err := net.SplitHostPort(host); err == nil {
		h = hostOnly
	}
	h = strings.TrimSuffix(strings.TrimPrefix(h, "["), "]")
	if strings.EqualFold(h, "localhost") {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func (a *API) Done() <-chan struct{} {
	return a.done
}

func (a *API) handleStatus(w http.ResponseWriter, _ *http.Request) {
	tunnels := a.mgr.GetTunnels()
	if tunnels == nil {
		tunnels = []TunnelInfo{}
	}
	writeJSON(w, http.StatusOK, StatusResponse{
		Running: true,
		PID:     os.Getpid(),
		Server:  a.server,
		Uptime:  time.Since(a.started).Round(time.Second).String(),
		Tunnels: tunnels,
	})
}

func (a *API) handleAddTunnel(w http.ResponseWriter, r *http.Request) {
	var req AddTunnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.Name == "" {
		req.Name = fmt.Sprintf("%s-%d", req.Type, req.LocalPort)
	}
	info, err := a.mgr.RequestTunnel(config.TunnelConfig{
		Name:          req.Name,
		Type:          req.Type,
		LocalPort:     req.LocalPort,
		RemotePort:    req.RemotePort,
		Subdomain:     req.Subdomain,
		BasicAuthHash: req.BasicAuthHash,
		AllowIPs:      req.AllowIPs,
		AutoClose:     req.AutoClose,
		MaxLifetime:   req.MaxLifetime,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (a *API) handleRemoveTunnel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.mgr.CloseTunnel(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) handleShutdown(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	go func() {
		a.mgr.Shutdown()
		close(a.done)
	}()
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
