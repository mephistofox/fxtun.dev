package auth

import (
	"context"
	"net/http"
	"testing"
)

func TestGetClientIP_IgnoresSpoofedHeaders(t *testing.T) {
	// A request with spoofed X-Forwarded-For and X-Real-IP headers
	// should return the actual RemoteAddr, not the spoofed values.
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.100:12345"
	r.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	r.Header.Set("X-Real-IP", "10.0.0.1")

	got := GetClientIP(r)
	want := "192.168.1.100"

	if got != want {
		t.Errorf("GetClientIP() = %q, want %q (should ignore spoofed headers)", got, want)
	}
}

func TestGetClientIP_UsesOriginalRemoteAddrFromContext(t *testing.T) {
	// When the original TCP address is stored in context (by saveOriginalIPMiddleware),
	// GetClientIP should return that address even if RemoteAddr was rewritten.
	r, _ := http.NewRequest("GET", "/", nil)
	// Simulate RealIP middleware having rewritten RemoteAddr
	r.RemoteAddr = "10.0.0.1:0"

	// Store the original TCP address in context
	ctx := context.WithValue(r.Context(), OriginalRemoteAddrKey, "203.0.113.50:54321")
	r = r.WithContext(ctx)

	got := GetClientIP(r)
	want := "203.0.113.50"

	if got != want {
		t.Errorf("GetClientIP() = %q, want %q (should use original addr from context)", got, want)
	}
}

func TestGetClientIP_FallsBackToRemoteAddr(t *testing.T) {
	// When there is no original address in context, fall back to RemoteAddr.
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.16.0.5:9999"

	got := GetClientIP(r)
	want := "172.16.0.5"

	if got != want {
		t.Errorf("GetClientIP() = %q, want %q", got, want)
	}
}

func TestGetClientIP_IPv6(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "[::1]:12345"

	got := GetClientIP(r)
	want := "::1"

	if got != want {
		t.Errorf("GetClientIP() = %q, want %q (IPv6)", got, want)
	}
}

func TestGetClientIP_IPv6FromContext(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:0"

	ctx := context.WithValue(r.Context(), OriginalRemoteAddrKey, "[2001:db8::1]:443")
	r = r.WithContext(ctx)

	got := GetClientIP(r)
	want := "2001:db8::1"

	if got != want {
		t.Errorf("GetClientIP() = %q, want %q (IPv6 from context)", got, want)
	}
}

func TestGetClientIP_AddrWithoutPort(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.5"

	got := GetClientIP(r)
	want := "10.0.0.5"

	if got != want {
		t.Errorf("GetClientIP() = %q, want %q (no port)", got, want)
	}
}
