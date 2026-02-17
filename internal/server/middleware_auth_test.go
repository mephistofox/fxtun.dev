package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// hashCredentials is a test helper that generates a bcrypt hash from "user:password".
func hashCredentials(t *testing.T, username, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(username+":"+password), bcrypt.DefaultCost)
	require.NoError(t, err)
	return string(hash)
}

func TestCheckBasicAuth_NoAuthRequired(t *testing.T) {
	tunnel := &Tunnel{BasicAuthHash: ""}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	ok := checkBasicAuth(w, req, tunnel)

	assert.True(t, ok)
	assert.Equal(t, http.StatusOK, w.Code) // no response written
	assert.Empty(t, w.Header().Get("WWW-Authenticate"))
}

func TestCheckBasicAuth_ValidCredentials(t *testing.T) {
	hash := hashCredentials(t, "admin", "secretpass")
	tunnel := &Tunnel{BasicAuthHash: hash}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "secretpass")
	w := httptest.NewRecorder()

	ok := checkBasicAuth(w, req, tunnel)

	assert.True(t, ok)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("WWW-Authenticate"))
}

func TestCheckBasicAuth_InvalidCredentials(t *testing.T) {
	hash := hashCredentials(t, "admin", "secretpass")
	tunnel := &Tunnel{BasicAuthHash: hash}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "wrongpassword")
	w := httptest.NewRecorder()

	ok := checkBasicAuth(w, req, tunnel)

	assert.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, `Basic realm="fxTunnel"`, w.Header().Get("WWW-Authenticate"))
}

func TestCheckBasicAuth_MissingCredentials(t *testing.T) {
	hash := hashCredentials(t, "admin", "secretpass")
	tunnel := &Tunnel{BasicAuthHash: hash}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No auth header set
	w := httptest.NewRecorder()

	ok := checkBasicAuth(w, req, tunnel)

	assert.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, `Basic realm="fxTunnel"`, w.Header().Get("WWW-Authenticate"))
}

func TestCheckBasicAuth_WrongUsername(t *testing.T) {
	hash := hashCredentials(t, "admin", "secretpass")
	tunnel := &Tunnel{BasicAuthHash: hash}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("wronguser", "secretpass")
	w := httptest.NewRecorder()

	ok := checkBasicAuth(w, req, tunnel)

	assert.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, `Basic realm="fxTunnel"`, w.Header().Get("WWW-Authenticate"))
}
