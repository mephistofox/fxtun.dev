package gui

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	if s.app.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	if s.app.authToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL("/api/domains")
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

	var result DomainsListResponse
	if err := json.Unmarshal(body, &result); err != nil {
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
	if s.app.authToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL(fmt.Sprintf("/api/domains/check/%s", subdomain))
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
	if s.app.authToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL("/api/domains")
	reqBody, _ := json.Marshal(map[string]string{"subdomain": subdomain})

	body, statusCode, err := s.app.api.Post(url, reqBody)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusCreated && statusCode != http.StatusOK {
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
	if s.app.authToken == "" {
		return fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL(fmt.Sprintf("/api/domains/%d", id))
	body, statusCode, err := s.app.api.Delete(url)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		return fmt.Errorf("%s", errResp.Error)
	}

	return nil
}
