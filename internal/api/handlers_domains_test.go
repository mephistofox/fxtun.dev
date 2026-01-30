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

func TestReserveDomain_Success(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+20000000001", "password123", "Domain User")

	body := `{"subdomain":"mytest"}`
	req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/domains", strings.NewReader(body))
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

	var result dto.DomainDTO
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Subdomain != "mytest" {
		t.Fatalf("expected subdomain 'mytest', got %q", result.Subdomain)
	}
	if result.URL != "https://mytest.test.localhost" {
		t.Fatalf("expected URL 'https://mytest.test.localhost', got %q", result.URL)
	}
}

func TestListDomains_Success(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+20000000002", "password123", "List Domain User")

	// Reserve a domain first
	body := `{"subdomain":"listdomain"}`
	createReq, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/domains", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+user.AccessToken)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	createResp.Body.Close()

	// List domains
	listReq, _ := http.NewRequest(http.MethodGet, env.Server.URL+"/api/domains", nil)
	listReq.Header.Set("Authorization", "Bearer "+user.AccessToken)

	resp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.DomainsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Total < 1 {
		t.Fatalf("expected at least 1 domain, got %d", result.Total)
	}

	found := false
	for _, d := range result.Domains {
		if d.Subdomain == "listdomain" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("reserved domain not found in list")
	}
}

func TestReleaseDomain_Success(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+20000000003", "password123", "Release User")

	// Reserve a domain
	body := `{"subdomain":"releaseme"}`
	createReq, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/domains", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+user.AccessToken)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}

	var created dto.DomainDTO
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	createResp.Body.Close()

	domainID := created.ID

	// Release it
	delReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/domains/%d", env.Server.URL, domainID), nil)
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

func TestCheckDomain_Available(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+20000000004", "password123", "Check User")

	req, _ := http.NewRequest(http.MethodGet, env.Server.URL+"/api/domains/check/availabletest", nil)
	req.Header.Set("Authorization", "Bearer "+user.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.DomainCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !result.Available {
		t.Fatal("expected domain to be available")
	}
	if result.Subdomain != "availabletest" {
		t.Fatalf("expected subdomain 'availabletest', got %q", result.Subdomain)
	}
}

func TestDomains_Unauthorized(t *testing.T) {
	env := setupTestEnv(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/domains"},
		{http.MethodPost, "/api/domains"},
		{http.MethodDelete, "/api/domains/1"},
		{http.MethodGet, "/api/domains/check/test"},
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
