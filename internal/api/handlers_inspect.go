package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/inspect"
)

func (s *Server) handleListExchanges(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
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
			"exchanges": []interface{}{},
			"total":     0,
		})
		return
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	exchanges := buf.List(offset, limit)
	summaries := make([]inspect.ExchangeSummary, len(exchanges))
	for i, ex := range exchanges {
		summaries[i] = ex.Summary()
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"exchanges": summaries,
		"total":     buf.Len(),
	})
}

func (s *Server) handleGetExchange(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tunnelID := chi.URLParam(r, "id")
	if err := s.checkTunnelAccess(tunnelID, user); err != nil {
		s.respondError(w, http.StatusForbidden, err.Error())
		return
	}

	exchangeID := chi.URLParam(r, "exchangeId")
	buf := s.getInspectBuffer(tunnelID)
	if buf == nil {
		s.respondError(w, http.StatusNotFound, "exchange not found")
		return
	}

	ex := buf.Get(exchangeID)
	if ex == nil {
		s.respondError(w, http.StatusNotFound, "exchange not found")
		return
	}

	s.respondJSON(w, http.StatusOK, ex)
}

func (s *Server) handleClearExchanges(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
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

	fmt.Fprintf(w, ": ping\n\n")
	flusher.Flush()

	for {
		select {
		case ex, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(ex.Summary())
			fmt.Fprintf(w, "event: exchange\ndata: %s\n\n", data)
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
