package gui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/client"
	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/keyring"
)

// ErrTOTPRequired is returned when TOTP code is required
var ErrTOTPRequired = errors.New("TOTP code required")

// AuthService handles authentication operations
type AuthService struct {
	app *App
	log zerolog.Logger
}

// NewAuthService creates a new auth service
func NewAuthService(app *App) *AuthService {
	return &AuthService{
		app: app,
		log: app.log.With().Str("service", "auth").Logger(),
	}
}

// AuthMethod represents the authentication method
type AuthMethod string

const (
	AuthMethodToken    AuthMethod = "token"
	AuthMethodPassword AuthMethod = "password"
)

// LoginRequest represents a login request from the frontend
type LoginRequest struct {
	Method        AuthMethod `json:"method"`
	ServerAddress string     `json:"server_address"`
	Token         string     `json:"token,omitempty"`
	RefreshToken  string     `json:"refresh_token,omitempty"`
	Phone         string     `json:"phone,omitempty"`
	Password      string     `json:"password,omitempty"`
	TOTPCode      string     `json:"totp_code,omitempty"`
	Remember      bool       `json:"remember"`
}

// LoginResponse represents the login result
type LoginResponse struct {
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	TOTPRequired bool   `json:"totp_required,omitempty"`
}

// Login authenticates the user and connects to the server
func (s *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
	s.log.Info().
		Str("method", string(req.Method)).
		Str("server", req.ServerAddress).
		Msg("Login attempt")

	var token string
	var refreshToken string

	if req.Method == AuthMethodToken {
		// Direct token authentication
		token = req.Token
		refreshToken = req.RefreshToken
	} else {
		// Password authentication - get JWT from server
		tokens, err := s.authenticateWithPassword(req.ServerAddress, req.Phone, req.Password, req.TOTPCode)
		if err != nil {
			if errors.Is(err, ErrTOTPRequired) {
				return &LoginResponse{
					Success:      false,
					Error:        "TOTP code required",
					ErrorCode:    "TOTP_REQUIRED",
					TOTPRequired: true,
				}, nil
			}
			return &LoginResponse{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
		token = tokens.AccessToken
		refreshToken = tokens.RefreshToken
	}

	// Create client config
	cfg := &config.ClientConfig{
		Server: config.ClientServerSettings{
			Address: req.ServerAddress,
			Token:   token,
		},
		Reconnect: config.ReconnectSettings{
			Enabled:  true,
			Interval: 5 * time.Second,
		},
	}

	// Save auth state
	s.app.serverAddress = req.ServerAddress
	s.app.authToken = token

	// Create and connect client
	s.app.client = client.New(cfg, s.log)
	s.app.subscribeToClientEvents()

	if err := s.app.client.Connect(); err != nil {
		return &LoginResponse{
			Success: false,
			Error:   fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}

	// Save credentials if remember is enabled
	if req.Remember {
		creds := keyring.Credentials{
			ServerAddress: req.ServerAddress,
			AuthMethod:    string(req.Method),
			Token:         token,
			RefreshToken:  refreshToken,
		}
		if req.Method == AuthMethodPassword {
			creds.Phone = req.Phone
		}
		if err := s.app.keyring.SaveCredentials(creds); err != nil {
			s.log.Error().Err(err).Msg("Failed to save credentials")
		}
	}

	// Pull data from server and apply to local storage
	go func() {
		if syncData, err := s.app.SyncService.Pull(); err == nil {
			s.app.SyncService.ApplyServerData(syncData)
			s.log.Info().Msg("Server data synced after login")
		} else {
			s.log.Debug().Err(err).Msg("Failed to sync data after login")
		}
	}()

	return &LoginResponse{
		Success:  true,
		ClientID: "", // Will be set after connect
	}, nil
}

// Logout disconnects and clears credentials
func (s *AuthService) Logout() error {
	s.log.Info().Msg("Logging out")

	if s.app.client != nil {
		s.app.client.Close()
		s.app.client = nil
	}

	// Clear saved credentials
	if err := s.app.keyring.Clear(); err != nil {
		s.log.Error().Err(err).Msg("Failed to clear credentials")
	}

	return nil
}

// CheckAuth checks if saved credentials exist and are valid
func (s *AuthService) CheckAuth() (*AuthStatus, error) {
	creds, err := s.app.keyring.LoadCredentials()
	if err != nil {
		return &AuthStatus{HasCredentials: false}, nil
	}

	if creds.Token == "" && creds.JWT == "" {
		return &AuthStatus{HasCredentials: false}, nil
	}

	return &AuthStatus{
		HasCredentials: true,
		ServerAddress:  creds.ServerAddress,
		AuthMethod:     creds.AuthMethod,
		Phone:          creds.Phone,
	}, nil
}

// AuthStatus represents the current auth status
type AuthStatus struct {
	HasCredentials bool   `json:"has_credentials"`
	ServerAddress  string `json:"server_address,omitempty"`
	AuthMethod     string `json:"auth_method,omitempty"`
	Phone          string `json:"phone,omitempty"`
}

// AutoLogin attempts to login with saved credentials
func (s *AuthService) AutoLogin() (*LoginResponse, error) {
	creds, err := s.app.keyring.LoadCredentials()
	if err != nil {
		return &LoginResponse{
			Success: false,
			Error:   "No saved credentials",
		}, nil
	}

	if creds.Token == "" {
		return &LoginResponse{
			Success: false,
			Error:   "No saved token",
		}, nil
	}

	// Try login with saved token
	resp, err := s.Login(LoginRequest{
		Method:        AuthMethodToken,
		ServerAddress: creds.ServerAddress,
		Token:         creds.Token,
		RefreshToken:  creds.RefreshToken,
		Phone:         creds.Phone,
		Remember:      true,
	})

	// If login failed and we have refresh token, try to refresh
	if (err != nil || !resp.Success) && creds.RefreshToken != "" {
		s.log.Info().Msg("Token may be expired, attempting refresh")

		tokens, refreshErr := s.refreshAccessToken(creds.ServerAddress, creds.RefreshToken)
		if refreshErr != nil {
			s.log.Warn().Err(refreshErr).Msg("Token refresh failed")
			// Return original error
			if err != nil {
				return nil, err
			}
			return resp, nil
		}

		s.log.Info().Msg("Token refreshed successfully")

		// Retry with new token
		return s.Login(LoginRequest{
			Method:        AuthMethodToken,
			ServerAddress: creds.ServerAddress,
			Token:         tokens.AccessToken,
			RefreshToken:  tokens.RefreshToken,
			Phone:         creds.Phone,
			Remember:      true,
		})
	}

	return resp, err
}

// IsConnected returns true if the client is connected
func (s *AuthService) IsConnected() bool {
	return s.app.client != nil
}

// GetServerAddress returns the current server address
func (s *AuthService) GetServerAddress() string {
	if s.app.client == nil {
		creds, _ := s.app.keyring.LoadCredentials()
		if creds != nil {
			return creds.ServerAddress
		}
		return ""
	}
	return ""
}

// authTokens holds access and refresh tokens
type authTokens struct {
	AccessToken  string
	RefreshToken string
}

// authenticateWithPassword authenticates with phone/password and returns tokens
func (s *AuthService) authenticateWithPassword(serverAddr, phone, password, totpCode string) (*authTokens, error) {
	apiURL := s.buildAPIURL(serverAddr, "/api/auth/login")

	// Create request body
	reqBody := map[string]string{
		"phone":    phone,
		"password": password,
	}
	if totpCode != "" {
		reqBody["totp_code"] = totpCode
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Make HTTP request
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Post(apiURL, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		json.Unmarshal(body, &errResp)
		if errResp.Code == "TOTP_REQUIRED" {
			return nil, ErrTOTPRequired
		}
		if errResp.Error != "" {
			return nil, fmt.Errorf(errResp.Error)
		}
		return nil, fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var loginResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &authTokens{
		AccessToken:  loginResp.AccessToken,
		RefreshToken: loginResp.RefreshToken,
	}, nil
}

// refreshAccessToken uses refresh token to get a new access token
func (s *AuthService) refreshAccessToken(serverAddr, refreshToken string) (*authTokens, error) {
	apiURL := s.buildAPIURL(serverAddr, "/api/auth/refresh")

	reqBody := map[string]string{
		"refresh_token": refreshToken,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Post(apiURL, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed with status %d", resp.StatusCode)
	}

	var refreshResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &refreshResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &authTokens{
		AccessToken:  refreshResp.AccessToken,
		RefreshToken: refreshResp.RefreshToken,
	}, nil
}

// buildAPIURL constructs API URL from server address
func (s *AuthService) buildAPIURL(serverAddr, path string) string {
	host := serverAddr
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return fmt.Sprintf("https://%s%s", host, path)
}
