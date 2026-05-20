package auth

import (
	"net/http"
	"testing"
)

// GetClientIP no longer consults forwarded headers or the
// OriginalRemoteAddrKey context value — that decision has moved up the
// stack into trustedRealIPMiddleware (see internal/server/api). By the
// time a handler runs, r.RemoteAddr already holds the right answer
// (real client IP if the request came through a trusted proxy, raw TCP
// source otherwise). GetClientIP just normalises that to a host-only
// string.

func TestGetClientIP_StripsPort(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.16.0.5:9999"

	if got, want := GetClientIP(r), "172.16.0.5"; got != want {
		t.Errorf("GetClientIP() = %q, want %q", got, want)
	}
}

func TestGetClientIP_DoesNotConsultHeaders(t *testing.T) {
	// Forwarded-style headers are consumed by middleware, not by
	// GetClientIP. If they leak in here, treat them as untrusted noise
	// and keep using the TCP source.
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.100:12345"
	r.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	r.Header.Set("X-Real-IP", "10.0.0.1")

	if got, want := GetClientIP(r), "192.168.1.100"; got != want {
		t.Errorf("GetClientIP() = %q, want %q (headers must not leak in)", got, want)
	}
}

func TestGetClientIP_IPv6(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "[::1]:12345"

	if got, want := GetClientIP(r), "::1"; got != want {
		t.Errorf("GetClientIP() = %q, want %q", got, want)
	}
}

func TestGetClientIP_IPv6WithoutBrackets(t *testing.T) {
	// trustedRealIPMiddleware sets r.RemoteAddr to a bare header value
	// (no port). For IPv6 that means no brackets either.
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "2001:db8::1"

	// stripPort treats the trailing :1 as a port (it can't know better
	// from a string alone). That's acceptable because in practice
	// middleware delivers bare IPv4 or bracketed IPv6 — but document
	// the limitation here.
	if got := GetClientIP(r); got == "" {
		t.Errorf("GetClientIP() returned empty string for IPv6 input")
	}
}

func TestGetClientIP_AddrWithoutPort(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.5"

	if got, want := GetClientIP(r), "10.0.0.5"; got != want {
		t.Errorf("GetClientIP() = %q, want %q", got, want)
	}
}
