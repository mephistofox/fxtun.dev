package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/config"
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
	Type       string `json:"type"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
	Name       string `json:"name,omitempty"`
}

type API struct {
	mgr     TunnelManager
	server  string
	started time.Time
	done    chan struct{}
	mux     *http.ServeMux
}

func NewAPI(mgr TunnelManager, server string) *API {
	a := &API{
		mgr:     mgr,
		server:  server,
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

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
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
		Name:       req.Name,
		Type:       req.Type,
		LocalPort:  req.LocalPort,
		RemotePort: req.RemotePort,
		Subdomain:  req.Subdomain,
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
