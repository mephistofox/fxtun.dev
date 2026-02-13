package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/inspect"
)

// Inspector is an embedded HTTP server that exposes a REST API for the local
// traffic inspector. It provides endpoints for listing, searching, replaying,
// and streaming captured HTTP exchanges.
type Inspector struct {
	manager     *inspect.Manager
	addr        string
	maxBodySize int
	startTime   time.Time
	tunnels     map[string]*ActiveTunnel
	tunnelsMu   *sync.RWMutex
	mux         *http.ServeMux
	server      *http.Server
	actualAddr  string
	log         zerolog.Logger

	// Global broadcast for SSE subscribers.
	sseSubsMu sync.RWMutex
	sseSubs   map[chan *inspect.CapturedExchange]struct{}
}

// NewInspector creates a new Inspector with all routes configured.
func NewInspector(manager *inspect.Manager, addr string, maxBodySize int, log zerolog.Logger) *Inspector {
	i := &Inspector{
		manager:     manager,
		addr:        addr,
		maxBodySize: maxBodySize,
		startTime:   time.Now(),
		mux:         http.NewServeMux(),
		log:         log.With().Str("component", "inspector").Logger(),
		sseSubs:     make(map[chan *inspect.CapturedExchange]struct{}),
	}

	// Register routes. summary must be registered before {id} to be safe.
	i.mux.HandleFunc("GET /api/requests/http/summary", i.handleSummary)
	i.mux.HandleFunc("GET /api/requests/http/stream", i.handleSSEStream)
	i.mux.HandleFunc("GET /api/requests/http/{id}", i.handleGetExchange)
	i.mux.HandleFunc("GET /api/requests/http", i.handleListExchanges)
	i.mux.HandleFunc("POST /api/requests/http", i.handleReplay)
	i.mux.HandleFunc("DELETE /api/requests/http", i.handleDeleteExchanges)
	i.mux.HandleFunc("GET /api/tunnels", i.handleListTunnels)
	i.mux.HandleFunc("GET /api/status", i.handleStatus)

	// Serve embedded UI files with no-cache to prevent stale JS.
	uiFS, err := fs.Sub(inspectorUIFS, "inspector_ui")
	if err == nil {
		i.mux.Handle("/", noCacheMiddleware(http.FileServerFS(uiFS)))
	}

	return i
}

// ServeHTTP implements http.Handler with CORS middleware.
func (i *Inspector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}
	i.mux.ServeHTTP(w, r)
}

// Start starts the inspector HTTP server. It tries the configured address first,
// then falls back to ports +1 through +9 if the port is busy.
func (i *Inspector) Start(ctx context.Context) error {
	host, portStr, err := net.SplitHostPort(i.addr)
	if err != nil {
		return fmt.Errorf("invalid inspector address %q: %w", i.addr, err)
	}

	basePort, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid inspector port %q: %w", portStr, err)
	}

	var ln net.Listener
	for offset := 0; offset < 10; offset++ {
		tryAddr := net.JoinHostPort(host, strconv.Itoa(basePort+offset))
		ln, err = net.Listen("tcp", tryAddr)
		if err == nil {
			break
		}
		i.log.Debug().Str("addr", tryAddr).Err(err).Msg("Port busy, trying next")
	}
	if ln == nil {
		return fmt.Errorf("failed to bind inspector on ports %d-%d: %w", basePort, basePort+9, err)
	}

	i.actualAddr = ln.Addr().String()
	i.server = &http.Server{
		Handler:           i,
		ReadHeaderTimeout: 10 * time.Second,
	}

	i.log.Info().Str("addr", i.actualAddr).Msg("Inspector started")

	go func() {
		if err := i.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			i.log.Error().Err(err).Msg("Inspector server error")
		}
	}()

	return nil
}

// Stop gracefully shuts down the inspector HTTP server.
func (i *Inspector) Stop() error {
	if i.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return i.server.Shutdown(ctx)
}

// Addr returns the actual address the inspector is listening on.
func (i *Inspector) Addr() string {
	return i.actualAddr
}

// AddExchange adds a captured exchange to the appropriate tunnel buffer
// and broadcasts to all SSE subscribers.
func (i *Inspector) AddExchange(ex *inspect.CapturedExchange) {
	buf := i.manager.GetOrCreate(ex.TunnelID)
	if buf != nil {
		buf.Add(ex)
	}
	// Broadcast to SSE subscribers.
	i.sseSubsMu.RLock()
	for ch := range i.sseSubs {
		select {
		case ch <- ex:
		default:
		}
	}
	i.sseSubsMu.RUnlock()
}

func noCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

// SetTunnels gives the inspector access to the client's active tunnels.
func (i *Inspector) SetTunnels(tunnels map[string]*ActiveTunnel, mu *sync.RWMutex) {
	i.tunnels = tunnels
	i.tunnelsMu = mu
}

// --- JSON response helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- Handlers ---

// exchangeListItem is a JSON representation of a captured exchange for list responses.
type exchangeListItem struct {
	ID               string `json:"id"`
	TunnelID         string `json:"tunnel_id"`
	Method           string `json:"method"`
	Path             string `json:"path"`
	Host             string `json:"host"`
	StatusCode       int    `json:"status_code"`
	DurationMS       int64  `json:"duration_ms"`
	Timestamp        string `json:"timestamp"`
	RequestBodySize  int64  `json:"request_body_size"`
	ResponseBodySize int64  `json:"response_body_size"`
	RemoteAddr       string `json:"remote_addr"`

	// Included only when include_body=true.
	RequestBody  *string `json:"request_body,omitempty"`
	ResponseBody *string `json:"response_body,omitempty"`
}

func exchangeToListItem(ex *inspect.CapturedExchange, includeBody bool) exchangeListItem {
	item := exchangeListItem{
		ID:               ex.ID,
		TunnelID:         ex.TunnelID,
		Method:           ex.Method,
		Path:             ex.Path,
		Host:             ex.Host,
		StatusCode:       ex.StatusCode,
		DurationMS:       ex.Duration.Milliseconds(),
		Timestamp:        ex.Timestamp.UTC().Format(time.RFC3339Nano),
		RequestBodySize:  ex.RequestBodySize,
		ResponseBodySize: ex.ResponseBodySize,
		RemoteAddr:       ex.RemoteAddr,
	}
	if includeBody {
		reqBody := base64.StdEncoding.EncodeToString(ex.RequestBody)
		respBody := base64.StdEncoding.EncodeToString(ex.ResponseBody)
		item.RequestBody = &reqBody
		item.ResponseBody = &respBody
	}
	return item
}

func (i *Inspector) handleListExchanges(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := intParam(q.Get("limit"), 50)
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := intParam(q.Get("offset"), 0)
	if offset < 0 {
		offset = 0
	}

	filterMethod := q.Get("method")
	filterStatus := q.Get("status")
	filterPath := q.Get("path")
	filterSearch := q.Get("search")
	filterTunnel := q.Get("tunnel_name")
	includeBody := q.Get("include_body") == "true"

	var filterSince time.Time
	if sinceStr := q.Get("since"); sinceStr != "" {
		dur, err := time.ParseDuration(sinceStr)
		if err == nil {
			filterSince = time.Now().Add(-dur)
		}
	}

	// Collect all exchanges from all buffers.
	var all []*inspect.CapturedExchange
	i.manager.ForEach(func(tunnelID string, buf *inspect.RingBuffer) {
		entries := buf.List(0, buf.Len())
		all = append(all, entries...)
	})

	// Apply filters.
	filtered := make([]*inspect.CapturedExchange, 0, len(all))
	for _, ex := range all {
		if filterMethod != "" && !strings.EqualFold(ex.Method, filterMethod) {
			continue
		}
		if filterStatus != "" && !matchStatus(ex.StatusCode, filterStatus) {
			continue
		}
		if filterPath != "" {
			matched, _ := path.Match(filterPath, ex.Path)
			if !matched {
				continue
			}
		}
		if filterSearch != "" {
			found := strings.Contains(string(ex.RequestBody), filterSearch) ||
				strings.Contains(string(ex.ResponseBody), filterSearch)
			if !found {
				continue
			}
		}
		if !filterSince.IsZero() && ex.Timestamp.Before(filterSince) {
			continue
		}
		if filterTunnel != "" {
			if !i.tunnelNameMatches(ex.TunnelID, filterTunnel) {
				continue
			}
		}
		filtered = append(filtered, ex)
	}

	// Sort by timestamp descending (newest first).
	sort.Slice(filtered, func(a, b int) bool {
		return filtered[a].Timestamp.After(filtered[b].Timestamp)
	})

	total := len(filtered)

	// Apply pagination.
	if offset > len(filtered) {
		filtered = nil
	} else {
		filtered = filtered[offset:]
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	items := make([]exchangeListItem, 0, len(filtered))
	for _, ex := range filtered {
		items = append(items, exchangeToListItem(ex, includeBody))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"requests": items,
		"total":    total,
	})
}

func (i *Inspector) handleGetExchange(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var found *inspect.CapturedExchange
	i.manager.ForEach(func(_ string, buf *inspect.RingBuffer) {
		if found != nil {
			return
		}
		if ex := buf.Get(id); ex != nil {
			found = ex
		}
	})

	if found == nil {
		writeError(w, http.StatusNotFound, "exchange not found")
		return
	}

	writeJSON(w, http.StatusOK, found)
}

func (i *Inspector) handleDeleteExchanges(w http.ResponseWriter, _ *http.Request) {
	i.manager.ForEach(func(_ string, buf *inspect.RingBuffer) {
		buf.Clear()
	})
	w.WriteHeader(http.StatusNoContent)
}

// summaryResponse is returned by GET /api/requests/http/summary.
type summaryResponse struct {
	Total         int            `json:"total"`
	ByStatus      map[string]int `json:"by_status"`
	ByMethod      map[string]int `json:"by_method"`
	ErrorRate     float64        `json:"error_rate"`
	AvgDurationMS int64          `json:"avg_duration_ms"`
	LastRequestAt *string        `json:"last_request_at"`
}

func (i *Inspector) handleSummary(w http.ResponseWriter, _ *http.Request) {
	byStatus := map[string]int{"2xx": 0, "3xx": 0, "4xx": 0, "5xx": 0}
	byMethod := map[string]int{}
	var total int
	var totalDuration int64
	var lastAt time.Time

	i.manager.ForEach(func(_ string, buf *inspect.RingBuffer) {
		entries := buf.List(0, buf.Len())
		for _, ex := range entries {
			total++
			totalDuration += ex.Duration.Milliseconds()

			byMethod[ex.Method]++

			switch {
			case ex.StatusCode >= 200 && ex.StatusCode < 300:
				byStatus["2xx"]++
			case ex.StatusCode >= 300 && ex.StatusCode < 400:
				byStatus["3xx"]++
			case ex.StatusCode >= 400 && ex.StatusCode < 500:
				byStatus["4xx"]++
			case ex.StatusCode >= 500 && ex.StatusCode < 600:
				byStatus["5xx"]++
			}

			if ex.Timestamp.After(lastAt) {
				lastAt = ex.Timestamp
			}
		}
	})

	resp := summaryResponse{
		Total:    total,
		ByStatus: byStatus,
		ByMethod: byMethod,
	}

	if total > 0 {
		errorCount := byStatus["4xx"] + byStatus["5xx"]
		resp.ErrorRate = float64(errorCount) / float64(total)
		resp.AvgDurationMS = totalDuration / int64(total)
	}

	if !lastAt.IsZero() {
		s := lastAt.UTC().Format(time.RFC3339)
		resp.LastRequestAt = &s
	}

	writeJSON(w, http.StatusOK, resp)
}

func (i *Inspector) handleSSEStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Subscribe to the global broadcast channel.
	ch := make(chan *inspect.CapturedExchange, 128)
	i.sseSubsMu.Lock()
	i.sseSubs[ch] = struct{}{}
	i.sseSubsMu.Unlock()

	defer func() {
		i.sseSubsMu.Lock()
		delete(i.sseSubs, ch)
		i.sseSubsMu.Unlock()
	}()

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ex := <-ch:
			data, err := json.Marshal(ex.Summary())
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: exchange\ndata: %s\n\n", data)
			flusher.Flush()
		case <-pingTicker.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (i *Inspector) handleListTunnels(w http.ResponseWriter, _ *http.Request) {
	type tunnelInfo struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		URL       string `json:"url,omitempty"`
		LocalPort int    `json:"local_port"`
	}

	var tunnels []tunnelInfo
	if i.tunnelsMu != nil {
		i.tunnelsMu.RLock()
		for _, t := range i.tunnels {
			tunnels = append(tunnels, tunnelInfo{
				ID:        t.ID,
				Name:      t.Config.Name,
				Type:      t.Config.Type,
				URL:       t.URL,
				LocalPort: t.Config.LocalPort,
			})
		}
		i.tunnelsMu.RUnlock()
	}

	if tunnels == nil {
		tunnels = []tunnelInfo{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tunnels": tunnels,
	})
}

func (i *Inspector) handleStatus(w http.ResponseWriter, _ *http.Request) {
	var totalExchanges int
	i.manager.ForEach(func(_ string, buf *inspect.RingBuffer) {
		totalExchanges += buf.Len()
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"version":          "dev",
		"uptime_seconds":   int(time.Since(i.startTime).Seconds()),
		"inspect_enabled":  i.manager.Enabled(),
		"total_exchanges":  totalExchanges,
	})
}

// replayRequest is the JSON body for POST /api/requests/http.
type replayRequest struct {
	ID      string            `json:"id"`
	Method  string            `json:"method,omitempty"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"` // base64 encoded
}

func (i *Inspector) handleReplay(w http.ResponseWriter, r *http.Request) {
	var req replayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Find original exchange.
	var original *inspect.CapturedExchange
	i.manager.ForEach(func(_ string, buf *inspect.RingBuffer) {
		if original != nil {
			return
		}
		if ex := buf.Get(req.ID); ex != nil {
			original = ex
		}
	})

	if original == nil {
		writeError(w, http.StatusNotFound, "exchange not found")
		return
	}

	// Determine local address from the tunnel.
	localAddr := i.resolveLocalAddr(original.TunnelID)
	if localAddr == "" {
		writeError(w, http.StatusBadRequest, "tunnel not found or no local address")
		return
	}

	// Apply modifications.
	method := original.Method
	if req.Method != "" {
		method = req.Method
	}
	reqPath := original.Path
	if req.Path != "" {
		reqPath = req.Path
	}

	var body io.Reader
	if req.Body != "" {
		decoded, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid base64 body")
			return
		}
		body = strings.NewReader(string(decoded))
	} else if original.RequestBody != nil {
		body = strings.NewReader(string(original.RequestBody))
	}

	url := fmt.Sprintf("http://%s%s", localAddr, reqPath)
	httpReq, err := http.NewRequestWithContext(r.Context(), method, url, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create request")
		return
	}

	// Copy original headers, then apply overrides.
	for k, vals := range original.RequestHeaders {
		for _, v := range vals {
			httpReq.Header.Add(k, v)
		}
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Send request.
	start := time.Now()
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("request to local service failed: %v", err))
		return
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, int64(i.maxBodySize)))
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to read response body")
		return
	}

	// Build request body for the captured exchange.
	var capturedReqBody []byte
	if req.Body != "" {
		capturedReqBody, _ = base64.StdEncoding.DecodeString(req.Body)
	} else {
		capturedReqBody = original.RequestBody
	}

	// Create new exchange.
	newEx := &inspect.CapturedExchange{
		ID:               generateID(),
		TunnelID:         original.TunnelID,
		ReplayRef:        original.ID,
		Timestamp:        time.Now(),
		Duration:         duration,
		Method:           method,
		Path:             reqPath,
		Host:             original.Host,
		RequestHeaders:   httpReq.Header,
		RequestBody:      capturedReqBody,
		RequestBodySize:  int64(len(capturedReqBody)),
		StatusCode:       resp.StatusCode,
		ResponseHeaders:  resp.Header,
		ResponseBody:     respBody,
		ResponseBodySize: int64(len(respBody)),
	}

	i.AddExchange(newEx)

	// Build response headers map for JSON.
	respHeaders := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status_code":      resp.StatusCode,
		"response_headers": respHeaders,
		"response_body":    base64.StdEncoding.EncodeToString(respBody),
		"exchange_id":      newEx.ID,
	})
}

// --- Helpers ---

func intParam(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// matchStatus checks if a status code matches a filter.
// The filter can be a category like "2xx", "3xx", "4xx", "5xx" or an exact code like "404".
func matchStatus(code int, filter string) bool {
	switch strings.ToLower(filter) {
	case "2xx":
		return code >= 200 && code < 300
	case "3xx":
		return code >= 300 && code < 400
	case "4xx":
		return code >= 400 && code < 500
	case "5xx":
		return code >= 500 && code < 600
	default:
		exact, err := strconv.Atoi(filter)
		if err != nil {
			return false
		}
		return code == exact
	}
}

func (i *Inspector) tunnelNameMatches(tunnelID, name string) bool {
	if i.tunnelsMu == nil {
		return false
	}
	i.tunnelsMu.RLock()
	defer i.tunnelsMu.RUnlock()
	t, ok := i.tunnels[tunnelID]
	if !ok {
		return false
	}
	return t.Config.Name == name
}

func (i *Inspector) resolveLocalAddr(tunnelID string) string {
	if i.tunnelsMu == nil {
		return ""
	}
	i.tunnelsMu.RLock()
	defer i.tunnelsMu.RUnlock()
	t, ok := i.tunnels[tunnelID]
	if !ok {
		return ""
	}
	return t.Config.GetLocalAddress()
}
