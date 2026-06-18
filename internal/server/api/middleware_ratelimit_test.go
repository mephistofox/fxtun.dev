package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	rl := newIPRateLimiter(10)
	handler := rateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should pass", i)
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := newIPRateLimiter(2)
	handler := rateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	rl := newIPRateLimiter(1)
	handler := rateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "1.1.1.1:1234"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "2.2.2.2:1234"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestRateLimiter_KeysOnRealClientBehindTrustedProxy(t *testing.T) {
	// Production wiring: trustedRealIPMiddleware runs first and rewrites
	// r.RemoteAddr to the real client IP (from X-Real-IP) when the TCP source
	// is a trusted proxy (nginx on loopback). The rate limiter must key on
	// that real client IP — NOT on the nginx upstream connection address.
	//
	// nginx reuses a small pool of keepalive upstream connections, so many
	// distinct real clients share the same TCP source (incl. ephemeral port).
	// If the limiter keys on the upstream connection, every user collapses
	// into one bucket and a single client can lock out everyone.
	rl := newIPRateLimiter(1)
	chain := trustedRealIPMiddleware([]string{"127.0.0.1"})(
		rateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	// Client A, through nginx upstream connection 127.0.0.1:5000.
	reqA := httptest.NewRequest("GET", "/", nil)
	reqA.RemoteAddr = "127.0.0.1:5000"
	reqA.Header.Set("X-Real-IP", "11.11.11.11")
	wA := httptest.NewRecorder()
	chain.ServeHTTP(wA, reqA)
	assert.Equal(t, http.StatusOK, wA.Code, "client A first request should pass")

	// Client B, through the SAME reused nginx upstream connection.
	reqB := httptest.NewRequest("GET", "/", nil)
	reqB.RemoteAddr = "127.0.0.1:5000"
	reqB.Header.Set("X-Real-IP", "22.22.22.22")
	wB := httptest.NewRecorder()
	chain.ServeHTTP(wB, reqB)
	assert.Equal(t, http.StatusOK, wB.Code,
		"a different real client must not be limited by another client's usage")
}

func TestRateLimiter_UsesRemoteAddr(t *testing.T) {
	// Rate limiter uses r.RemoteAddr which is set by trustedRealIPMiddleware upstream.
	// It should NOT read X-Real-IP or X-Forwarded-For headers directly.
	rl := newIPRateLimiter(1)
	handler := rateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request from 5.5.5.5 (simulating trustedRealIPMiddleware having set RemoteAddr)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "5.5.5.5"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request from same RemoteAddr should be rate limited
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "5.5.5.5"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)

	// X-Real-IP header should be ignored; a different RemoteAddr should not be limited
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "6.6.6.6"
	req3.Header.Set("X-Real-IP", "5.5.5.5")
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code, "should use RemoteAddr not X-Real-IP header")
}
