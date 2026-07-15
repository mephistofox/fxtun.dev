package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/rs/zerolog"
)

// newPaymentTestServer creates a minimal API server for webhook IP-check tests.
// It does not require a database or auth service because the webhook route is
// public and IP validation happens before any DB access.
func newPaymentTestServer(t *testing.T) *Server {
	t.Helper()

	cfg := &config.ServerConfig{
		Domain: config.DomainSettings{
			Base: "test.localhost",
		},
		Web: config.WebSettings{
			Port: 8081,
			RateLimit: config.RateLimitConfig{
				Enabled: false,
			},
		},
		YooKassa: config.YooKassaSettings{
			Enabled:  true,
			TestMode: false,
		},
	}

	log := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	tp := newMockTunnelProvider()

	srv := New(cfg, nil, nil, tp, nil, nil, log)
	t.Cleanup(func() { close(srv.shutdownCh) })

	return srv
}

// TestYooKassaWebhook_SpoofedXForwardedFor verifies that an attacker cannot
// bypass the YooKassa IP allowlist by spoofing X-Forwarded-For.
// The original TCP remote address (1.2.3.4) must be checked, not the
// header-injected IP (185.71.76.1).
func TestYooKassaWebhook_SpoofedXForwardedFor(t *testing.T) {
	srv := newPaymentTestServer(t)

	body := `{"type":"notification","event":"payment.succeeded","object":{"id":"test","status":"succeeded"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/payments/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Attacker spoofs X-Forwarded-For with a YooKassa-allowed IP.
	req.Header.Set("X-Forwarded-For", "185.71.76.1")

	// The real TCP connection comes from an unauthorized IP.
	req.RemoteAddr = "1.2.3.4:12345"

	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for spoofed X-Forwarded-For, got %d; body: %s",
			rr.Code, rr.Body.String())
	}
}

// TestYooKassaWebhook_LegitimateIP verifies that a request from a real
// YooKassa IP (in the 185.71.76.0/27 range) is NOT rejected by the IP check.
// The request will pass the IP check and may fail later (e.g. parsing), but
// must NOT return 403.
func TestYooKassaWebhook_LegitimateIP(t *testing.T) {
	srv := newPaymentTestServer(t)

	body := `{"type":"notification","event":"payment.succeeded","object":{"id":"test","status":"succeeded"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/payments/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Legitimate YooKassa IP as the real TCP remote address.
	req.RemoteAddr = "185.71.76.1:54321"

	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	if rr.Code == http.StatusForbidden {
		t.Fatalf("expected request from legitimate YooKassa IP to pass IP check, got 403; body: %s",
			rr.Body.String())
	}
}

// TestYooKassaWebhook_DisabledReturns503 verifies that when YooKassa is
// disabled, the webhook returns 503 Service Unavailable.
func TestYooKassaWebhook_DisabledReturns503(t *testing.T) {
	srv := newPaymentTestServer(t)
	srv.cfg.YooKassa.Enabled = false

	req := httptest.NewRequest(http.MethodPost, "/api/payments/webhook", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "1.2.3.4:12345"

	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when YooKassa disabled, got %d", rr.Code)
	}
}
