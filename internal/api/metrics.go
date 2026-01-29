package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	activeTunnels = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fxtunnel_active_tunnels",
		Help: "Number of currently active tunnels",
	}, []string{"type"})

	activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fxtunnel_active_connections",
		Help: "Number of currently active client connections",
	})

	connectionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fxtunnel_connections_total",
		Help: "Total number of tunnel connections",
	}, []string{"type"})

	authAttemptsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fxtunnel_auth_attempts_total",
		Help: "Total authentication attempts",
	}, []string{"result"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fxtunnel_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})
)

func metricsHandler() http.Handler {
	return promhttp.Handler()
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		pattern := r.URL.Path
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			if p := rctx.RoutePattern(); p != "" {
				pattern = p
			}
		}
		httpRequestDuration.WithLabelValues(
			r.Method,
			pattern,
			strconv.Itoa(wrapped.Status()),
		).Observe(duration)
	})
}
