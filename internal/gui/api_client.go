package gui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// apiClient provides a centralized HTTP client with automatic token refresh.
type apiClient struct {
	app *App
	log zerolog.Logger
}

// BuildURL constructs a full API URL from a path.
func (c *apiClient) BuildURL(path string) string {
	host := c.app.serverAddress
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return fmt.Sprintf("https://%s%s", host, path)
}

// Get performs an authenticated GET request.
func (c *apiClient) Get(url string) ([]byte, int, error) {
	return c.doRequest("GET", url, nil)
}

// Post performs an authenticated POST request.
func (c *apiClient) Post(url string, body []byte) ([]byte, int, error) {
	return c.doRequest("POST", url, body)
}

// Put performs an authenticated PUT request.
func (c *apiClient) Put(url string, body []byte) ([]byte, int, error) {
	return c.doRequest("PUT", url, body)
}

// Delete performs an authenticated DELETE request.
func (c *apiClient) Delete(url string) ([]byte, int, error) {
	return c.doRequest("DELETE", url, nil)
}

func (c *apiClient) doRequest(method, url string, body []byte) ([]byte, int, error) {
	respBody, statusCode, err := c.executeRequest(method, url, body)
	if err != nil {
		return nil, 0, err
	}

	if statusCode == http.StatusUnauthorized {
		// Check if user is inactive
		var errResp struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		json.Unmarshal(respBody, &errResp)

		if strings.Contains(errResp.Error, "inactive") || errResp.Code == "USER_INACTIVE" {
			c.app.emitEvent("user_blocked", nil)
			return respBody, statusCode, fmt.Errorf("user account is inactive")
		}

		// Token expired — try refresh
		c.log.Info().Msg("Token expired, attempting refresh")
		tokens, refreshErr := c.app.AuthService.refreshAccessToken(c.app.serverAddress, c.app.refreshToken)
		if refreshErr != nil {
			c.log.Error().Err(refreshErr).Msg("Failed to refresh token")
			return respBody, statusCode, fmt.Errorf("token refresh failed: %w", refreshErr)
		}

		c.log.Info().Msg("Token refreshed successfully")

		// Update stored tokens
		c.app.authToken = tokens.AccessToken
		c.app.refreshToken = tokens.RefreshToken

		// Update keyring
		creds, _ := c.app.keyring.LoadCredentials()
		if creds != nil {
			creds.Token = tokens.AccessToken
			creds.RefreshToken = tokens.RefreshToken
			if err := c.app.keyring.SaveCredentials(*creds); err != nil {
				c.log.Error().Err(err).Msg("Failed to update credentials in keyring")
			}
		}

		// Retry with new token
		return c.executeRequest(method, url, body)
	}

	// Check for 403 user_inactive
	if statusCode == http.StatusForbidden {
		var errResp struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		json.Unmarshal(respBody, &errResp)
		if strings.Contains(errResp.Error, "inactive") || errResp.Code == "USER_INACTIVE" {
			c.app.emitEvent("user_blocked", nil)
			return respBody, statusCode, fmt.Errorf("user account is inactive")
		}
	}

	return respBody, statusCode, nil
}

func (c *apiClient) executeRequest(method, url string, body []byte) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.app.authToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
