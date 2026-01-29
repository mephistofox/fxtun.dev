package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mephistofox/fxtunnel/internal/api/dto"
)

func TestAdminStats_Success(t *testing.T) {
	env := setupTestEnv(t)
	admin := env.createTestAdmin(t, "+10000000001", "adminpass1", "Admin")

	env.TunnelProvider.stats = Stats{
		ActiveClients: 3,
		ActiveTunnels: 5,
		HTTPTunnels:   2,
		TCPTunnels:    2,
		UDPTunnels:    1,
	}

	req, _ := http.NewRequest("GET", env.Server.URL+"/api/admin/stats", nil)
	req.Header.Set("Authorization", "Bearer "+admin.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var stats dto.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if stats.ActiveClients != 3 {
		t.Errorf("expected ActiveClients=3, got %d", stats.ActiveClients)
	}
	if stats.ActiveTunnels != 5 {
		t.Errorf("expected ActiveTunnels=5, got %d", stats.ActiveTunnels)
	}
	if stats.HTTPTunnels != 2 {
		t.Errorf("expected HTTPTunnels=2, got %d", stats.HTTPTunnels)
	}
	if stats.TCPTunnels != 2 {
		t.Errorf("expected TCPTunnels=2, got %d", stats.TCPTunnels)
	}
	if stats.UDPTunnels != 1 {
		t.Errorf("expected UDPTunnels=1, got %d", stats.UDPTunnels)
	}
	if stats.TotalUsers < 1 {
		t.Errorf("expected TotalUsers >= 1, got %d", stats.TotalUsers)
	}
}

func TestAdminStats_NonAdmin(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+10000000002", "userpass12", "Regular")

	req, _ := http.NewRequest("GET", env.Server.URL+"/api/admin/stats", nil)
	req.Header.Set("Authorization", "Bearer "+user.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestAdminListUsers_Success(t *testing.T) {
	env := setupTestEnv(t)
	admin := env.createTestAdmin(t, "+10000000003", "adminpass1", "Admin")
	env.createTestUser(t, "+10000000004", "userpass12", "User2")

	req, _ := http.NewRequest("GET", env.Server.URL+"/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+admin.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Users []*dto.UserDTO `json:"users"`
		Total int            `json:"total"`
		Page  int            `json:"page"`
		Limit int            `json:"limit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Total < 2 {
		t.Errorf("expected at least 2 users, got %d", result.Total)
	}
	if len(result.Users) < 2 {
		t.Errorf("expected at least 2 users in list, got %d", len(result.Users))
	}
}

func TestAdminInviteCodes_Create(t *testing.T) {
	env := setupTestEnv(t)
	admin := env.createTestAdmin(t, "+10000000005", "adminpass1", "Admin")

	body, _ := json.Marshal(dto.CreateInviteCodeRequest{ExpiresInDays: 7})
	req, _ := http.NewRequest("POST", env.Server.URL+"/api/admin/invite-codes", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+admin.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var code dto.InviteCodeDTO
	if err := json.NewDecoder(resp.Body).Decode(&code); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if code.Code == "" {
		t.Error("expected non-empty invite code")
	}
	if code.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	} else if code.ExpiresAt.Before(time.Now()) {
		t.Error("expected ExpiresAt to be in the future")
	}
}

func TestAdminInviteCodes_List(t *testing.T) {
	env := setupTestEnv(t)
	admin := env.createTestAdmin(t, "+10000000006", "adminpass1", "Admin")

	// Create an invite code first
	body, _ := json.Marshal(dto.CreateInviteCodeRequest{})
	createReq, _ := http.NewRequest("POST", env.Server.URL+"/api/admin/invite-codes", bytes.NewReader(body))
	createReq.Header.Set("Authorization", "Bearer "+admin.AccessToken)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	createResp.Body.Close()

	// List invite codes
	req, _ := http.NewRequest("GET", env.Server.URL+"/api/admin/invite-codes", nil)
	req.Header.Set("Authorization", "Bearer "+admin.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result dto.InviteCodesListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Total < 1 {
		t.Errorf("expected at least 1 invite code, got %d", result.Total)
	}
	if len(result.Codes) < 1 {
		t.Errorf("expected at least 1 code in list, got %d", len(result.Codes))
	}
}

func TestAdminListTunnels(t *testing.T) {
	env := setupTestEnv(t)
	admin := env.createTestAdmin(t, "+10000000007", "adminpass1", "Admin")

	env.TunnelProvider.tunnels = []TunnelInfo{
		{
			ID:        "tun-1",
			Type:      "http",
			Name:      "web",
			Subdomain: "myapp",
			LocalPort: 3000,
			ClientID:  "client-1",
			UserID:    admin.User.ID,
			CreatedAt: time.Now(),
		},
	}

	req, _ := http.NewRequest("GET", env.Server.URL+"/api/admin/tunnels", nil)
	req.Header.Set("Authorization", "Bearer "+admin.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result dto.AdminTunnelsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("expected 1 tunnel, got %d", result.Total)
	}
	if len(result.Tunnels) != 1 {
		t.Fatalf("expected 1 tunnel in list, got %d", len(result.Tunnels))
	}
	if result.Tunnels[0].ID != "tun-1" {
		t.Errorf("expected tunnel ID 'tun-1', got '%s'", result.Tunnels[0].ID)
	}
	if result.Tunnels[0].Type != "http" {
		t.Errorf("expected tunnel type 'http', got '%s'", result.Tunnels[0].Type)
	}
	if result.Tunnels[0].Subdomain != "myapp" {
		t.Errorf("expected subdomain 'myapp', got '%s'", result.Tunnels[0].Subdomain)
	}
}
