package api

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/inspect"
)

func (s *Server) checkInspectorAccess(w http.ResponseWriter, user *auth.AuthenticatedUser) bool {
	if !user.IsAdmin && (user.Plan == nil || !user.Plan.InspectorEnabled) {
		s.respondErrorWithCode(w, http.StatusForbidden, "INSPECTOR_DISABLED", "inspector not available on your plan")
		return false
	}
	return true
}

func (s *Server) handleListExchanges(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.checkInspectorAccess(w, user) {
		return
	}

	tunnelID := chi.URLParam(r, "id")
	if err := s.checkTunnelAccess(tunnelID, user); err != nil {
		s.respondError(w, http.StatusForbidden, err.Error())
		return
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	buf := s.getInspectBuffer(tunnelID)
	if buf != nil {
		// Tunnel is online — use in-memory buffer
		exchanges := buf.List(offset, limit)
		summaries := make([]inspect.ExchangeSummary, len(exchanges))
		for i, ex := range exchanges {
			summaries[i] = ex.Summary()
		}
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"exchanges": summaries,
			"total":     buf.Len(),
		})
		return
	}

	// Tunnel offline — fallback to persisted data
	if s.inspectProvider != nil {
		exchanges, total, err := s.inspectProvider.ListPersisted(tunnelID, offset, limit)
		if err == nil && len(exchanges) > 0 {
			summaries := make([]inspect.ExchangeSummary, len(exchanges))
			for i, ex := range exchanges {
				summaries[i] = ex.Summary()
			}
			s.respondJSON(w, http.StatusOK, map[string]interface{}{
				"exchanges": summaries,
				"total":     total,
			})
			return
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"exchanges": []interface{}{},
		"total":     0,
	})
}

func (s *Server) handleGetExchange(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.checkInspectorAccess(w, user) {
		return
	}

	tunnelID := chi.URLParam(r, "id")
	if err := s.checkTunnelAccess(tunnelID, user); err != nil {
		s.respondError(w, http.StatusForbidden, err.Error())
		return
	}

	exchangeID := chi.URLParam(r, "exchangeId")

	// Try in-memory buffer first
	buf := s.getInspectBuffer(tunnelID)
	if buf != nil {
		if ex := buf.Get(exchangeID); ex != nil {
			s.respondJSON(w, http.StatusOK, ex)
			return
		}
	}

	// Fallback to persisted data
	if s.inspectProvider != nil {
		ex, err := s.inspectProvider.GetPersisted(exchangeID)
		if err == nil && ex != nil {
			s.respondJSON(w, http.StatusOK, ex)
			return
		}
	}

	s.respondError(w, http.StatusNotFound, "exchange not found")
}

func (s *Server) handleClearExchanges(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.checkInspectorAccess(w, user) {
		return
	}

	tunnelID := chi.URLParam(r, "id")
	if err := s.checkTunnelAccess(tunnelID, user); err != nil {
		s.respondError(w, http.StatusForbidden, err.Error())
		return
	}

	buf := s.getInspectBuffer(tunnelID)
	if buf != nil {
		buf.Clear()
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (s *Server) handleInspectStream(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.checkInspectorAccess(w, user) {
		return
	}

	tunnelID := chi.URLParam(r, "id")
	if err := s.checkTunnelAccess(tunnelID, user); err != nil {
		s.respondError(w, http.StatusForbidden, err.Error())
		return
	}

	buf := s.getInspectBuffer(tunnelID)
	if buf == nil {
		s.respondError(w, http.StatusNotFound, "inspection not available")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.respondError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	ch := buf.Subscribe()
	defer buf.Unsubscribe(ch)

	_, _ = fmt.Fprintf(w, ": ping\n\n")
	flusher.Flush()

	for {
		select {
		case ex, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(ex.Summary())
			_, _ = fmt.Fprintf(w, "event: exchange\ndata: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleInspectStatus(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.checkInspectorAccess(w, user) {
		return
	}

	tunnelID := chi.URLParam(r, "id")
	if err := s.checkTunnelAccess(tunnelID, user); err != nil {
		s.respondError(w, http.StatusForbidden, err.Error())
		return
	}

	buf := s.getInspectBuffer(tunnelID)
	if buf == nil {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"enabled":     false,
			"bufferSize":  0,
			"subscribers": 0,
		})
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"enabled":     true,
		"bufferSize":  buf.Len(),
		"subscribers": buf.SubscribersCount(),
	})
}

func (s *Server) handleReplayExchange(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.checkInspectorAccess(w, user) {
		return
	}

	tunnelID := chi.URLParam(r, "id")
	if err := s.checkTunnelAccess(tunnelID, user); err != nil {
		s.respondError(w, http.StatusForbidden, err.Error())
		return
	}

	// Parse optional modifications from request body
	var mods dto.ReplayExchangeRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&mods)
	}

	exchangeID := chi.URLParam(r, "exchangeId")

	// Find original exchange from buffer or DB
	var ex *inspect.CapturedExchange
	if buf := s.getInspectBuffer(tunnelID); buf != nil {
		ex = buf.Get(exchangeID)
	}
	if ex == nil && s.inspectProvider != nil {
		ex, _ = s.inspectProvider.GetPersisted(exchangeID)
	}
	if ex == nil {
		s.respondError(w, http.StatusNotFound, "exchange not found")
		return
	}

	if s.replayProvider == nil {
		s.respondError(w, http.StatusServiceUnavailable, "replay not available")
		return
	}

	// Find subdomain for this tunnel from the tunnel provider
	var subdomain string
	if s.tunnelProvider != nil {
		tunnels := s.tunnelProvider.GetTunnelsByUserID(user.ID)
		if user.IsAdmin {
			tunnels = s.tunnelProvider.GetAllTunnels()
		}
		for _, t := range tunnels {
			if t.ID == tunnelID {
				subdomain = t.Subdomain
				break
			}
		}
	}
	if subdomain == "" {
		s.respondError(w, http.StatusNotFound, "tunnel subdomain not found")
		return
	}

	// Apply modifications or use original values
	method := ex.Method
	if mods.Method != nil {
		method = *mods.Method
	}
	path := ex.Path
	if mods.Path != nil {
		path = *mods.Path
	}
	reqHeaders := ex.RequestHeaders.Clone()
	if mods.Headers != nil {
		reqHeaders = http.Header(mods.Headers)
	}
	reqBody := ex.RequestBody
	if mods.Body != nil {
		decoded, err := base64.StdEncoding.DecodeString(*mods.Body)
		if err == nil {
			reqBody = decoded
		}
	}

	// Build replay request
	bodyReader := bytes.NewReader(reqBody)
	replayReq, err := http.NewRequest(method, path, bodyReader)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to create replay request")
		return
	}
	replayReq.Host = ex.Host
	replayReq.Header = reqHeaders

	startTime := time.Now()
	result, err := s.replayProvider.ReplayRequest(subdomain, replayReq)
	if err != nil {
		s.respondError(w, http.StatusBadGateway, fmt.Sprintf("replay failed: %v", err))
		return
	}

	// Build new CapturedExchange from replay
	newEx := &inspect.CapturedExchange{
		ID:               generateReplayID(),
		TunnelID:         tunnelID,
		ReplayRef:        exchangeID,
		Timestamp:        startTime,
		Duration:         time.Since(startTime),
		Method:           method,
		Path:             path,
		Host:             ex.Host,
		RequestHeaders:   reqHeaders,
		RequestBody:      reqBody,
		RequestBodySize:  int64(len(reqBody)),
		RemoteAddr:       "replay",
		StatusCode:       result.StatusCode,
		ResponseHeaders:  result.Headers,
		ResponseBody:     result.Body,
		ResponseBodySize: int64(len(result.Body)),
	}

	// Add to inspect buffer + persist
	if s.inspectProvider != nil {
		s.inspectProvider.AddAndPersist(tunnelID, newEx)
	}

	// Build response headers map
	respHeaders := make(map[string][]string)
	for k, v := range result.Headers {
		respHeaders[k] = v
	}

	s.respondJSON(w, http.StatusOK, dto.ReplayResponse{
		StatusCode:      result.StatusCode,
		ResponseHeaders: respHeaders,
		ResponseBody:    result.Body,
		ExchangeID:      newEx.ID,
	})
}

func (s *Server) checkTunnelAccess(tunnelID string, user *auth.AuthenticatedUser) error {
	if user.IsAdmin {
		return nil
	}
	if s.tunnelProvider == nil {
		return fmt.Errorf("access denied")
	}
	tunnels := s.tunnelProvider.GetTunnelsByUserID(user.ID)
	for _, t := range tunnels {
		if t.ID == tunnelID {
			return nil
		}
	}
	return fmt.Errorf("access denied")
}

func (s *Server) getInspectBuffer(tunnelID string) *inspect.RingBuffer {
	if s.inspectProvider == nil {
		return nil
	}
	return s.inspectProvider.Get(tunnelID)
}

func generateReplayID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "r-" + hex.EncodeToString(b)
}
