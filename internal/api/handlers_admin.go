package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/database"
)

// handleGetStats returns server statistics
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	var stats Stats
	if s.tunnelProvider != nil {
		stats = s.tunnelProvider.GetStats()
	}

	totalUsers, _ := s.db.Users.Count()

	s.respondJSON(w, http.StatusOK, dto.StatsResponse{
		ActiveClients: stats.ActiveClients,
		ActiveTunnels: stats.ActiveTunnels,
		HTTPTunnels:   stats.HTTPTunnels,
		TCPTunnels:    stats.TCPTunnels,
		UDPTunnels:    stats.UDPTunnels,
		TotalUsers:    totalUsers,
	})
}

// handleListUsers returns a list of all users
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	users, total, err := s.db.Users.List(limit, offset)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to list users")
		s.respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	userDTOs := make([]*dto.UserDTO, len(users))
	for i, u := range users {
		userDTOs[i] = dto.UserFromModel(u)
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": userDTOs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// handleListInviteCodes returns a list of invite codes
func (s *Server) handleListInviteCodes(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Check if only unused codes are requested
	unusedOnly := r.URL.Query().Get("unused") == "true"

	var codes []*database.InviteCode
	var total int
	var err error

	if unusedOnly {
		codes, total, err = s.db.Invites.ListUnused(limit, offset)
	} else {
		codes, total, err = s.db.Invites.List(limit, offset)
	}

	if err != nil {
		s.log.Error().Err(err).Msg("Failed to list invite codes")
		s.respondError(w, http.StatusInternalServerError, "failed to list invite codes")
		return
	}

	codeDTOs := make([]*dto.InviteCodeDTO, len(codes))
	for i, c := range codes {
		codeDTOs[i] = dto.InviteCodeFromModel(c)
	}

	s.respondJSON(w, http.StatusOK, dto.InviteCodesListResponse{
		Codes: codeDTOs,
		Total: total,
	})
}

// handleCreateInviteCode creates a new invite code
func (s *Server) handleCreateInviteCode(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.CreateInviteCodeRequest
	if err := s.decodeJSON(r, &req); err != nil {
		// Allow empty body
		req = dto.CreateInviteCodeRequest{}
	}

	// Generate invite code
	code, err := auth.GenerateInviteCode()
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to generate invite code")
		s.respondError(w, http.StatusInternalServerError, "failed to generate invite code")
		return
	}

	invite := &database.InviteCode{
		Code:            code,
		CreatedByUserID: &user.ID,
	}

	// Set expiration if requested
	if req.ExpiresInDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, req.ExpiresInDays)
		invite.ExpiresAt = &expiresAt
	}

	if err := s.db.Invites.Create(invite); err != nil {
		s.log.Error().Err(err).Msg("Failed to create invite code")
		s.respondError(w, http.StatusInternalServerError, "failed to create invite code")
		return
	}

	// Log audit
	ipAddress := auth.GetClientIP(r)
	s.db.Audit.Log(&user.ID, database.ActionInviteCreated, map[string]interface{}{
		"invite_id": invite.ID,
	}, ipAddress)

	s.respondJSON(w, http.StatusCreated, dto.InviteCodeFromModel(invite))
}

// handleDeleteInviteCode deletes an invite code
func (s *Server) handleDeleteInviteCode(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid invite code id")
		return
	}

	if err := s.db.Invites.Delete(id); err != nil {
		if errors.Is(err, database.ErrInviteNotFound) {
			s.respondError(w, http.StatusNotFound, "invite code not found")
			return
		}
		s.log.Error().Err(err).Msg("Failed to delete invite code")
		s.respondError(w, http.StatusInternalServerError, "failed to delete invite code")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "invite code deleted successfully",
	})
}
