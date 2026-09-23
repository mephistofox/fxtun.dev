package api

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/mephistofox/fxtun.dev/internal/server/auth"
)

// trustedRealIPMiddleware rewrites r.RemoteAddr from X-Real-IP /
// True-Client-IP / X-Forwarded-For headers, but ONLY when the immediate
// TCP source is in the trusted-proxies list. Headers from an untrusted
// direct connection are ignored, so an attacker who can reach the API
// directly cannot spoof a different IP through forwarded headers.
//
// The TCP source is always preserved in auth.OriginalRemoteAddrKey so
// handlers that need the proxy address itself (e.g. payment webhook
// IP-allowlist checks where the proxy is the trust boundary) can still
// read it via getOriginalRemoteAddr.
func trustedRealIPMiddleware(trustedProxies []string) func(http.Handler) http.Handler {
	trusted := make(map[string]struct{}, len(trustedProxies))
	for _, p := range trustedProxies {
		if ip := canonicalIP(p); ip != "" {
			trusted[ip] = struct{}{}
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), auth.OriginalRemoteAddrKey, r.RemoteAddr)
			r = r.WithContext(ctx)

			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			host = canonicalIP(host)

			if _, ok := trusted[host]; ok {
				if real := realIPFromHeaders(r); real != "" {
					r.RemoteAddr = real
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// realIPFromHeaders returns the first non-empty client IP candidate from
// the conventional reverse-proxy headers. For X-Forwarded-For, only the
// first entry (the original client) is returned; the rest is the proxy
// chain and not trustworthy.
func realIPFromHeaders(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Real-IP")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get("True-Client-IP")); v != "" {
		return v
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		if i := strings.IndexByte(v, ','); i >= 0 {
			return strings.TrimSpace(v[:i])
		}
		return strings.TrimSpace(v)
	}
	return ""
}

// canonicalIP normalises an IP literal: strips IPv6 brackets and returns
// the form net.ParseIP produces (so "[::1]" and "::1" compare equal).
// Returns the original string on parse failure so the caller can decide.
func canonicalIP(host string) string {
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return host
}

// getOriginalRemoteAddr returns the TCP connection's remote address,
// unmodified by trustedRealIPMiddleware. Falls back to r.RemoteAddr.
func getOriginalRemoteAddr(r *http.Request) string {
	if addr, ok := r.Context().Value(auth.OriginalRemoteAddrKey).(string); ok {
		return addr
	}
	return r.RemoteAddr
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; "+
			"script-src 'self' https://www.googletagmanager.com https://mc.yandex.ru https://mc.yandex.com https://yastatic.net; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data: https://mc.yandex.ru https://www.googletagmanager.com https://*.google-analytics.com; "+
			"connect-src 'self' https://www.google-analytics.com https://*.google-analytics.com https://mc.yandex.ru https://mc.yandex.com wss://mc.yandex.ru wss://mc.yandex.com; "+
			"font-src 'self'; "+
			"frame-src https://mc.yandex.ru https://mc.yandex.com https://mc.yandex.md; "+
			"frame-ancestors 'none'")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}
