package api

import (
	"context"
	"net"
	"net/http"
)

type contextKey string

const originalRemoteAddrKey contextKey = "originalRemoteAddr"

// saveOriginalIPMiddleware preserves the original TCP remote address
// before middleware.RealIP rewrites it from X-Forwarded-For.
// Use getOriginalRemoteAddr(r) to retrieve the unmodified IP (without port).
func saveOriginalIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), originalRemoteAddrKey, r.RemoteAddr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getOriginalRemoteAddr returns the TCP connection's remote IP (without port),
// unmodified by RealIP middleware. The port is stripped so rate limiters
// key on IP only — otherwise every new TCP connection from the same IP gets
// a fresh limiter bucket and rate limiting is effectively bypassed.
func getOriginalRemoteAddr(r *http.Request) string {
	raw, _ := r.Context().Value(originalRemoteAddrKey).(string)
	if raw == "" {
		raw = r.RemoteAddr
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		return host
	}
	return raw
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
