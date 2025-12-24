package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/fxcode/fxtunnel/internal/protocol"
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

	// Write the original request to the stream
	if err := req.Write(stream); err != nil {
		r.log.Error().Err(err).Msg("Failed to write request to stream")
		return
	}

	// Also write any buffered data from the reader
	if reader.Buffered() > 0 {
		buffered := make([]byte, reader.Buffered())
		reader.Read(buffered)
		stream.Write(buffered)
	}

	// Bidirectional copy
	done := make(chan struct{})

	go func() {
		io.Copy(stream, conn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(conn, stream)
		done <- struct{}{}
	}()

	// Wait for either direction to finish
	<-done

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
		if strings.HasPrefix(host, "www.") {
			host = host[4:]
		}
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

// sendErrorResponse sends an HTTP error response
func (r *HTTPRouter) sendErrorResponse(conn net.Conn, status int, message string) {
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>%d %s</title></head>
<body>
<h1>%d %s</h1>
<p>%s</p>
<hr>
<p>fxTunnel</p>
</body>
</html>`, status, http.StatusText(status), status, http.StatusText(status), message)

	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n"+
		"Content-Type: text/html; charset=utf-8\r\n"+
		"Content-Length: %d\r\n"+
		"Connection: close\r\n"+
		"\r\n%s", status, http.StatusText(status), len(body), body)

	conn.Write([]byte(response))
}
