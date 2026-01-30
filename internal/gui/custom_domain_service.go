package gui

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
)

// CustomDomainService handles custom domain operations via the server API.
type CustomDomainService struct {
	app *App
	log zerolog.Logger
}

// NewCustomDomainService creates a new custom domain service.
func NewCustomDomainService(app *App) *CustomDomainService {
	return &CustomDomainService{
		app: app,
		log: app.log.With().Str("service", "custom_domain").Logger(),
	}
}

// CustomDomainInfo represents a custom domain entry.
type CustomDomainInfo struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id"`
	Domain          string `json:"domain"`
	TargetSubdomain string `json:"target_subdomain"`
	Verified        bool   `json:"verified"`
	VerifiedAt      string `json:"verified_at,omitempty"`
	CreatedAt       string `json:"created_at"`
}

// CustomDomainListResult contains the list of custom domains.
type CustomDomainListResult struct {
	Domains    []*CustomDomainInfo `json:"domains"`
	Total      int                 `json:"total"`
	MaxDomains int                 `json:"max_domains"`
	BaseDomain string              `json:"base_domain"`
	ServerIP   string              `json:"server_ip"`
}

// VerifyResult contains the result of a domain verification.
type VerifyResult struct {
	Verified bool   `json:"verified"`
	Error    string `json:"error,omitempty"`
	Expected string `json:"expected,omitempty"`
}

// List returns all custom domains for the current user.
func (s *CustomDomainService) List() (*CustomDomainListResult, error) {
	if s.app.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	if s.app.authToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL("/api/custom-domains")
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

	var result CustomDomainListResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	s.log.Info().Int("count", len(result.Domains)).Msg("Custom domains loaded")
	return &result, nil
}

// Add adds a new custom domain.
func (s *CustomDomainService) Add(domain, targetSubdomain string) (*CustomDomainInfo, error) {
	if s.app.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	if s.app.authToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL("/api/custom-domains")
	reqBody, _ := json.Marshal(map[string]string{
		"domain":           domain,
		"target_subdomain": targetSubdomain,
	})

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

	var result CustomDomainInfo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete removes a custom domain.
func (s *CustomDomainService) Delete(id int64) error {
	if s.app.client == nil {
		return fmt.Errorf("not connected")
	}
	if s.app.authToken == "" {
		return fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL(fmt.Sprintf("/api/custom-domains/%d", id))
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

// Verify triggers CNAME verification for a custom domain.
func (s *CustomDomainService) Verify(id int64) (*VerifyResult, error) {
	if s.app.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	if s.app.authToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := s.app.api.BuildURL(fmt.Sprintf("/api/custom-domains/%d/verify", id))
	body, statusCode, err := s.app.api.Post(url, nil)
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

	var result VerifyResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
