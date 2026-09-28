package api

import (
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type ipBanDTO struct {
	IP        string    `json:"ip"`
	Reason    string    `json:"reason"`
	BannedAt  time.Time `json:"banned_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type createIPBanRequest struct {
	IP     string `json:"ip"`
	Reason string `json:"reason"`
	TTL    string `json:"ttl"` // Go duration string, e.g. "72h"; empty → config default
}

// handleListIPBans returns all active IP bans.
func (s *Server) handleListIPBans(w http.ResponseWriter, r *http.Request) {
	if s.ipBanStore == nil {
		s.respondJSON(w, http.StatusOK, []ipBanDTO{})
		return
	}
	entries, err := s.ipBanStore.List()
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list IP bans")
		s.respondError(w, http.StatusInternalServerError, "failed to list bans")
		return
	}
	out := make([]ipBanDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, ipBanDTO{
			IP:        e.IP,
			Reason:    e.Reason,
			BannedAt:  e.BannedAt,
			ExpiresAt: e.ExpiresAt,
		})
	}
	s.respondJSON(w, http.StatusOK, out)
}

// handleCreateIPBan creates a manual IP ban.
func (s *Server) handleCreateIPBan(w http.ResponseWriter, r *http.Request) {
	if s.ipBanStore == nil {
		s.respondError(w, http.StatusServiceUnavailable, "ip ban store unavailable")
		return
	}
	var req createIPBanRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if net.ParseIP(req.IP) == nil {
		s.respondError(w, http.StatusBadRequest, "invalid ip address")
		return
	}
	ttl := s.cfg.Auth.TarpitBanTTL
	if ttl <= 0 {
		ttl = 72 * time.Hour
	}
	if req.TTL != "" {
		parsed, err := time.ParseDuration(req.TTL)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, "invalid ttl format")
			return
		}
		ttl = parsed
	}
	reason := req.Reason
	if reason == "" {
		reason = "manual ban"
	}
	if _, err := s.ipBanStore.Ban(req.IP, reason, ttl); err != nil {
		s.log.Error().Err(err).Str("ip", req.IP).Msg("failed to ban IP")
		s.respondError(w, http.StatusInternalServerError, "failed to ban IP")
		return
	}
	s.respondJSON(w, http.StatusCreated, ipBanDTO{
		IP:        req.IP,
		Reason:    reason,
		BannedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(ttl),
	})
}

// handleDeleteIPBan removes an active IP ban.
func (s *Server) handleDeleteIPBan(w http.ResponseWriter, r *http.Request) {
	if s.ipBanStore == nil {
		s.respondError(w, http.StatusServiceUnavailable, "ip ban store unavailable")
		return
	}
	ip := chi.URLParam(r, "ip")
	if net.ParseIP(ip) == nil {
		s.respondError(w, http.StatusBadRequest, "invalid ip address")
		return
	}
	if err := s.ipBanStore.Unban(ip); err != nil {
		s.log.Error().Err(err).Str("ip", ip).Msg("failed to unban IP")
		s.respondError(w, http.StatusInternalServerError, "failed to unban IP")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
