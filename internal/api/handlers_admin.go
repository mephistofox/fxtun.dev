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

// handleListAuditLogs returns a list of audit logs
func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
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

	// Parse optional user_id filter
	userIDStr := r.URL.Query().Get("user_id")

	var logs []*database.AuditLog
	var total int
	var err error

	if userIDStr != "" {
		userID, parseErr := strconv.ParseInt(userIDStr, 10, 64)
		if parseErr != nil {
			s.respondError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		logs, total, err = s.db.Audit.GetByUserID(userID, limit, offset)
	} else {
		logs, total, err = s.db.Audit.List(limit, offset)
	}

	if err != nil {
		s.log.Error().Err(err).Msg("Failed to list audit logs")
		s.respondError(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	// Batch fetch users for logs
	userIDs := make([]int64, 0)
	for _, log := range logs {
		if log.UserID != nil {
			userIDs = append(userIDs, *log.UserID)
		}
	}
	usersMap, _ := s.db.Users.GetByIDs(userIDs)

	logDTOs := make([]*dto.AuditLogDTO, len(logs))
	for i, log := range logs {
		var userPhone string
		if log.UserID != nil {
			if user, ok := usersMap[*log.UserID]; ok {
				userPhone = user.Phone
			}
		}
		logDTOs[i] = dto.AuditLogFromModel(log, userPhone)
	}

	s.respondJSON(w, http.StatusOK, dto.AuditLogsListResponse{
		Logs:  logDTOs,
		Total: total,
	})
}

// handleListAllTunnels returns all active tunnels for admin
func (s *Server) handleListAllTunnels(w http.ResponseWriter, r *http.Request) {
	if s.tunnelProvider == nil {
		s.respondJSON(w, http.StatusOK, dto.AdminTunnelsListResponse{
			Tunnels: []*dto.AdminTunnelDTO{},
			Total:   0,
		})
		return
	}

	// Get all tunnels (userID 0 means all users)
	tunnels := s.tunnelProvider.GetAllTunnels()

	// Batch fetch users for tunnels
	userIDs := make([]int64, 0)
	for _, t := range tunnels {
		if t.UserID > 0 {
			userIDs = append(userIDs, t.UserID)
		}
	}
	usersMap, _ := s.db.Users.GetByIDs(userIDs)

	tunnelDTOs := make([]*dto.AdminTunnelDTO, len(tunnels))
	for i, t := range tunnels {
		var userPhone string
		if t.UserID > 0 {
			if user, ok := usersMap[t.UserID]; ok {
				userPhone = user.Phone
			}
		}

		url := ""
		if t.Type == "http" && t.Subdomain != "" {
			url = "https://" + t.Subdomain + "." + s.baseDomain
		} else if t.RemotePort > 0 {
			url = t.Type + "://" + s.baseDomain + ":" + strconv.Itoa(t.RemotePort)
		}

		tunnelDTOs[i] = &dto.AdminTunnelDTO{
			ID:         t.ID,
			Type:       t.Type,
			Name:       t.Name,
			Subdomain:  t.Subdomain,
			RemotePort: t.RemotePort,
			LocalPort:  t.LocalPort,
			URL:        url,
			ClientID:   t.ClientID,
			UserID:     t.UserID,
			UserPhone:  userPhone,
			CreatedAt:  t.CreatedAt,
		}
	}

	s.respondJSON(w, http.StatusOK, dto.AdminTunnelsListResponse{
		Tunnels: tunnelDTOs,
		Total:   len(tunnelDTOs),
	})
}

// handleUpdateUser updates a user's admin status or active status
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	// Prevent self-modification of admin status
	if id == currentUser.ID {
		s.respondError(w, http.StatusForbidden, "cannot modify your own account")
		return
	}

	var req dto.UpdateUserRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := s.db.Users.GetByID(id)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			s.respondError(w, http.StatusNotFound, "user not found")
			return
		}
		s.log.Error().Err(err).Msg("Failed to get user")
		s.respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	// Update fields
	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := s.db.Users.Update(user); err != nil {
		s.log.Error().Err(err).Msg("Failed to update user")
		s.respondError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	// Log audit
	ipAddress := auth.GetClientIP(r)
	details := map[string]interface{}{
		"target_user_id": id,
	}
	if req.IsAdmin != nil {
		details["is_admin"] = *req.IsAdmin
	}
	if req.IsActive != nil {
		details["is_active"] = *req.IsActive
	}
	s.db.Audit.Log(&currentUser.ID, database.ActionUserUpdated, details, ipAddress)

	s.respondJSON(w, http.StatusOK, dto.UserFromModel(user))
}

// handleDeleteUser deletes a user
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	// Prevent self-deletion
	if id == currentUser.ID {
		s.respondError(w, http.StatusForbidden, "cannot delete your own account")
		return
	}

	// Get user info before deletion for audit
	user, err := s.db.Users.GetByID(id)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			s.respondError(w, http.StatusNotFound, "user not found")
			return
		}
		s.log.Error().Err(err).Msg("Failed to get user")
		s.respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if err := s.db.Users.Delete(id); err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			s.respondError(w, http.StatusNotFound, "user not found")
			return
		}
		s.log.Error().Err(err).Msg("Failed to delete user")
		s.respondError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	// Log audit
	ipAddress := auth.GetClientIP(r)
	s.db.Audit.Log(&currentUser.ID, database.ActionUserDeleted, map[string]interface{}{
		"deleted_user_id":    id,
		"deleted_user_phone": user.Phone,
	}, ipAddress)

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "user deleted successfully",
	})
}

// handleAdminCloseTunnel closes any tunnel (admin only)
func (s *Server) handleAdminCloseTunnel(w http.ResponseWriter, r *http.Request) {
	tunnelID := chi.URLParam(r, "id")
	if tunnelID == "" {
		s.respondError(w, http.StatusBadRequest, "tunnel id required")
		return
	}

	if s.tunnelProvider == nil {
		s.respondError(w, http.StatusServiceUnavailable, "tunnel provider not available")
		return
	}

	// Admin can close any tunnel (userID 0 bypasses user check)
	if err := s.tunnelProvider.AdminCloseTunnel(tunnelID); err != nil {
		s.respondError(w, http.StatusNotFound, "tunnel not found")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "tunnel closed",
	})
}
