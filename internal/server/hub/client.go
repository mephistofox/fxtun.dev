package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// ClientAuthInfo holds the result of a token verification against the hub.
type ClientAuthInfo struct {
	Valid            bool  `json:"valid"`
	UserID           int64 `json:"user_id"`
	MaxTunnels       int   `json:"max_tunnels"`
	MaxDataSessions  int   `json:"max_data_sessions"`
	IsAdmin          bool  `json:"is_admin"`
	InspectorEnabled bool  `json:"inspector_enabled"`
	Error            string `json:"error,omitempty"`
}

// Client communicates with the hub API from an edge node.
type Client struct {
	hubURL string
	token  string
	nodeID string
	http   *http.Client
	log    zerolog.Logger
}

// NewClient creates a new HubClient.
func NewClient(hubURL, token string, log zerolog.Logger) *Client {
	return &Client{
		hubURL: hubURL,
		token:  token,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: log.With().Str("component", "hub-client").Logger(),
	}
}

// NodeID returns the node's assigned ID after registration.
func (h *Client) NodeID() string {
	return h.nodeID
}

// registerRequest is the body for POST /api/nodes/register.
type registerRequest struct {
	Name       string `json:"name"`
	Region     string `json:"region"`
	PublicAddr string `json:"public_addr"`
	HTTPAddr   string `json:"http_addr"`
	Version    string `json:"version"`
}

// registerResponse is returned by the hub on successful registration.
type registerResponse struct {
	NodeID string `json:"node_id"`
	Status string `json:"status"`
}

// Register registers this node with the hub. Returns assigned nodeID.
func (h *Client) Register(name, region, publicAddr, httpAddr, version string) (string, error) {
	body := registerRequest{
		Name:       name,
		Region:     region,
		PublicAddr: publicAddr,
		HTTPAddr:   httpAddr,
		Version:    version,
	}

	var resp registerResponse
	if err := h.doJSON(http.MethodPost, "/api/nodes/register", body, &resp); err != nil {
		return "", fmt.Errorf("register node: %w", err)
	}

	h.nodeID = resp.NodeID
	h.log.Info().
		Str("node_id", resp.NodeID).
		Str("status", resp.Status).
		Msg("Registered with hub")

	return resp.NodeID, nil
}

// heartbeatRequest is the body for POST /api/nodes/heartbeat.
type heartbeatRequest struct {
	NodeID      string `json:"node_id"`
	TunnelCount int    `json:"tunnel_count"`
	ClientCount int    `json:"client_count"`
}

// Heartbeat sends a heartbeat to the hub with current stats.
func (h *Client) Heartbeat(tunnelCount, clientCount int) error {
	body := heartbeatRequest{
		NodeID:      h.nodeID,
		TunnelCount: tunnelCount,
		ClientCount: clientCount,
	}
	return h.doJSON(http.MethodPost, "/api/nodes/heartbeat", body, nil)
}

// verifyTokenRequest is the body for POST /api/internal/auth/verify.
type verifyTokenRequest struct {
	Token string `json:"token"`
}

// VerifyClientToken asks the hub to validate a client's tunnel token.
func (h *Client) VerifyClientToken(token string) (*ClientAuthInfo, error) {
	body := verifyTokenRequest{Token: token}
	var info ClientAuthInfo
	if err := h.doJSON(http.MethodPost, "/api/internal/auth/verify", body, &info); err != nil {
		return nil, fmt.Errorf("verify client token: %w", err)
	}
	return &info, nil
}

// StartHeartbeatLoop runs periodic heartbeat in background.
// statsFunc is called each tick to get current tunnel and client counts.
func (h *Client) StartHeartbeatLoop(ctx context.Context, interval time.Duration, statsFunc func() (tunnels, clients int)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tunnels, clients := statsFunc()
			if err := h.Heartbeat(tunnels, clients); err != nil {
				h.log.Warn().Err(err).Msg("Hub heartbeat failed")
			}
		}
	}
}

// doJSON performs an HTTP request with JSON body and decodes the response.
func (h *Client) doJSON(method, path string, reqBody interface{}, respBody interface{}) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, h.hubURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.token)

	resp, err := h.http.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("hub returned %d: %s", resp.StatusCode, string(body))
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// TLSCertResponse holds the TLS certificate and key from the hub.
type TLSCertResponse struct {
	CertPEM string `json:"cert_pem"`
	KeyPEM  string `json:"key_pem"`
}

// FetchTLSCert retrieves the hub's TLS certificate for use by approved edge nodes.
func (h *Client) FetchTLSCert() (*TLSCertResponse, error) {
	var resp TLSCertResponse
	if err := h.doJSON("GET", "/api/internal/tls-cert?node_id="+h.nodeID, nil, &resp); err != nil {
		return nil, fmt.Errorf("fetch TLS cert: %w", err)
	}
	if resp.CertPEM == "" || resp.KeyPEM == "" {
		return nil, fmt.Errorf("hub returned empty TLS cert")
	}
	return &resp, nil
}
