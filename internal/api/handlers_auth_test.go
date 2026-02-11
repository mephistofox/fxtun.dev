package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mephistofox/fxtun.dev/internal/api/dto"
)

func TestHealth_Returns200(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := http.Get(env.Server.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body dto.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected status 'ok', got %q", body.Status)
	}
}

func postJSON(t *testing.T, url string, payload interface{}) *http.Response {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(data)) //nolint:gosec // test code
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func TestRegister_Success(t *testing.T) {
	env := setupTestEnv(t)

	resp := postJSON(t, env.Server.URL+"/api/auth/register", dto.RegisterRequest{
		Phone:       "+1234567890",
		Password:    "securepass123",
		DisplayName: "Test User",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var body dto.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if body.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if body.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if body.User == nil {
		t.Fatal("expected non-nil user")
	}
	if body.User.Phone != "+1234567890" {
		t.Fatalf("expected phone '+1234567890', got %q", body.User.Phone)
	}
}

func TestRegister_DuplicatePhone(t *testing.T) {
	env := setupTestEnv(t)

	resp := postJSON(t, env.Server.URL+"/api/auth/register", dto.RegisterRequest{
		Phone:    "+9999999999",
		Password: "securepass123",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first register expected 201, got %d", resp.StatusCode)
	}

	resp2 := postJSON(t, env.Server.URL+"/api/auth/register", dto.RegisterRequest{
		Phone:    "+9999999999",
		Password: "securepass123",
	})
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp2.StatusCode)
	}
}

func TestRegister_MissingFields(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name string
		req  dto.RegisterRequest
	}{
		{"empty phone", dto.RegisterRequest{Phone: "", Password: "securepass123"}},
		{"empty password", dto.RegisterRequest{Phone: "+1234567890", Password: ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := postJSON(t, env.Server.URL+"/api/auth/register", tc.req)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Register first
	resp := postJSON(t, env.Server.URL+"/api/auth/register", dto.RegisterRequest{
		Phone:    "+1112223333",
		Password: "securepass123",
	})
	resp.Body.Close()

	// Login
	resp2 := postJSON(t, env.Server.URL+"/api/auth/login", dto.LoginRequest{
		Phone:    "+1112223333",
		Password: "securepass123",
	})
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}

	var body dto.AuthResponse
	if err := json.NewDecoder(resp2.Body).Decode(&body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if body.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if body.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	env := setupTestEnv(t)

	resp := postJSON(t, env.Server.URL+"/api/auth/register", dto.RegisterRequest{
		Phone:    "+5556667777",
		Password: "securepass123",
	})
	resp.Body.Close()

	resp2 := postJSON(t, env.Server.URL+"/api/auth/login", dto.LoginRequest{
		Phone:    "+5556667777",
		Password: "wrongpassword",
	})
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp2.StatusCode)
	}
}

func TestRefresh_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Register
	resp := postJSON(t, env.Server.URL+"/api/auth/register", dto.RegisterRequest{
		Phone:    "+8889990000",
		Password: "securepass123",
	})
	defer resp.Body.Close()

	var authResp dto.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Refresh
	resp2 := postJSON(t, env.Server.URL+"/api/auth/refresh", dto.RefreshRequest{
		RefreshToken: authResp.RefreshToken,
	})
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}

	var refreshResp dto.AuthResponse
	if err := json.NewDecoder(resp2.Body).Decode(&refreshResp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if refreshResp.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if refreshResp.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	env := setupTestEnv(t)

	resp := postJSON(t, env.Server.URL+"/api/auth/refresh", dto.RefreshRequest{
		RefreshToken: "invalid-refresh-token",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
