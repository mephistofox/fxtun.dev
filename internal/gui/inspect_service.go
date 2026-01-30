package gui

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// InspectService handles traffic inspection operations
type InspectService struct {
	app       *App
	log       zerolog.Logger
	cancelSSE map[string]func() // tunnelID -> cancel function
}

// NewInspectService creates a new inspect service
func NewInspectService(app *App) *InspectService {
	return &InspectService{
		app:       app,
		log:       app.log.With().Str("service", "inspect").Logger(),
		cancelSSE: make(map[string]func()),
	}
}

// ExchangeSummary represents a compact exchange for listing
type ExchangeSummary struct {
	ID               string `json:"id"`
	TunnelID         string `json:"tunnel_id"`
	Timestamp        string `json:"timestamp"`
	DurationNs       int64  `json:"duration_ns"`
	Method           string `json:"method"`
	Path             string `json:"path"`
	Host             string `json:"host"`
	StatusCode       int    `json:"status_code"`
	RequestBodySize  int64  `json:"request_body_size"`
	ResponseBodySize int64  `json:"response_body_size"`
	RemoteAddr       string `json:"remote_addr"`
}

// CapturedExchange represents a full exchange with bodies
type CapturedExchange struct {
	ExchangeSummary
	RequestHeaders  map[string][]string `json:"request_headers"`
	RequestBody     interface{}         `json:"request_body"`
	ResponseHeaders map[string][]string `json:"response_headers"`
	ResponseBody    interface{}         `json:"response_body"`
}

// ExchangeListResponse represents list response
type ExchangeListResponse struct {
	Exchanges []*ExchangeSummary `json:"exchanges"`
	Total     int                `json:"total"`
}

// List returns exchanges for a tunnel
func (s *InspectService) List(tunnelID string, offset, limit int) (*ExchangeListResponse, error) {
	if s.app.authToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL(fmt.Sprintf("/api/tunnels/%s/inspect?offset=%d&limit=%d", tunnelID, offset, limit))
	body, statusCode, err := s.app.api.Get(url)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		return nil, fmt.Errorf("%s", errResp.Error)
	}

	var result ExchangeListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get returns a single exchange with full bodies
func (s *InspectService) Get(tunnelID, exchangeID string) (*CapturedExchange, error) {
	if s.app.authToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL(fmt.Sprintf("/api/tunnels/%s/inspect/%s", tunnelID, exchangeID))
	body, statusCode, err := s.app.api.Get(url)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		return nil, fmt.Errorf("%s", errResp.Error)
	}

	var result CapturedExchange
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Clear clears the inspection buffer for a tunnel
func (s *InspectService) Clear(tunnelID string) error {
	if s.app.authToken == "" {
		return fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL(fmt.Sprintf("/api/tunnels/%s/inspect", tunnelID))
	_, statusCode, err := s.app.api.Delete(url)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("failed to clear exchanges")
	}
	return nil
}

// Subscribe starts SSE streaming for a tunnel and emits Wails events
func (s *InspectService) Subscribe(tunnelID string) error {
	// Cancel existing subscription for this tunnel
	s.Unsubscribe(tunnelID)

	if s.app.authToken == "" {
		return fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL(fmt.Sprintf("/api/tunnels/%s/inspect/stream?token=%s", tunnelID, s.app.authToken))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("SSE connection failed: %d", resp.StatusCode)
	}

	done := make(chan struct{})
	s.cancelSSE[tunnelID] = func() {
		close(done)
		resp.Body.Close()
	}

	go func() {
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-done:
				return
			default:
			}

			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				var ex ExchangeSummary
				if err := json.Unmarshal([]byte(data), &ex); err == nil {
					runtime.EventsEmit(s.app.ctx, "inspect_exchange", ex)
				}
			}
		}
		// Connection closed
		runtime.EventsEmit(s.app.ctx, "inspect_disconnected", tunnelID)
	}()

	return nil
}

// Unsubscribe stops SSE streaming for a tunnel
func (s *InspectService) Unsubscribe(tunnelID string) {
	if cancel, ok := s.cancelSSE[tunnelID]; ok {
		cancel()
		delete(s.cancelSSE, tunnelID)
	}
}
