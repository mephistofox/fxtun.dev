package gui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

	if req.Method == AuthMethodToken {
		// Direct token authentication
		token = req.Token
	} else {
		// Password authentication - get JWT from server
		jwt, err := s.authenticateWithPassword(req.ServerAddress, req.Phone, req.Password, req.TOTPCode)
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
		token = jwt
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
		}
		if req.Method == AuthMethodPassword {
			creds.Phone = req.Phone
		}
		if err := s.app.keyring.SaveCredentials(creds); err != nil {
			s.log.Error().Err(err).Msg("Failed to save credentials")
		}
	}

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

	return s.Login(LoginRequest{
		Method:        AuthMethod(creds.AuthMethod),
		ServerAddress: creds.ServerAddress,
		Token:         creds.Token,
		Remember:      true,
	})
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

// authenticateWithPassword authenticates with phone/password and returns JWT
func (s *AuthService) authenticateWithPassword(serverAddr, phone, password, totpCode string) (string, error) {
	// Build API URL
	apiURL := fmt.Sprintf("https://%s/api/auth/login", serverAddr)

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
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Make HTTP request
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Post(apiURL, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		json.Unmarshal(body, &errResp)
		if errResp.Code == "TOTP_REQUIRED" {
			return "", ErrTOTPRequired
		}
		if errResp.Error != "" {
			return "", fmt.Errorf(errResp.Error)
		}
		return "", fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var loginResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	// For now, use access token as the tunnel auth token
	// In production, you might need to exchange this for an API token
	return loginResp.AccessToken, nil
}
