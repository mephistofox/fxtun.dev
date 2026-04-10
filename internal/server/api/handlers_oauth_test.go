package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mephistofox/fxtunnel/internal/server/database"
)

func TestGitHubLinkCallback_ConflictWhenLinkedToAnotherUser(t *testing.T) {
	env := setupTestEnv(t)

	// Create two users
	user1 := env.createTestUser(t, "+1111111111", "password1", "User One")
	user2 := env.createTestUser(t, "+2222222222", "password2", "User Two")

	// Link GitHub ID 12345 to user2
	if err := env.AuthService.LinkGitHub(user2.User.ID, 12345, "user2@github.com", ""); err != nil {
		t.Fatalf("failed to link github to user2: %v", err)
	}

	// Now try to link the same GitHub ID to user1 — should get error redirect
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/auth/github/callback", nil)

	ghUser := &githubUser{
		ID:    12345,
		Login: "testuser",
		Email: "user2@github.com",
	}

	env.APIServer.handleGitHubLinkCallback(w, r, user1.User.ID, ghUser)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307 redirect, got %d", resp.StatusCode)
	}

	loc, err := resp.Location()
	if err != nil {
		t.Fatalf("expected Location header: %v", err)
	}

	errMsg := loc.Query().Get("error")
	if errMsg == "" {
		t.Fatal("expected error parameter in redirect URL")
	}
	if errMsg != "this GitHub account is already linked to another user" {
		t.Fatalf("unexpected error message: %q", errMsg)
	}
}

func TestGitHubLinkCallback_SuccessWhenLinkedToSameUser(t *testing.T) {
	env := setupTestEnv(t)

	user1 := env.createTestUser(t, "+1111111111", "password1", "User One")

	// Link GitHub ID 12345 to user1
	if err := env.AuthService.LinkGitHub(user1.User.ID, 12345, "user1@github.com", ""); err != nil {
		t.Fatalf("failed to link github to user1: %v", err)
	}

	// Try to link the same GitHub ID to the same user — should succeed
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/auth/github/callback", nil)

	ghUser := &githubUser{
		ID:    12345,
		Login: "testuser",
		Email: "user1@github.com",
	}

	env.APIServer.handleGitHubLinkCallback(w, r, user1.User.ID, ghUser)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307 redirect, got %d", resp.StatusCode)
	}

	loc, err := resp.Location()
	if err != nil {
		t.Fatalf("expected Location header: %v", err)
	}

	if loc.Query().Get("error") != "" {
		t.Fatalf("expected no error, got: %s", loc.Query().Get("error"))
	}
	if loc.Query().Get("github_linked") != "true" {
		t.Fatal("expected github_linked=true in redirect URL")
	}
}

func TestGitHubLinkCallback_SuccessNewLink(t *testing.T) {
	env := setupTestEnv(t)

	user1 := env.createTestUser(t, "+1111111111", "password1", "User One")

	// Link a brand new GitHub ID — should succeed
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/auth/github/callback", nil)

	ghUser := &githubUser{
		ID:    99999,
		Login: "newuser",
		Email: "new@github.com",
	}

	env.APIServer.handleGitHubLinkCallback(w, r, user1.User.ID, ghUser)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307 redirect, got %d", resp.StatusCode)
	}

	loc, err := resp.Location()
	if err != nil {
		t.Fatalf("expected Location header: %v", err)
	}

	if loc.Query().Get("error") != "" {
		t.Fatalf("expected no error, got: %s", loc.Query().Get("error"))
	}
	if loc.Query().Get("github_linked") != "true" {
		t.Fatal("expected github_linked=true in redirect URL")
	}

	// Verify the link was actually created in the database
	linkedUser, err := env.DB.Users.GetByGitHubID(99999)
	if err != nil {
		t.Fatalf("expected to find user by github ID: %v", err)
	}
	if linkedUser.ID != user1.User.ID {
		t.Fatalf("expected github ID linked to user %d, got %d", user1.User.ID, linkedUser.ID)
	}
}

func TestGoogleLinkCallback_ConflictWhenLinkedToAnotherUser(t *testing.T) {
	env := setupTestEnv(t)

	user1 := env.createTestUser(t, "+3333333333", "password1", "User One")
	user2 := env.createTestUser(t, "+4444444444", "password2", "User Two")

	// Link Google ID "google-123" to user2
	if err := env.AuthService.LinkGoogle(user2.User.ID, "google-123", "user2@google.com", ""); err != nil {
		t.Fatalf("failed to link google to user2: %v", err)
	}

	// Now try to link the same Google ID to user1 — should get error redirect
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/auth/google/callback", nil)

	gUser := &googleUser{
		ID:    "google-123",
		Email: "user2@google.com",
		Name:  "Test User",
	}

	env.APIServer.handleGoogleLinkCallback(w, r, user1.User.ID, gUser)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307 redirect, got %d", resp.StatusCode)
	}

	loc, err := resp.Location()
	if err != nil {
		t.Fatalf("expected Location header: %v", err)
	}

	errMsg := loc.Query().Get("error")
	if errMsg == "" {
		t.Fatal("expected error parameter in redirect URL")
	}
	if errMsg != "this Google account is already linked to another user" {
		t.Fatalf("unexpected error message: %q", errMsg)
	}
}

func TestGoogleLinkCallback_SuccessWhenLinkedToSameUser(t *testing.T) {
	env := setupTestEnv(t)

	user1 := env.createTestUser(t, "+3333333333", "password1", "User One")

	// Link Google ID "google-456" to user1
	if err := env.AuthService.LinkGoogle(user1.User.ID, "google-456", "user1@google.com", ""); err != nil {
		t.Fatalf("failed to link google to user1: %v", err)
	}

	// Try to link the same Google ID to the same user — should succeed
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/auth/google/callback", nil)

	gUser := &googleUser{
		ID:    "google-456",
		Email: "user1@google.com",
		Name:  "User One",
	}

	env.APIServer.handleGoogleLinkCallback(w, r, user1.User.ID, gUser)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307 redirect, got %d", resp.StatusCode)
	}

	loc, err := resp.Location()
	if err != nil {
		t.Fatalf("expected Location header: %v", err)
	}

	if loc.Query().Get("error") != "" {
		t.Fatalf("expected no error, got: %s", loc.Query().Get("error"))
	}
	if loc.Query().Get("google_linked") != "true" {
		t.Fatal("expected google_linked=true in redirect URL")
	}
}

func TestGoogleLinkCallback_SuccessNewLink(t *testing.T) {
	env := setupTestEnv(t)

	user1 := env.createTestUser(t, "+3333333333", "password1", "User One")

	// Link a brand new Google ID — should succeed
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/auth/google/callback", nil)

	gUser := &googleUser{
		ID:    "google-new-789",
		Email: "new@google.com",
		Name:  "New User",
	}

	env.APIServer.handleGoogleLinkCallback(w, r, user1.User.ID, gUser)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307 redirect, got %d", resp.StatusCode)
	}

	loc, err := resp.Location()
	if err != nil {
		t.Fatalf("expected Location header: %v", err)
	}

	if loc.Query().Get("error") != "" {
		t.Fatalf("expected no error, got: %s", loc.Query().Get("error"))
	}
	if loc.Query().Get("google_linked") != "true" {
		t.Fatal("expected google_linked=true in redirect URL")
	}

	// Verify the link was actually created in the database
	linkedUser, err := env.DB.Users.GetByGoogleID("google-new-789")
	if err != nil {
		t.Fatalf("expected to find user by google ID: %v", err)
	}
	if linkedUser.ID != user1.User.ID {
		t.Fatalf("expected google ID linked to user %d, got %d", user1.User.ID, linkedUser.ID)
	}
}

// TestGitHubLinkCallback_NoMergeOccurs verifies that the auto-merge vulnerability is fixed:
// when a GitHub ID is linked to another user, no data is transferred between accounts.
func TestGitHubLinkCallback_NoMergeOccurs(t *testing.T) {
	env := setupTestEnv(t)

	user1 := env.createTestUser(t, "+5555555555", "password1", "User One")
	user2 := env.createTestUser(t, "+6666666666", "password2", "User Two")

	// Create an API token for user2
	token := &database.APIToken{
		UserID:            user2.User.ID,
		TokenHash:         "test-hash-for-merge-check",
		Name:              "user2-token",
		AllowedSubdomains: []string{"*"},
		MaxTunnels:        5,
	}
	if err := env.DB.Tokens.Create(token); err != nil {
		t.Fatalf("failed to create token for user2: %v", err)
	}

	// Link GitHub ID to user2
	if err := env.AuthService.LinkGitHub(user2.User.ID, 77777, "user2@github.com", ""); err != nil {
		t.Fatalf("failed to link github to user2: %v", err)
	}

	// Try to link the same GitHub ID to user1 — should be rejected
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/auth/github/callback", nil)

	ghUser := &githubUser{ID: 77777, Login: "testuser", Email: "user2@github.com"}
	env.APIServer.handleGitHubLinkCallback(w, r, user1.User.ID, ghUser)

	// Verify user2's token was NOT transferred to user1
	user1Tokens, err := env.DB.Tokens.GetByUserID(user1.User.ID)
	if err != nil {
		t.Fatalf("failed to list user1 tokens: %v", err)
	}
	if len(user1Tokens) != 0 {
		t.Fatalf("expected user1 to have 0 tokens (no merge), got %d", len(user1Tokens))
	}

	// Verify user2's token is still with user2
	user2Tokens, err := env.DB.Tokens.GetByUserID(user2.User.ID)
	if err != nil {
		t.Fatalf("failed to list user2 tokens: %v", err)
	}
	if len(user2Tokens) != 1 {
		t.Fatalf("expected user2 to still have 1 token, got %d", len(user2Tokens))
	}
}

