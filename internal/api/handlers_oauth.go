package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/config"
)

const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token" //nolint:gosec // not a credential, this is GitHub's OAuth endpoint URL
	githubUserURL      = "https://api.github.com/user"
)

type githubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// handleGitHubAuth initiates the GitHub OAuth flow.
func (s *Server) handleGitHubAuth(w http.ResponseWriter, r *http.Request) {
	creds := s.cfg.OAuth.GitHub.GetCredentials(r.Host)
	if creds == nil {
		s.respondError(w, http.StatusNotImplemented, "GitHub OAuth is not configured for this domain")
		return
	}
	clientID := creds.ClientID

	state := "login"
	if r.URL.Query().Get("link") == "true" {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			s.respondError(w, http.StatusUnauthorized, "authorization token required for account linking")
			return
		}
		state = "link:" + token
	} else if desktopRedirect := r.URL.Query().Get("redirect_uri"); desktopRedirect != "" {
		if isLocalhostURI(desktopRedirect) {
			state = "login:" + desktopRedirect
		}
	}

	redirectURI := s.buildRedirectURI(r)

	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", "read:user,user:email")
	params.Set("state", state)

	http.Redirect(w, r, githubAuthorizeURL+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

// handleGitHubCallback handles the GitHub OAuth callback.
func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		s.redirectWithError(w, r, "missing authorization code")
		return
	}

	// Get credentials for this domain
	creds := s.cfg.OAuth.GitHub.GetCredentials(r.Host)
	if creds == nil {
		s.redirectWithError(w, r, "GitHub OAuth is not configured for this domain")
		return
	}

	// Exchange code for access token
	ghToken, err := s.exchangeGitHubCode(code, s.buildRedirectURI(r), creds)
	if err != nil {
		s.log.Error().Err(err).Msg("GitHub code exchange failed")
		s.redirectWithError(w, r, "failed to exchange authorization code")
		return
	}

	// Get GitHub user info
	ghUser, err := s.getGitHubUser(ghToken)
	if err != nil {
		s.log.Error().Err(err).Msg("GitHub user info request failed")
		s.redirectWithError(w, r, "failed to get GitHub user info")
		return
	}

	// Account linking flow
	if strings.HasPrefix(state, "link:") {
		jwtToken := strings.TrimPrefix(state, "link:")
		claims, err := s.authService.ValidateAccessToken(jwtToken)
		if err != nil {
			s.redirectWithError(w, r, "invalid or expired token")
			return
		}

		// Check if another user already has this GitHub ID — if so, merge
		existingUser, mergeErr := s.db.Users.GetByGitHubID(ghUser.ID)
		if mergeErr == nil && existingUser.ID != claims.UserID {
			if err := s.db.Users.MergeUsers(claims.UserID, existingUser.ID); err != nil {
				s.log.Error().Err(err).Int64("primary", claims.UserID).Int64("secondary", existingUser.ID).Msg("GitHub account merge failed")
				s.redirectWithError(w, r, "failed to merge accounts")
				return
			}
		} else if err := s.authService.LinkGitHub(claims.UserID, ghUser.ID, ghUser.Email, ghUser.AvatarURL); err != nil {
			s.log.Error().Err(err).Int64("user_id", claims.UserID).Msg("GitHub account linking failed")
			s.redirectWithError(w, r, "failed to link GitHub account")
			return
		}

		http.Redirect(w, r, "/profile?github_linked=true", http.StatusTemporaryRedirect)
		return
	}

	// Login / register flow
	displayName := ghUser.Name
	if displayName == "" {
		displayName = ghUser.Login
	}

	info := &auth.OAuthUserInfo{
		GitHubID:    ghUser.ID,
		Email:       ghUser.Email,
		DisplayName: displayName,
		AvatarURL:   ghUser.AvatarURL,
	}

	userAgent := r.UserAgent()
	ipAddress := r.RemoteAddr

	_, tokenPair, err := s.authService.RegisterOrLoginOAuth(info, userAgent, ipAddress)
	if err != nil {
		s.log.Error().Err(err).Msg("OAuth register/login failed")
		s.redirectWithError(w, r, "authentication failed")
		return
	}

	params := url.Values{}
	params.Set("access_token", tokenPair.AccessToken)
	params.Set("refresh_token", tokenPair.RefreshToken)
	params.Set("expires_in", fmt.Sprintf("%d", tokenPair.ExpiresIn))

	redirectTarget := "/auth/callback"
	if desktopRedirect, ok := parseDesktopRedirectFromState(state); ok {
		redirectTarget = desktopRedirect
	}

	http.Redirect(w, r, redirectTarget+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

// exchangeGitHubCode exchanges an authorization code for an access token.
func (s *Server) exchangeGitHubCode(code, redirectURI string, creds *config.GitHubDomainCredentials) (string, error) {
	data := url.Values{}
	data.Set("client_id", creds.ClientID)
	data.Set("client_secret", creds.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", githubTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var tokenResp githubTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	return tokenResp.AccessToken, nil
}

// getGitHubUser fetches the authenticated user's info from GitHub.
func (s *Server) getGitHubUser(accessToken string) (*githubUser, error) {
	req, err := http.NewRequest("GET", githubUserURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var user githubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &user, nil
}

const (
	googleAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL     = "https://oauth2.googleapis.com/token"        //nolint:gosec // not a credential, this is Google's OAuth endpoint URL
	googleUserInfoURL  = "https://www.googleapis.com/oauth2/v2/userinfo"
)

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type googleUser struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// handleGoogleAuth initiates the Google OAuth flow.
func (s *Server) handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	clientID := s.cfg.OAuth.Google.ClientID
	if clientID == "" {
		s.respondError(w, http.StatusNotImplemented, "Google OAuth is not configured")
		return
	}

	state := "login"
	if r.URL.Query().Get("link") == "true" {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			s.respondError(w, http.StatusUnauthorized, "authorization token required for account linking")
			return
		}
		state = "link:" + token
	} else if desktopRedirect := r.URL.Query().Get("redirect_uri"); desktopRedirect != "" {
		if isLocalhostURI(desktopRedirect) {
			state = "login:" + desktopRedirect
		}
	}

	redirectURI := s.buildGoogleRedirectURI(r)

	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)

	http.Redirect(w, r, googleAuthorizeURL+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

// handleGoogleCallback handles the Google OAuth callback.
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		s.redirectWithError(w, r, "missing authorization code")
		return
	}

	// Exchange code for access token
	gToken, err := s.exchangeGoogleCode(code, s.buildGoogleRedirectURI(r))
	if err != nil {
		s.log.Error().Err(err).Msg("Google code exchange failed")
		s.redirectWithError(w, r, "failed to exchange authorization code")
		return
	}

	// Get Google user info
	gUser, err := s.getGoogleUser(gToken)
	if err != nil {
		s.log.Error().Err(err).Msg("Google user info request failed")
		s.redirectWithError(w, r, "failed to get Google user info")
		return
	}

	// Account linking flow
	if strings.HasPrefix(state, "link:") {
		jwtToken := strings.TrimPrefix(state, "link:")
		claims, err := s.authService.ValidateAccessToken(jwtToken)
		if err != nil {
			s.redirectWithError(w, r, "invalid or expired token")
			return
		}

		// Check if another user already has this Google ID — if so, merge
		existingUser, mergeErr := s.db.Users.GetByGoogleID(gUser.ID)
		if mergeErr == nil && existingUser.ID != claims.UserID {
			if err := s.db.Users.MergeUsers(claims.UserID, existingUser.ID); err != nil {
				s.log.Error().Err(err).Int64("primary", claims.UserID).Int64("secondary", existingUser.ID).Msg("Google account merge failed")
				s.redirectWithError(w, r, "failed to merge accounts")
				return
			}
		} else if err := s.authService.LinkGoogle(claims.UserID, gUser.ID, gUser.Email, gUser.Picture); err != nil {
			s.log.Error().Err(err).Int64("user_id", claims.UserID).Msg("Google account linking failed")
			s.redirectWithError(w, r, "failed to link Google account")
			return
		}

		http.Redirect(w, r, "/profile?google_linked=true", http.StatusTemporaryRedirect)
		return
	}

	// Login / register flow
	info := &auth.GoogleOAuthUserInfo{
		GoogleID:    gUser.ID,
		Email:       gUser.Email,
		DisplayName: gUser.Name,
		AvatarURL:   gUser.Picture,
	}

	userAgent := r.UserAgent()
	ipAddress := r.RemoteAddr

	_, tokenPair, err := s.authService.RegisterOrLoginGoogleOAuth(info, userAgent, ipAddress)
	if err != nil {
		s.log.Error().Err(err).Msg("Google OAuth register/login failed")
		s.redirectWithError(w, r, "authentication failed")
		return
	}

	params := url.Values{}
	params.Set("access_token", tokenPair.AccessToken)
	params.Set("refresh_token", tokenPair.RefreshToken)
	params.Set("expires_in", fmt.Sprintf("%d", tokenPair.ExpiresIn))

	redirectTarget := "/auth/callback"
	if desktopRedirect, ok := parseDesktopRedirectFromState(state); ok {
		redirectTarget = desktopRedirect
	}

	http.Redirect(w, r, redirectTarget+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

// exchangeGoogleCode exchanges an authorization code for an access token.
func (s *Server) exchangeGoogleCode(code, redirectURI string) (string, error) {
	data := url.Values{}
	data.Set("client_id", s.cfg.OAuth.Google.ClientID)
	data.Set("client_secret", s.cfg.OAuth.Google.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequest("POST", googleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var tokenResp googleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	return tokenResp.AccessToken, nil
}

// getGoogleUser fetches the authenticated user's info from Google.
func (s *Server) getGoogleUser(accessToken string) (*googleUser, error) {
	req, err := http.NewRequest("GET", googleUserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var user googleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &user, nil
}

// buildGoogleRedirectURI constructs the Google OAuth callback URL from the incoming request.
func (s *Server) buildGoogleRedirectURI(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
			scheme = fwd
		} else {
			scheme = "http"
		}
	}
	return fmt.Sprintf("%s://%s/api/auth/google/callback", scheme, r.Host)
}

// buildRedirectURI constructs the OAuth callback URL from the incoming request.
func (s *Server) buildRedirectURI(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
			scheme = fwd
		} else {
			scheme = "http"
		}
	}
	return fmt.Sprintf("%s://%s/api/auth/github/callback", scheme, r.Host)
}

// redirectWithError redirects to the frontend auth callback with an error message.
func (s *Server) redirectWithError(w http.ResponseWriter, r *http.Request, message string) {
	params := url.Values{}
	params.Set("error", message)

	redirectTarget := "/auth/callback"
	if state := r.URL.Query().Get("state"); state != "" {
		if desktopRedirect, ok := parseDesktopRedirectFromState(state); ok {
			redirectTarget = desktopRedirect
		}
	}

	http.Redirect(w, r, redirectTarget+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

// isLocalhostURI checks if a URI starts with http://localhost: or http://127.0.0.1:
func isLocalhostURI(uri string) bool {
	return strings.HasPrefix(uri, "http://localhost:") || strings.HasPrefix(uri, "http://127.0.0.1:")
}

// parseDesktopRedirectFromState extracts desktop redirect URI from state like "login:http://localhost:12345/callback".
func parseDesktopRedirectFromState(state string) (string, bool) {
	if !strings.HasPrefix(state, "login:") {
		return "", false
	}
	redirect := strings.TrimPrefix(state, "login:")
	if redirect == "" || !isLocalhostURI(redirect) {
		return "", false
	}
	return redirect, true
}
