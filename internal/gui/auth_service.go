package gui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mephistofox/fxtun.dev/internal/client"
	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/keyring"
)

// ErrTOTPRequired is returned when TOTP code is required
var ErrTOTPRequired = errors.New("TOTP code required")

// AuthService handles authentication operations
type AuthService struct {
	app *App
	log zerolog.Logger

	// OAuth callback state
	oauthMu     sync.Mutex
	oauthCh     chan *authTokens
	oauthServer *http.Server
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
	AuthMethodOAuth    AuthMethod = "oauth"
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
	s.app.refreshToken = refreshToken

	// Create and connect client
	s.app.client = client.New(cfg, s.log)
	s.app.subscribeToClientEvents()

	// Set token refresher for automatic token renewal on reconnect
	if refreshToken != "" {
		s.app.client.SetTokenRefresher(s.createTokenRefresher(req.ServerAddress, req.Remember))
	}

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

	// Pull data from server and apply to local storage, then auto-connect bundles
	go func() {
		if syncData, err := s.app.SyncService.Pull(); err == nil {
			s.app.SyncService.ApplyServerData(syncData)
			s.log.Info().Msg("Server data synced after login")
		} else {
			s.log.Debug().Err(err).Msg("Failed to sync data after login")
		}

		// Auto-connect bundles marked for auto-start
		if tunnels, err := s.app.BundleService.ConnectAutoStart(); err == nil && len(tunnels) > 0 {
			s.log.Info().Int("count", len(tunnels)).Msg("Auto-connected bundles")
		}
	}()

	return &LoginResponse{
		Success:  true,
		ClientID: "", // Will be set after connect
	}, nil
}

// createTokenRefresher creates a callback function that refreshes the access token
func (s *AuthService) createTokenRefresher(serverAddr string, saveToKeyring bool) client.TokenRefresher {
	return func(_ string) (string, error) {
		s.log.Info().Msg("Token expired, attempting refresh...")

		// Use the stored refresh token
		refreshToken := s.app.refreshToken
		if refreshToken == "" {
			return "", fmt.Errorf("no refresh token available")
		}

		// Refresh the token
		tokens, err := s.refreshAccessToken(serverAddr, refreshToken)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to refresh token")
			return "", err
		}

		s.log.Info().Msg("Token refreshed successfully")

		// Update stored tokens
		s.app.authToken = tokens.AccessToken
		s.app.refreshToken = tokens.RefreshToken

		// Save to keyring if remember is enabled
		if saveToKeyring {
			creds, _ := s.app.keyring.LoadCredentials()
			if creds != nil {
				creds.Token = tokens.AccessToken
				creds.RefreshToken = tokens.RefreshToken
				if err := s.app.keyring.SaveCredentials(*creds); err != nil {
					s.log.Error().Err(err).Msg("Failed to update credentials in keyring")
				}
			}
		}

		return tokens.AccessToken, nil
	}
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
			return nil, fmt.Errorf("%s", errResp.Error)
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

// StartOAuthFlow opens the system browser for OAuth and starts a localhost callback server.
// Returns the provider URL that was opened.
func (s *AuthService) StartOAuthFlow(serverAddr, provider string) (string, error) {
	s.oauthMu.Lock()
	defer s.oauthMu.Unlock()

	// Clean up any previous OAuth server
	s.stopOAuthServerLocked()

	// Listen on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("listen: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	s.oauthCh = make(chan *authTokens, 1)
	ch := s.oauthCh

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.URL.Query().Get("access_token")
		refreshToken := r.URL.Query().Get("refresh_token")

		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html><html><body><h2>Ошибка авторизации</h2><p>%s</p><p>Вы можете закрыть это окно.</p></body></html>`, html.EscapeString(errMsg))
			select {
			case ch <- nil:
			default:
			}
			return
		}

		if accessToken == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<!DOCTYPE html><html><body><h2>Ошибка</h2><p>Токен не получен.</p></body></html>`)
			select {
			case ch <- nil:
			default:
			}
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><body><h2>Авторизация успешна!</h2><p>Вы можете закрыть это окно и вернуться в приложение.</p><script>window.close()</script></body></html>`)

		select {
		case ch <- &authTokens{AccessToken: accessToken, RefreshToken: refreshToken}:
		default:
		}
	})

	srv := &http.Server{Handler: mux}
	s.oauthServer = srv

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error().Err(err).Msg("OAuth callback server error")
		}
	}()

	// Build OAuth URL
	host := serverAddr
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	callbackURI := fmt.Sprintf("http://localhost:%d/callback", port)
	oauthURL := fmt.Sprintf("https://%s/api/auth/%s?redirect_uri=%s", host, provider, url.QueryEscape(callbackURI))

	s.log.Info().Str("provider", provider).Int("port", port).Msg("Starting OAuth flow")

	// Open in system browser
	wailsRuntime.BrowserOpenURL(s.app.ctx, oauthURL)

	return oauthURL, nil
}

// WaitOAuthCallback waits for the OAuth callback and returns a LoginResponse.
func (s *AuthService) WaitOAuthCallback(serverAddr string, remember bool) (*LoginResponse, error) {
	s.oauthMu.Lock()
	ch := s.oauthCh
	s.oauthMu.Unlock()

	if ch == nil {
		return &LoginResponse{Success: false, Error: "no OAuth flow in progress"}, nil
	}

	// Wait with timeout
	select {
	case tokens, ok := <-ch:
		s.stopOAuthServer()

		if !ok || tokens == nil {
			return &LoginResponse{Success: false, Error: "OAuth authentication cancelled"}, nil
		}

		// Bring the app window to the front
		s.bringWindowToFront()

		// Login with the received tokens
		return s.Login(LoginRequest{
			Method:        AuthMethodToken,
			ServerAddress: serverAddr,
			Token:         tokens.AccessToken,
			RefreshToken:  tokens.RefreshToken,
			Remember:      remember,
		})

	case <-time.After(5 * time.Minute):
		s.stopOAuthServer()
		return &LoginResponse{Success: false, Error: "OAuth flow timed out"}, nil
	}
}

// CancelOAuthFlow cancels any in-progress OAuth flow.
func (s *AuthService) CancelOAuthFlow() {
	s.stopOAuthServer()
}

func (s *AuthService) stopOAuthServer() {
	s.oauthMu.Lock()
	defer s.oauthMu.Unlock()
	s.stopOAuthServerLocked()
}

func (s *AuthService) stopOAuthServerLocked() {
	if s.oauthServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.oauthServer.Shutdown(ctx)
		s.oauthServer = nil
	}
	if s.oauthCh != nil {
		close(s.oauthCh)
		s.oauthCh = nil
	}
}

// bringWindowToFront activates the app window after OAuth callback.
func (s *AuthService) bringWindowToFront() {
	if s.app.ctx == nil {
		return
	}
	wailsRuntime.WindowUnminimise(s.app.ctx)
	wailsRuntime.WindowShow(s.app.ctx)
	wailsRuntime.WindowSetAlwaysOnTop(s.app.ctx, true)
	wailsRuntime.WindowSetAlwaysOnTop(s.app.ctx, false)
}

// buildAPIURL constructs API URL from server address
func (s *AuthService) buildAPIURL(serverAddr, path string) string {
	host := serverAddr
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return fmt.Sprintf("https://%s%s", host, path)
}
