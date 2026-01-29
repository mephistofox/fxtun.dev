package api

import (
	"net/http"
	"strconv"
	"time"

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

		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		httpRequestDuration.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(wrapped.status),
		).Observe(duration)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
