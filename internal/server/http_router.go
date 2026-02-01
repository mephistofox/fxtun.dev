package server

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/inspect"
	"github.com/mephistofox/fxtunnel/internal/protocol"
)

//go:embed templates/*.html
var templateFS embed.FS

var (
	interstitialTmpl = template.Must(template.ParseFS(templateFS, "templates/interstitial.html"))
	errorTmpl        = template.Must(template.ParseFS(templateFS, "templates/error.html"))
)

// HTTPRouter routes HTTP requests to the appropriate tunnel.
// It implements http.Handler for use with net/http.Server.
type HTTPRouter struct {
	server  *Server
	log     zerolog.Logger
	tunnels map[string]*Tunnel // subdomain -> tunnel
	mu      sync.RWMutex
}

// NewHTTPRouter creates a new HTTP router
func NewHTTPRouter(server *Server, log zerolog.Logger) *HTTPRouter {
	return &HTTPRouter{
		server:  server,
		log:     log.With().Str("component", "http_router").Logger(),
		tunnels: make(map[string]*Tunnel),
	}
}

// RegisterTunnel registers a tunnel for a subdomain
func (r *HTTPRouter) RegisterTunnel(subdomain string, tunnel *Tunnel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	subdomain = strings.ToLower(subdomain)

	if _, exists := r.tunnels[subdomain]; exists {
		return fmt.Errorf("subdomain already in use: %s", subdomain)
	}

	r.tunnels[subdomain] = tunnel
	r.log.Debug().Str("subdomain", subdomain).Str("tunnel_id", tunnel.ID).Msg("Tunnel registered")
	return nil
}

// UnregisterTunnel removes a tunnel for a subdomain
func (r *HTTPRouter) UnregisterTunnel(subdomain string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	subdomain = strings.ToLower(subdomain)
	delete(r.tunnels, subdomain)
	r.log.Debug().Str("subdomain", subdomain).Msg("Tunnel unregistered")
}

// GetTunnel returns the tunnel for a subdomain
func (r *HTTPRouter) GetTunnel(subdomain string) *Tunnel {
	r.mu.RLock()
	defer r.mu.RUnlock()

	subdomain = strings.ToLower(subdomain)
	return r.tunnels[subdomain]
}

// ServeHTTP implements http.Handler. Go's net/http.Server handles HTTP/1.1
// keep-alive automatically when using this interface.
func (r *HTTPRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.server.activeConns.Add(1)
	defer r.server.activeConns.Done()

	// ACME challenge intercept
	if r.server.certManager != nil && strings.HasPrefix(req.URL.Path, "/.well-known/acme-challenge/") {
		r.server.certManager.HandleACMEChallenge(w, req)
		return
	}

	// Extract subdomain from Host header
	subdomain := r.extractSubdomain(req.Host)
	if subdomain == "" {
		// Try custom domain lookup
		cd := r.server.LookupCustomDomain(req.Host)
		if cd != nil && cd.Verified {
			subdomain = cd.TargetSubdomain
		}
	}
	if subdomain == "" {
		r.log.Debug().Str("host", req.Host).Msg("No subdomain or custom domain found")
		r.serveErrorPage(w, http.StatusNotFound, "Tunnel not found")
		return
	}

	// Find tunnel
	tunnel := r.GetTunnel(subdomain)
	if tunnel == nil {
		r.log.Debug().Str("subdomain", subdomain).Msg("Tunnel not found")
		r.serveErrorPage(w, http.StatusNotFound, "Tunnel not found")
		return
	}

	// Get client
	client := r.server.GetClient(tunnel.ClientID)
	if client == nil {
		r.log.Warn().Str("client_id", tunnel.ClientID).Msg("Client not found for tunnel")
		r.serveErrorPage(w, http.StatusBadGateway, "Tunnel unavailable")
		return
	}

	// Check for interstitial warning (skip for admin tunnels and custom domains)
	isCustomDomain := r.server.LookupCustomDomain(req.Host) != nil
	if !client.IsAdmin && !isCustomDomain && r.shouldShowInterstitial(req, subdomain) {
		r.serveInterstitialPage(w, req, subdomain)
		return
	}

	// Generate trace ID for this request
	traceID := generateShortID() + generateShortID() // 16 hex chars
	req.Header.Set("X-Trace-Id", traceID)

	// Open stream to client
	stream, err := client.OpenStream()
	if err != nil {
		r.log.Error().Err(err).Msg("Failed to open stream to client")
		r.serveErrorPage(w, http.StatusBadGateway, "Failed to connect to tunnel")
		return
	}
	defer stream.Close()

	// Send binary stream header
	remoteAddr := req.RemoteAddr
	if err := protocol.WriteStreamHeader(stream, tunnel.ID, remoteAddr); err != nil {
		r.log.Error().Err(err).Msg("Failed to send connection info")
		r.serveErrorPage(w, http.StatusBadGateway, "Failed to connect to tunnel")
		return
	}

	// Add forwarding headers
	clientIP := remoteAddr
	if host, _, err := net.SplitHostPort(clientIP); err == nil {
		clientIP = host
	}
	if prior := req.Header.Get("X-Forwarded-For"); prior != "" {
		clientIP = prior + ", " + clientIP
	}
	req.Header.Set("X-Forwarded-For", clientIP)
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-Host", req.Host)

	// WebSocket / HTTP Upgrade: hijack and do bidirectional proxy
	if isUpgradeRequest(req) {
		r.serveUpgrade(w, req, stream)
		return
	}

	// --- Inspection: capture request body ---
	inspectBuf := r.server.inspectMgr.Get(tunnel.ID)
	if inspectBuf == nil {
		r.log.Debug().Str("tunnel_id", tunnel.ID).Msg("Inspect buffer not found for tunnel")
	}
	startTime := time.Now()
	var capturedReqBody []byte

	if inspectBuf != nil && req.Body != nil {
		maxBody := r.server.inspectMgr.MaxBodySize()
		capturedReqBody, _ = io.ReadAll(io.LimitReader(req.Body, int64(maxBody)))
		req.Body = io.NopCloser(bytes.NewReader(capturedReqBody))
	}

	// Write the HTTP request to the stream
	if err := req.Write(stream); err != nil {
		r.log.Error().Err(err).Msg("Failed to write request to stream")
		r.serveErrorPage(w, http.StatusBadGateway, "Failed to proxy request")
		return
	}

	// Read response from stream
	streamReader := bufio.NewReader(stream)
	resp, err := http.ReadResponse(streamReader, req)
	if err != nil {
		r.log.Error().Err(err).Msg("Failed to read response from tunnel")
		r.serveErrorPage(w, http.StatusBadGateway, "Failed to read tunnel response")
		return
	}
	defer resp.Body.Close()

	// Copy response headers to ResponseWriter
	for key, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// --- Inspection: set up TeeReader to capture while streaming ---
	var capturedRespBuf bytes.Buffer
	bodyReader := io.Reader(resp.Body)
	if inspectBuf != nil {
		maxBody := r.server.inspectMgr.MaxBodySize()
		bodyReader = io.TeeReader(resp.Body, &limitedWriter{w: &capturedRespBuf, remaining: maxBody})
	}

	// Copy response body, using Flusher for streaming
	if flusher, ok := w.(http.Flusher); ok {
		buf := proxyBufPool.Get().(*[]byte)
		defer proxyBufPool.Put(buf)
		for {
			n, readErr := bodyReader.Read(*buf)
			if n > 0 {
				if _, writeErr := w.Write((*buf)[:n]); writeErr != nil {
					break
				}
				flusher.Flush()
			}
			if readErr != nil {
				break
			}
		}
	} else {
		bp := proxyBufPool.Get().(*[]byte)
		_, _ = io.CopyBuffer(w, bodyReader, *bp)
		proxyBufPool.Put(bp)
	}

	// --- Inspection: build and store exchange ---
	if inspectBuf != nil {
		ex := r.buildCapturedExchangeFromResponse(tunnel.ID, traceID, req, startTime, capturedReqBody, remoteAddr, resp, capturedRespBuf.Bytes())
		inspectBuf.Add(ex)
		r.log.Debug().
			Str("tunnel_id", tunnel.ID).
			Str("exchange_id", ex.ID).
			Int("buffer_len", inspectBuf.Len()).
			Msg("Exchange captured in inspect buffer")
	}

	r.log.Debug().
		Str("trace_id", traceID).
		Str("subdomain", subdomain).
		Str("method", req.Method).
		Str("path", req.URL.Path).
		Int("status", resp.StatusCode).
		Msg("HTTP request completed")
}

// isUpgradeRequest returns true if the request is a WebSocket or other HTTP upgrade.
func isUpgradeRequest(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("Connection"), "Upgrade") ||
		strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}

// serveUpgrade hijacks the connection and performs bidirectional proxying
// for WebSocket and other HTTP upgrade protocols.
func (r *HTTPRouter) serveUpgrade(w http.ResponseWriter, req *http.Request, stream net.Conn) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		r.log.Error().Msg("ResponseWriter does not support hijacking for upgrade")
		r.serveErrorPage(w, http.StatusInternalServerError, "Upgrade not supported")
		return
	}

	// Write the HTTP request to the tunnel stream
	if err := req.Write(stream); err != nil {
		r.log.Error().Err(err).Msg("Failed to write upgrade request to stream")
		r.serveErrorPage(w, http.StatusBadGateway, "Failed to proxy upgrade request")
		return
	}

	// Hijack the client connection
	clientConn, clientBuf, err := hj.Hijack()
	if err != nil {
		r.log.Error().Err(err).Msg("Failed to hijack connection for upgrade")
		return
	}
	defer clientConn.Close()

	r.log.Debug().
		Str("upgrade", req.Header.Get("Upgrade")).
		Str("path", req.URL.Path).
		Msg("WebSocket/Upgrade connection established")

	// Bidirectional copy between hijacked client conn and tunnel stream
	var wg sync.WaitGroup
	wg.Add(2)

	// stream → client (tunnel response back to browser)
	go func() {
		defer wg.Done()
		bp := proxyBufPool.Get().(*[]byte)
		_, _ = io.CopyBuffer(clientConn, stream, *bp)
		proxyBufPool.Put(bp)
		// Close write side to signal EOF
		if tc, ok := clientConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	// client → stream (flush any buffered data, then copy)
	go func() {
		defer wg.Done()
		// Flush any data already buffered by the http server
		if clientBuf.Reader.Buffered() > 0 {
			buffered := make([]byte, clientBuf.Reader.Buffered())
			n, _ := clientBuf.Read(buffered)
			if n > 0 {
				_, _ = stream.Write(buffered[:n])
			}
		}
		bp := proxyBufPool.Get().(*[]byte)
		_, _ = io.CopyBuffer(stream, clientConn, *bp)
		proxyBufPool.Put(bp)
		// Close write side to signal EOF
		if cs, ok := stream.(interface{ CloseWrite() error }); ok {
			_ = cs.CloseWrite()
		}
	}()

	wg.Wait()
}

// extractSubdomain extracts the subdomain from the host
func (r *HTTPRouter) extractSubdomain(host string) string {
	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	baseDomain := r.server.cfg.Domain.Base

	// Check if host ends with base domain
	if !strings.HasSuffix(host, "."+baseDomain) && host != baseDomain {
		// Try without www
		host = strings.TrimPrefix(host, "www.")
		if !strings.HasSuffix(host, "."+baseDomain) {
			return ""
		}
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, "."+baseDomain)
	if subdomain == host || subdomain == "" || subdomain == "www" {
		return ""
	}

	return strings.ToLower(subdomain)
}

// shouldShowInterstitial determines if an interstitial warning page should be shown
func (r *HTTPRouter) shouldShowInterstitial(req *http.Request, subdomain string) bool {
	if req.Method != http.MethodGet {
		return false
	}

	accept := req.Header.Get("Accept")
	if accept != "" && !strings.Contains(accept, "text/html") && !strings.Contains(accept, "*/*") {
		return false
	}

	if req.Header.Get("X-FxTunnel-Skip-Warning") != "" {
		return false
	}

	cookieName := "_fxt_consent_" + subdomain
	if cookie, err := req.Cookie(cookieName); err == nil && cookie.Value == "1" {
		return false
	}

	return true
}

// interstitialTexts holds localized strings for the interstitial page
type interstitialTexts struct {
	Lang, Title, Text, Button string
}

var interstitialLocales = map[string]interstitialTexts{
	"en": {
		Lang:   "en",
		Title:  "Dev Tunnel Warning",
		Text:   "You are about to visit a site served through a developer tunnel. This content is provided by a third party and is not verified. Do not enter sensitive information unless you trust the tunnel owner.",
		Button: "Continue to site",
	},
	"ru": {
		Lang:   "ru",
		Title:  "Предупреждение",
		Text:   "Вы собираетесь посетить сайт, работающий через туннель разработчика. Содержимое предоставлено третьей стороной и не проверено. Не вводите конфиденциальные данные, если вы не доверяете владельцу туннеля.",
		Button: "Продолжить",
	},
}

// detectLanguage returns "ru" or "en" based on Accept-Language header
func detectLanguage(req *http.Request) string {
	accept := req.Header.Get("Accept-Language")
	if strings.Contains(accept, "ru") {
		return "ru"
	}
	return "en"
}

// interstitialData holds template data for the interstitial page
type interstitialData struct {
	Lang, Title, Host, Text, Subdomain, Button string
}

// serveInterstitialPage serves the interstitial warning page via http.ResponseWriter
func (r *HTTPRouter) serveInterstitialPage(w http.ResponseWriter, req *http.Request, subdomain string) {
	lang := detectLanguage(req)
	texts := interstitialLocales[lang]

	var buf bytes.Buffer
	_ = interstitialTmpl.Execute(&buf, interstitialData{
		Lang:      texts.Lang,
		Title:     texts.Title,
		Host:      req.Host,
		Text:      texts.Text,
		Subdomain: subdomain,
		Button:    texts.Button,
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// errorData holds template data for the error page
type errorData struct {
	StatusCode int
	StatusText string
	Message    string
}

// serveErrorPage serves an error page via http.ResponseWriter
func (r *HTTPRouter) serveErrorPage(w http.ResponseWriter, status int, message string) {
	var buf bytes.Buffer
	_ = errorTmpl.Execute(&buf, errorData{
		StatusCode: status,
		StatusText: http.StatusText(status),
		Message:    message,
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}


// buildCapturedExchangeFromResponse constructs a CapturedExchange from a parsed HTTP response.
func (r *HTTPRouter) buildCapturedExchangeFromResponse(tunnelID, traceID string, req *http.Request, startTime time.Time, reqBody []byte, remoteAddr string, resp *http.Response, respBody []byte) *inspect.CapturedExchange {
	ex := &inspect.CapturedExchange{
		ID:              generateID(),
		TunnelID:        tunnelID,
		TraceID:         traceID,
		Timestamp:       startTime,
		Duration:        time.Since(startTime),
		Method:          req.Method,
		Path:            req.URL.RequestURI(),
		Host:            req.Host,
		RequestHeaders:  req.Header.Clone(),
		RequestBody:     reqBody,
		RequestBodySize: int64(len(reqBody)),
		RemoteAddr:      remoteAddr,
		StatusCode:      resp.StatusCode,
		ResponseHeaders: resp.Header.Clone(),
		ResponseBody:    respBody,
		ResponseBodySize: int64(len(respBody)),
	}

	if resp.ContentLength > int64(len(respBody)) {
		ex.ResponseBodySize = resp.ContentLength
	}

	return ex
}

// ReplayRequest sends an HTTP request through a tunnel and returns the response.
// Used by the inspect replay feature.
func (r *HTTPRouter) ReplayRequest(subdomain string, req *http.Request) (*http.Response, error) {
	tunnel := r.GetTunnel(subdomain)
	if tunnel == nil {
		return nil, fmt.Errorf("tunnel not found for subdomain: %s", subdomain)
	}

	client := r.server.GetClient(tunnel.ClientID)
	if client == nil {
		return nil, fmt.Errorf("client not connected for tunnel: %s", tunnel.ID)
	}

	stream, err := client.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}
	defer stream.Close()

	// Send binary stream header
	if err := protocol.WriteStreamHeader(stream, tunnel.ID, "replay"); err != nil {
		return nil, fmt.Errorf("send connection info: %w", err)
	}

	if err := req.Write(stream); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	streamReader := bufio.NewReader(stream)
	resp, err := http.ReadResponse(streamReader, req)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return resp, nil
}

// limitedWriter writes up to `remaining` bytes, then silently discards the rest.
type limitedWriter struct {
	w         io.Writer
	remaining int
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.remaining <= 0 {
		return len(p), nil
	}
	n := len(p)
	if n > lw.remaining {
		n = lw.remaining
	}
	written, err := lw.w.Write(p[:n])
	lw.remaining -= written
	if err != nil {
		return written, err
	}
	return len(p), nil
}
