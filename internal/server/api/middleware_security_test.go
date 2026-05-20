package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mephistofox/fxtunnel/internal/server/auth"
)

// captureHandler stores r.RemoteAddr and the OriginalRemoteAddrKey value
// from context so tests can assert what middleware produced.
type captureHandler struct {
	gotRemoteAddr string
	gotOriginal   string
}

func (c *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.gotRemoteAddr = r.RemoteAddr
	if v, ok := r.Context().Value(auth.OriginalRemoteAddrKey).(string); ok {
		c.gotOriginal = v
	}
	w.WriteHeader(http.StatusOK)
}

func runMiddleware(t *testing.T, trusted []string, remoteAddr string, headers map[string]string) *captureHandler {
	t.Helper()
	h := &captureHandler{}
	mw := trustedRealIPMiddleware(trusted)(h)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = remoteAddr
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	mw.ServeHTTP(httptest.NewRecorder(), req)
	return h
}

func TestTrustedRealIP_TrustedSourceWithXRealIP(t *testing.T) {
	h := runMiddleware(t, []string{"127.0.0.1"}, "127.0.0.1:54321", map[string]string{
		"X-Real-IP": "5.6.7.8",
	})
	if h.gotRemoteAddr != "5.6.7.8" {
		t.Errorf("expected RemoteAddr rewritten to 5.6.7.8, got %q", h.gotRemoteAddr)
	}
	if h.gotOriginal != "127.0.0.1:54321" {
		t.Errorf("expected OriginalRemoteAddrKey = %q, got %q", "127.0.0.1:54321", h.gotOriginal)
	}
}

func TestTrustedRealIP_TrustedSourceWithXForwardedFor(t *testing.T) {
	// First entry in XFF chain is the original client; the rest is the proxy chain.
	h := runMiddleware(t, []string{"127.0.0.1"}, "127.0.0.1:54321", map[string]string{
		"X-Forwarded-For": "1.1.1.1, 2.2.2.2, 3.3.3.3",
	})
	if h.gotRemoteAddr != "1.1.1.1" {
		t.Errorf("expected first XFF entry, got %q", h.gotRemoteAddr)
	}
}

func TestTrustedRealIP_TrustedSourceXRealIPPreferredOverXFF(t *testing.T) {
	h := runMiddleware(t, []string{"127.0.0.1"}, "127.0.0.1:54321", map[string]string{
		"X-Real-IP":       "5.6.7.8",
		"X-Forwarded-For": "1.1.1.1, 2.2.2.2",
	})
	if h.gotRemoteAddr != "5.6.7.8" {
		t.Errorf("X-Real-IP should win over X-Forwarded-For; got %q", h.gotRemoteAddr)
	}
}

func TestTrustedRealIP_UntrustedSourceIgnoresHeaders(t *testing.T) {
	// Attacker connects directly to the API from 9.9.9.9 with a spoofed
	// X-Forwarded-For pretending to be YooKassa. Headers must be ignored.
	h := runMiddleware(t, []string{"127.0.0.1"}, "9.9.9.9:1234", map[string]string{
		"X-Real-IP":       "185.71.76.1",
		"X-Forwarded-For": "185.71.76.1",
	})
	if h.gotRemoteAddr != "9.9.9.9:1234" {
		t.Errorf("expected RemoteAddr untouched, got %q", h.gotRemoteAddr)
	}
	if h.gotOriginal != "9.9.9.9:1234" {
		t.Errorf("expected OriginalRemoteAddrKey = TCP source, got %q", h.gotOriginal)
	}
}

func TestTrustedRealIP_NoHeadersNoRewrite(t *testing.T) {
	h := runMiddleware(t, []string{"127.0.0.1"}, "127.0.0.1:54321", nil)
	if h.gotRemoteAddr != "127.0.0.1:54321" {
		t.Errorf("expected RemoteAddr untouched without headers, got %q", h.gotRemoteAddr)
	}
}

func TestTrustedRealIP_IPv6LoopbackTrusted(t *testing.T) {
	// Both ::1 and [::1] must be recognised as trusted loopback.
	h := runMiddleware(t, []string{"::1"}, "[::1]:54321", map[string]string{
		"X-Real-IP": "5.6.7.8",
	})
	if h.gotRemoteAddr != "5.6.7.8" {
		t.Errorf("expected IPv6 loopback to be trusted, got RemoteAddr=%q", h.gotRemoteAddr)
	}
}

func TestTrustedRealIP_EmptyTrustListIgnoresAllHeaders(t *testing.T) {
	// An operator with auth.trusted_proxies = [] is opting out of header
	// rewriting entirely — even loopback shouldn't promote headers.
	h := runMiddleware(t, []string{}, "127.0.0.1:54321", map[string]string{
		"X-Real-IP": "5.6.7.8",
	})
	if h.gotRemoteAddr != "127.0.0.1:54321" {
		t.Errorf("empty trust list must not rewrite even from loopback, got %q", h.gotRemoteAddr)
	}
}

func TestTrustedRealIP_TrueClientIPHeader(t *testing.T) {
	// Some Cloudflare-style setups use True-Client-IP instead of X-Real-IP.
	h := runMiddleware(t, []string{"127.0.0.1"}, "127.0.0.1:54321", map[string]string{
		"True-Client-IP": "203.0.113.10",
	})
	if h.gotRemoteAddr != "203.0.113.10" {
		t.Errorf("True-Client-IP not honoured: %q", h.gotRemoteAddr)
	}
}
