package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtun.dev/internal/api/dto"
	"github.com/mephistofox/fxtun.dev/internal/auth"
	"github.com/mephistofox/fxtun.dev/internal/database"
)

// handleListTunnels returns the user's active tunnels
func (s *Server) handleListTunnels(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if s.tunnelProvider == nil {
		s.respondJSON(w, http.StatusOK, dto.TunnelsListResponse{
			Tunnels: []*dto.TunnelDTO{},
			Total:   0,
		})
		return
	}

	tunnels := s.tunnelProvider.GetTunnelsByUserID(user.ID)

	tunnelDTOs := make([]*dto.TunnelDTO, len(tunnels))
	for i, t := range tunnels {
		tunnelDTO := &dto.TunnelDTO{
			ID:         t.ID,
			Type:       t.Type,
			Name:       t.Name,
			Subdomain:  t.Subdomain,
			RemotePort: t.RemotePort,
			LocalPort:  t.LocalPort,
			ClientID:   t.ClientID,
			CreatedAt:  t.CreatedAt,
		}

		// Generate URL for HTTP tunnels
		if t.Type == "http" && t.Subdomain != "" {
			tunnelDTO.URL = "https://" + t.Subdomain + "." + s.baseDomain
		}

		tunnelDTOs[i] = tunnelDTO
	}

	s.respondJSON(w, http.StatusOK, dto.TunnelsListResponse{
		Tunnels: tunnelDTOs,
		Total:   len(tunnelDTOs),
	})
}

// handleCloseTunnel closes a tunnel
func (s *Server) handleCloseTunnel(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tunnelID := chi.URLParam(r, "id")
	if tunnelID == "" {
		s.respondError(w, http.StatusBadRequest, "tunnel id is required")
		return
	}

	if s.tunnelProvider == nil {
		s.respondError(w, http.StatusNotFound, "tunnel not found")
		return
	}

	if err := s.tunnelProvider.CloseTunnelByID(tunnelID, user.ID); err != nil {
		s.log.Error().Err(err).Str("tunnel_id", tunnelID).Msg("Failed to close tunnel")
		s.respondError(w, http.StatusNotFound, "tunnel not found or access denied")
		return
	}

	// Log audit
	ipAddress := auth.GetClientIP(r)
	_ = s.db.Audit.Log(&user.ID, database.ActionTunnelClosed, map[string]interface{}{
		"tunnel_id": tunnelID,
	}, ipAddress)

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "tunnel closed successfully",
	})
}
