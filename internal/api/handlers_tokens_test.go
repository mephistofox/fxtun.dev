package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/stretchr/testify/require"
)

func TestCreateToken_Success(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+10000000001", "password123", "Token User")

	body := `{"name":"my-token","max_tunnels":5}`
	req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/tokens", strings.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var result dto.CreateTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if result.Info == nil {
		t.Fatal("expected non-nil info")
	}
	if result.Info.Name != "my-token" {
		t.Fatalf("expected name 'my-token', got %q", result.Info.Name)
	}
	if result.Info.MaxTunnels != 5 {
		t.Fatalf("expected max_tunnels 5, got %d", result.Info.MaxTunnels)
	}
}

func TestListTokens_Success(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+10000000002", "password123", "List User")

	// Create a token first
	body := `{"name":"list-test-token"}`
	createReq, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/tokens", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+user.AccessToken)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	createResp.Body.Close()

	// List tokens
	listReq, _ := http.NewRequest(http.MethodGet, env.Server.URL+"/api/tokens", nil)
	listReq.Header.Set("Authorization", "Bearer "+user.AccessToken)

	resp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.TokensListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Total < 1 {
		t.Fatalf("expected at least 1 token, got %d", result.Total)
	}

	found := false
	for _, tk := range result.Tokens {
		if tk.Name == "list-test-token" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("created token not found in list")
	}
}

func TestDeleteToken_Success(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+10000000003", "password123", "Delete User")

	// Create a token
	body := `{"name":"delete-test-token"}`
	createReq, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/tokens", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+user.AccessToken)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}

	var created dto.CreateTokenResponse
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	createResp.Body.Close()

	tokenID := created.Info.ID

	// Delete it
	delReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/tokens/%d", env.Server.URL, tokenID), nil)
	delReq.Header.Set("Authorization", "Bearer "+user.AccessToken)

	resp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success to be true")
	}
}

func TestTokens_Unauthorized(t *testing.T) {
	env := setupTestEnv(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/tokens"},
		{http.MethodPost, "/api/tokens"},
		{http.MethodDelete, "/api/tokens/1"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req, _ := http.NewRequest(ep.method, env.Server.URL+ep.path, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("expected status 401, got %d", resp.StatusCode)
			}
		})
	}
}
