package gui

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
)

// AccountInfo represents the user's account and plan information.
type AccountInfo struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	AvatarURL   string `json:"avatar_url"`

	// Plan info
	PlanName         string `json:"plan_name"`
	PlanSlug         string `json:"plan_slug"`
	MaxTunnels       int    `json:"max_tunnels"`
	MaxDomains       int    `json:"max_domains"`
	MaxCustomDomains int    `json:"max_custom_domains"`
	MaxTokens        int    `json:"max_tokens"`
	InspectorEnabled bool   `json:"inspector_enabled"`

	// Usage
	TunnelCount int `json:"tunnel_count"`
	DomainCount int `json:"domain_count"`
	TokenCount  int `json:"token_count"`
}

// AccountService handles account information operations.
type AccountService struct {
	app *App
	log zerolog.Logger
}

// NewAccountService creates a new account service.
func NewAccountService(app *App) *AccountService {
	return &AccountService{
		app: app,
		log: app.log.With().Str("component", "account-service").Logger(),
	}
}

// GetAccountInfo fetches the user's profile from the server.
func (s *AccountService) GetAccountInfo() (*AccountInfo, error) {
	url := s.app.api.BuildURL("/api/profile")
	body, status, err := s.app.api.Get(url)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("server returned status %d", status)
	}

	var resp struct {
		User *struct {
			DisplayName string `json:"display_name"`
			Email       string `json:"email"`
			Phone       string `json:"phone"`
			AvatarURL   string `json:"avatar_url"`
		} `json:"user"`
		Plan *struct {
			Name             string `json:"name"`
			Slug             string `json:"slug"`
			MaxTunnels       int    `json:"max_tunnels"`
			MaxDomains       int    `json:"max_domains"`
			MaxCustomDomains int    `json:"max_custom_domains"`
			MaxTokens        int    `json:"max_tokens"`
			InspectorEnabled bool   `json:"inspector_enabled"`
		} `json:"plan"`
		ReservedDomains []json.RawMessage `json:"reserved_domains"`
		TokenCount      int               `json:"token_count"`
		TunnelCount     int               `json:"tunnel_count"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	info := &AccountInfo{
		TokenCount:  resp.TokenCount,
		TunnelCount: resp.TunnelCount,
		DomainCount: len(resp.ReservedDomains),
	}

	if resp.User != nil {
		info.DisplayName = resp.User.DisplayName
		info.Email = resp.User.Email
		info.Phone = resp.User.Phone
		info.AvatarURL = resp.User.AvatarURL
	}

	if resp.Plan != nil {
		info.PlanName = resp.Plan.Name
		info.PlanSlug = resp.Plan.Slug
		info.MaxTunnels = resp.Plan.MaxTunnels
		info.MaxDomains = resp.Plan.MaxDomains
		info.MaxCustomDomains = resp.Plan.MaxCustomDomains
		info.MaxTokens = resp.Plan.MaxTokens
		info.InspectorEnabled = resp.Plan.InspectorEnabled
	}

	return info, nil
}

// GetUpgradeURL returns the URL for the upgrade/checkout page.
func (s *AccountService) GetUpgradeURL() string {
	return s.app.api.BuildURL("/checkout")
}

// GetManageURL returns the URL for the profile/manage page.
func (s *AccountService) GetManageURL() string {
	return s.app.api.BuildURL("/profile")
}
