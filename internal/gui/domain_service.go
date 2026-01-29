package gui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

// DomainService handles domain operations
type DomainService struct {
	app *App
	log zerolog.Logger
}

// NewDomainService creates a new domain service
func NewDomainService(app *App) *DomainService {
	return &DomainService{
		app: app,
		log: app.log.With().Str("service", "domain").Logger(),
	}
}

// getAPIHost returns the hostname without port for API calls
func (s *DomainService) getAPIHost() string {
	addr := s.app.serverAddress
	if idx := strings.Index(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// Domain represents a reserved domain
type Domain struct {
	ID        int64  `json:"id"`
	Subdomain string `json:"subdomain"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

// DomainsListResponse represents the list of domains response
type DomainsListResponse struct {
	Domains    []*Domain `json:"domains"`
	Total      int       `json:"total"`
	MaxDomains int       `json:"max_domains"`
}

// DomainCheckResponse represents domain availability check
type DomainCheckResponse struct {
	Subdomain string `json:"subdomain"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// List returns all reserved domains for the current user
func (s *DomainService) List() (*DomainsListResponse, error) {
	s.log.Debug().
		Bool("client_nil", s.app.client == nil).
		Str("server_address", s.app.serverAddress).
		Str("auth_token_prefix", func() string {
			if len(s.app.authToken) > 20 {
				return s.app.authToken[:20] + "..."
			}
			return s.app.authToken
		}()).
		Msg("List domains called")

	if s.app.client == nil {
		s.log.Error().Msg("Client is nil - not connected")
		return nil, fmt.Errorf("not connected")
	}

	token := s.app.authToken
	if token == "" {
		s.log.Error().Msg("Auth token is empty")
		return nil, fmt.Errorf("not authenticated")
	}

	host := s.getAPIHost()
	url := fmt.Sprintf("https://%s/api/domains", host)
	s.log.Debug().Str("url", url).Msg("Making API request")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to create request")
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.log.Error().Err(err).Msg("HTTP request failed")
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s.log.Debug().Int("status", resp.StatusCode).Str("body", string(body)).Msg("API response")

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		s.log.Error().Str("error", errResp.Error).Msg("API returned error")
		return nil, fmt.Errorf("%s", errResp.Error)
	}

	var result DomainsListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		s.log.Error().Err(err).Msg("Failed to parse response")
		return nil, err
	}

	s.log.Info().Int("count", len(result.Domains)).Msg("Domains loaded successfully")
	return &result, nil
}

// Check checks if a subdomain is available
func (s *DomainService) Check(subdomain string) (*DomainCheckResponse, error) {
	if s.app.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	token := s.app.authToken
	if token == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	host := s.getAPIHost()
	url := fmt.Sprintf("https://%s/api/domains/check/%s", host, subdomain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		return nil, fmt.Errorf("%s", errResp.Error)
	}

	var result DomainCheckResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Reserve reserves a new subdomain
func (s *DomainService) Reserve(subdomain string) (*Domain, error) {
	if s.app.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	token := s.app.authToken
	if token == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	host := s.getAPIHost()
	url := fmt.Sprintf("https://%s/api/domains", host)

	reqBody, _ := json.Marshal(map[string]string{"subdomain": subdomain})
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		return nil, fmt.Errorf("%s", errResp.Error)
	}

	var result Domain
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Release releases a reserved domain
func (s *DomainService) Release(id int64) error {
	if s.app.client == nil {
		return fmt.Errorf("not connected")
	}

	token := s.app.authToken
	if token == "" {
		return fmt.Errorf("not authenticated")
	}

	host := s.getAPIHost()
	url := fmt.Sprintf("https://%s/api/domains/%d", host, id)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		return fmt.Errorf("%s", errResp.Error)
	}

	return nil
}
