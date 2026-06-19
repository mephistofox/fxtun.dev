package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mephistofox/fxtun.dev/internal/server/api/dto"
	"github.com/mephistofox/fxtun.dev/internal/server/auth"
	"github.com/mephistofox/fxtun.dev/internal/server/database"
)

func TestDeviceAuthorize_TokenLimitReached(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+10000000010", "password123", "Device User")

	// The free plan has max_tokens=1. Create one token manually to fill the limit.
	plainToken, err := auth.GenerateAPIToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	tokenHash := auth.HashToken(plainToken)
	if err := env.DB.Tokens.Create(&database.APIToken{
		UserID:            user.User.ID,
		TokenHash:         tokenHash,
		Name:              "existing-token",
		AllowedSubdomains: []string{"*"},
		MaxTunnels:        3,
	}); err != nil {
		t.Fatalf("failed to create existing token: %v", err)
	}

	// Create a device session
	codeReq, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/auth/device/code", nil)
	codeResp, err := http.DefaultClient.Do(codeReq)
	if err != nil {
		t.Fatalf("device code request failed: %v", err)
	}
	defer codeResp.Body.Close()

	if codeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for device code, got %d", codeResp.StatusCode)
	}

	var codeResult dto.DeviceCodeResponse
	if err := json.NewDecoder(codeResp.Body).Decode(&codeResult); err != nil {
		t.Fatalf("failed to decode device code response: %v", err)
	}

	// Try to authorize the device session — should fail with 403
	body := `{"session_id":"` + codeResult.SessionID + `"}`
	authReq, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/auth/device/authorize", strings.NewReader(body))
	authReq.Header.Set("Content-Type", "application/json")
	authReq.Header.Set("Authorization", "Bearer "+user.AccessToken)

	authResp, err := http.DefaultClient.Do(authReq)
	if err != nil {
		t.Fatalf("device authorize request failed: %v", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", authResp.StatusCode)
	}

	var errResult dto.ErrorResponse
	if err := json.NewDecoder(authResp.Body).Decode(&errResult); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResult.Code != "MAX_TOKENS" {
		t.Fatalf("expected error code 'MAX_TOKENS', got %q", errResult.Code)
	}

	if !strings.Contains(errResult.Error, "token limit reached") {
		t.Fatalf("expected error message to contain 'token limit reached', got %q", errResult.Error)
	}
}

func TestDeviceAuthorize_Success(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+10000000011", "password123", "Device User 2")

	// Free plan has max_tokens=1, user has 0 tokens — should succeed

	// Create a device session
	codeReq, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/auth/device/code", nil)
	codeResp, err := http.DefaultClient.Do(codeReq)
	if err != nil {
		t.Fatalf("device code request failed: %v", err)
	}
	defer codeResp.Body.Close()

	var codeResult dto.DeviceCodeResponse
	if err := json.NewDecoder(codeResp.Body).Decode(&codeResult); err != nil {
		t.Fatalf("failed to decode device code response: %v", err)
	}

	// Authorize the device session — should succeed
	body := `{"session_id":"` + codeResult.SessionID + `"}`
	authReq, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/auth/device/authorize", strings.NewReader(body))
	authReq.Header.Set("Content-Type", "application/json")
	authReq.Header.Set("Authorization", "Bearer "+user.AccessToken)

	authResp, err := http.DefaultClient.Do(authReq)
	if err != nil {
		t.Fatalf("device authorize request failed: %v", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", authResp.StatusCode)
	}

	// Verify the token was created with plan's MaxTunnelsPerToken (3 for free plan)
	tokens, err := env.DB.Tokens.GetByUserID(user.User.ID)
	if err != nil {
		t.Fatalf("failed to get tokens: %v", err)
	}

	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}

	if tokens[0].MaxTunnels != 3 {
		t.Fatalf("expected max_tunnels=3 (from free plan), got %d", tokens[0].MaxTunnels)
	}

	if tokens[0].Name != "CLI (device flow)" {
		t.Fatalf("expected token name 'CLI (device flow)', got %q", tokens[0].Name)
	}
}
