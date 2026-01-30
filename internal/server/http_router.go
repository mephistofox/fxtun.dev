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

// HTTPRouter routes HTTP requests to the appropriate tunnel
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

// HandleConnection handles an incoming HTTP connection
func (r *HTTPRouter) HandleConnection(conn net.Conn) {
	defer conn.Close()

	tuneTCPConn(conn)

	// Set initial read deadline for parsing request
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read the HTTP request to extract Host header
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		r.log.Debug().Err(err).Msg("Failed to read HTTP request")
		r.sendErrorResponse(conn, http.StatusBadRequest, "Bad Request")
		return
	}

	conn.SetReadDeadline(time.Time{}) // Clear deadline

	// Extract subdomain from Host header
	subdomain := r.extractSubdomain(req.Host)
	if subdomain == "" {
		r.log.Debug().Str("host", req.Host).Msg("No subdomain in request")
		r.sendErrorResponse(conn, http.StatusNotFound, "Tunnel not found")
		return
	}

	// Find tunnel
	tunnel := r.GetTunnel(subdomain)
	if tunnel == nil {
		r.log.Debug().Str("subdomain", subdomain).Msg("Tunnel not found")
		r.sendErrorResponse(conn, http.StatusNotFound, "Tunnel not found")
		return
	}

	// Get client
	client := r.server.GetClient(tunnel.ClientID)
	if client == nil {
		r.log.Warn().Str("client_id", tunnel.ClientID).Msg("Client not found for tunnel")
		r.sendErrorResponse(conn, http.StatusBadGateway, "Tunnel unavailable")
		return
	}

	// Check for interstitial warning (skip for admin tunnels)
	if !client.IsAdmin && r.shouldShowInterstitial(req, subdomain) {
		r.sendInterstitialResponse(conn, req, subdomain)
		return
	}

	// Open stream to client
	stream, err := client.OpenStream()
	if err != nil {
		r.log.Error().Err(err).Msg("Failed to open stream to client")
		r.sendErrorResponse(conn, http.StatusBadGateway, "Failed to connect to tunnel")
		return
	}
	defer stream.Close()

	// Notify client about new connection
	connID := generateID()
	newConn := &protocol.NewConnectionMessage{
		Message:      protocol.NewMessage(protocol.MsgNewConnection),
		TunnelID:     tunnel.ID,
		ConnectionID: connID,
		RemoteAddr:   conn.RemoteAddr().String(),
		Host:         req.Host,
		Method:       req.Method,
		Path:         req.URL.Path,
	}

	// Send connection info through the stream first (as a header)
	streamCodec := protocol.NewCodec(stream, stream)
	if err := streamCodec.Encode(newConn); err != nil {
		r.log.Error().Err(err).Msg("Failed to send connection info")
		r.sendErrorResponse(conn, http.StatusBadGateway, "Failed to connect to tunnel")
		return
	}

	// Add forwarding headers
	clientIP := conn.RemoteAddr().String()
	if host, _, err := net.SplitHostPort(clientIP); err == nil {
		clientIP = host
	}
	if prior := req.Header.Get("X-Forwarded-For"); prior != "" {
		clientIP = prior + ", " + clientIP
	}
	req.Header.Set("X-Forwarded-For", clientIP)
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-Host", req.Host)

	// --- Inspection: capture request body ---
	inspectBuf := r.server.inspectMgr.Get(tunnel.ID)
	startTime := time.Now()
	var capturedReqBody []byte

	if inspectBuf != nil && req.Body != nil {
		maxBody := r.server.inspectMgr.MaxBodySize()
		capturedReqBody, _ = io.ReadAll(io.LimitReader(req.Body, int64(maxBody)))
		req.Body = io.NopCloser(bytes.NewReader(capturedReqBody))
	}

	// Write the original request to the stream
	if err := req.Write(stream); err != nil {
		r.log.Error().Err(err).Msg("Failed to write request to stream")
		return
	}

	// Also write any buffered data from the reader
	if reader.Buffered() > 0 {
		buffered := make([]byte, reader.Buffered())
		n, _ := reader.Read(buffered)
		if n > 0 {
			stream.Write(buffered[:n])
		}
	}

	// --- Bidirectional copy with optional response capture ---
	var capturedRespBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		bp := proxyBufPool.Get().(*[]byte)
		io.CopyBuffer(stream, conn, *bp)
		proxyBufPool.Put(bp)
		// Signal EOF to the client by closing write side
		if tcpConn, ok := stream.(interface{ CloseWrite() error }); ok {
			tcpConn.CloseWrite()
		}
	}()

	go func() {
		defer wg.Done()
		if inspectBuf != nil {
			maxCapture := r.server.inspectMgr.MaxBodySize() + 8192
			lw := &limitedWriter{w: &capturedRespBuf, remaining: maxCapture}
			tee := io.TeeReader(stream, lw)
			bp := proxyBufPool.Get().(*[]byte)
			io.CopyBuffer(conn, tee, *bp)
			proxyBufPool.Put(bp)
		} else {
			bp := proxyBufPool.Get().(*[]byte)
			io.CopyBuffer(conn, stream, *bp)
			proxyBufPool.Put(bp)
		}
	}()

	wg.Wait()

	// --- Inspection: build and store exchange ---
	if inspectBuf != nil {
		ex := r.buildCapturedExchange(tunnel.ID, req, startTime, capturedReqBody, conn.RemoteAddr().String(), &capturedRespBuf)
		inspectBuf.Add(ex)
	}

	r.log.Debug().
		Str("subdomain", subdomain).
		Str("method", req.Method).
		Str("path", req.URL.Path).
		Msg("HTTP request completed")
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

// sendInterstitialResponse sends the interstitial warning page
func (r *HTTPRouter) sendInterstitialResponse(conn net.Conn, req *http.Request, subdomain string) {
	lang := detectLanguage(req)
	texts := interstitialLocales[lang]

	var buf bytes.Buffer
	interstitialTmpl.Execute(&buf, interstitialData{
		Lang:      texts.Lang,
		Title:     texts.Title,
		Host:      req.Host,
		Text:      texts.Text,
		Subdomain: subdomain,
		Button:    texts.Button,
	})
	body := buf.Bytes()

	response := fmt.Sprintf("HTTP/1.1 200 OK\r\n"+
		"Content-Type: text/html; charset=utf-8\r\n"+
		"Content-Length: %d\r\n"+
		"Cache-Control: no-store\r\n"+
		"Connection: close\r\n"+
		"\r\n", len(body))

	conn.Write([]byte(response))
	conn.Write(body)
}

// errorData holds template data for the error page
type errorData struct {
	StatusCode int
	StatusText string
	Message    string
}

// limitedWriter writes up to `remaining` bytes, then silently discards.
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

// buildCapturedExchange constructs a CapturedExchange from captured data.
func (r *HTTPRouter) buildCapturedExchange(tunnelID string, req *http.Request, startTime time.Time, reqBody []byte, remoteAddr string, respBuf *bytes.Buffer) *inspect.CapturedExchange {
	ex := &inspect.CapturedExchange{
		ID:              generateID(),
		TunnelID:        tunnelID,
		Timestamp:       startTime,
		Duration:        time.Since(startTime),
		Method:          req.Method,
		Path:            req.URL.RequestURI(),
		Host:            req.Host,
		RequestHeaders:  req.Header.Clone(),
		RequestBody:     reqBody,
		RequestBodySize: int64(len(reqBody)),
		RemoteAddr:      remoteAddr,
	}

	// Parse response from captured bytes
	if respBuf.Len() > 0 {
		resp, err := http.ReadResponse(bufio.NewReader(respBuf), req)
		if err == nil {
			defer resp.Body.Close()
			ex.StatusCode = resp.StatusCode
			ex.ResponseHeaders = resp.Header.Clone()
			maxBody := r.server.inspectMgr.MaxBodySize()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxBody)))
			ex.ResponseBody = body
			ex.ResponseBodySize = int64(len(body))
			if resp.ContentLength > int64(len(body)) {
				ex.ResponseBodySize = resp.ContentLength
			}
		}
	}

	return ex
}

// sendErrorResponse sends an HTTP error response
func (r *HTTPRouter) sendErrorResponse(conn net.Conn, status int, message string) {
	var buf bytes.Buffer
	errorTmpl.Execute(&buf, errorData{
		StatusCode: status,
		StatusText: http.StatusText(status),
		Message:    message,
	})
	body := buf.Bytes()

	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n"+
		"Content-Type: text/html; charset=utf-8\r\n"+
		"Content-Length: %d\r\n"+
		"Connection: close\r\n"+
		"\r\n", status, http.StatusText(status), len(body))

	conn.Write([]byte(response))
	conn.Write(body)
}

