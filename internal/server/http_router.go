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

	"github.com/mephistofox/fxtunnel/internal/protocol"
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

	// Bidirectional copy - wait for BOTH directions to complete
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(stream, conn)
		// Signal EOF to the client by closing write side
		if tcpConn, ok := stream.(interface{ CloseWrite() error }); ok {
			tcpConn.CloseWrite()
		}
	}()

	go func() {
		defer wg.Done()
		io.Copy(conn, stream)
	}()

	// Wait for both directions to complete
	wg.Wait()

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

// sendErrorResponse sends an HTTP error response
func (r *HTTPRouter) sendErrorResponse(conn net.Conn, status int, message string) {
	body := r.buildErrorPage(status, message)

	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n"+
		"Content-Type: text/html; charset=utf-8\r\n"+
		"Content-Length: %d\r\n"+
		"Connection: close\r\n"+
		"\r\n%s", status, http.StatusText(status), len(body), body)

	conn.Write([]byte(response))
}

// buildErrorPage generates a styled error page
func (r *HTTPRouter) buildErrorPage(status int, message string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%d %s | fxTunnel</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Onest:wght@400;500;600;700&family=Unbounded:wght@500;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --background: hsl(220 20%% 4%%);
            --foreground: hsl(0 0%% 95%%);
            --primary: hsl(75 100%% 50%%);
            --primary-dim: hsl(75 80%% 35%%);
            --accent: hsl(280 100%% 65%%);
            --muted: hsl(220 10%% 55%%);
            --card: hsl(220 15%% 8%%);
            --border: hsl(220 15%% 15%%);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        html, body {
            overflow: hidden;
            width: 100%%;
            height: 100%%;
        }

        body {
            min-height: 100vh;
            min-height: 100dvh;
            display: flex;
            align-items: center;
            justify-content: center;
            background: var(--background);
            color: var(--foreground);
            font-family: 'Onest', system-ui, sans-serif;
            position: relative;
        }

        /* Animated grid background */
        .grid-bg {
            position: absolute;
            inset: 0;
            background-image:
                linear-gradient(var(--border) 1px, transparent 1px),
                linear-gradient(90deg, var(--border) 1px, transparent 1px);
            background-size: 60px 60px;
            opacity: 0.3;
            animation: grid-move 20s linear infinite;
        }

        @keyframes grid-move {
            0%% { transform: translate(0, 0); }
            100%% { transform: translate(60px, 60px); }
        }

        /* Glowing orbs */
        .orb {
            position: absolute;
            border-radius: 50%%;
            filter: blur(80px);
            opacity: 0.4;
            animation: float 8s ease-in-out infinite;
        }

        .orb-1 {
            width: 400px;
            height: 400px;
            background: var(--primary);
            top: -200px;
            right: -100px;
            animation-delay: 0s;
        }

        .orb-2 {
            width: 300px;
            height: 300px;
            background: var(--accent);
            bottom: -150px;
            left: -100px;
            animation-delay: -4s;
        }

        @keyframes float {
            0%%, 100%% { transform: translate(0, 0) scale(1); }
            50%% { transform: translate(20px, -20px) scale(1.05); }
        }

        /* Mobile: smaller orbs, no animation */
        @media (max-width: 640px) {
            .orb-1 {
                width: 200px;
                height: 200px;
                top: -100px;
                right: -50px;
                animation: none;
            }
            .orb-2 {
                width: 150px;
                height: 150px;
                bottom: -75px;
                left: -50px;
                animation: none;
            }
            .grid-bg {
                animation: none;
            }
            .scanline {
                display: none;
            }
        }

        .container {
            position: relative;
            z-index: 10;
            text-align: center;
            padding: 2rem;
        }

        .error-code {
            font-family: 'Unbounded', sans-serif;
            font-size: clamp(8rem, 20vw, 14rem);
            font-weight: 700;
            line-height: 1;
            background: linear-gradient(135deg, var(--primary) 0%%, var(--accent) 100%%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            text-shadow: 0 0 80px hsla(75, 100%%, 50%%, 0.3);
            animation: glow-pulse 3s ease-in-out infinite;
        }

        @keyframes glow-pulse {
            0%%, 100%% { filter: brightness(1); }
            50%% { filter: brightness(1.2); }
        }

        .error-title {
            font-family: 'Unbounded', sans-serif;
            font-size: clamp(1.5rem, 4vw, 2.5rem);
            font-weight: 500;
            margin-top: 1rem;
            color: var(--foreground);
        }

        .error-message {
            font-size: 1.125rem;
            color: var(--muted);
            margin-top: 1rem;
            max-width: 400px;
            margin-left: auto;
            margin-right: auto;
        }

        .card {
            margin-top: 2.5rem;
            padding: 1.5rem 2rem;
            background: var(--card);
            border: 1px solid var(--border);
            border-radius: 1rem;
            display: inline-block;
        }

        .card-content {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            color: var(--muted);
            font-size: 0.95rem;
        }

        .pulse-dot {
            width: 10px;
            height: 10px;
            background: var(--primary);
            border-radius: 50%%;
            animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
            0%%, 100%% { opacity: 1; transform: scale(1); }
            50%% { opacity: 0.5; transform: scale(0.8); }
        }

        .brand {
            margin-top: 3rem;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 0.5rem;
            color: var(--muted);
            font-size: 0.875rem;
        }

        .brand-logo {
            width: 24px;
            height: 24px;
            background: linear-gradient(135deg, var(--primary) 0%%, var(--accent) 100%%);
            border-radius: 6px;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .brand-logo svg {
            width: 14px;
            height: 14px;
        }

        .brand-name {
            font-family: 'Unbounded', sans-serif;
            font-weight: 500;
            color: var(--foreground);
        }

        /* Scan line effect */
        .scanline {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 4px;
            background: linear-gradient(90deg, transparent, var(--primary), transparent);
            opacity: 0.1;
            animation: scan 4s linear infinite;
        }

        @keyframes scan {
            0%% { top: 0; }
            100%% { top: 100%%; }
        }
    </style>
</head>
<body>
    <div class="grid-bg"></div>
    <div class="orb orb-1"></div>
    <div class="orb orb-2"></div>
    <div class="scanline"></div>

    <div class="container">
        <div class="error-code">%d</div>
        <h1 class="error-title">%s</h1>
        <p class="error-message">%s</p>

        <div class="card">
            <div class="card-content">
                <div class="pulse-dot"></div>
                <span>No active tunnel on this subdomain</span>
            </div>
        </div>

        <div class="brand">
            <div class="brand-logo">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="hsl(220, 20%%, 4%%)"/>
                </svg>
            </div>
            <span>Powered by <span class="brand-name">fxTunnel</span></span>
        </div>
    </div>
</body>
</html>`, status, http.StatusText(status), status, http.StatusText(status), message)
}
