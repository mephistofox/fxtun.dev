package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtunnel/internal/server/api/dto"
	"github.com/mephistofox/fxtunnel/internal/server/auth"
	"github.com/mephistofox/fxtunnel/internal/server/database"
)

// handleGetStats returns server statistics
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	var stats Stats
	if s.tunnelProvider != nil {
		stats = s.tunnelProvider.GetStats()
	}

	totalUsers, err := s.db.Users.Count()
	if err != nil {
		s.log.Error().Err(err).Msg("failed to count users for admin stats")
	}

	s.respondJSON(w, http.StatusOK, dto.StatsResponse{
		ActiveClients: stats.ActiveClients,
		ActiveTunnels: stats.ActiveTunnels,
		HTTPTunnels:   stats.HTTPTunnels,
		TCPTunnels:    stats.TCPTunnels,
		UDPTunnels:    stats.UDPTunnels,
		TotalUsers:    totalUsers,
	})
}

// handleListUsers returns a list of all users with server-side filtering, search, and stats
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	filter := r.URL.Query().Get("filter")
	if filter != "active" && filter != "blocked" && filter != "admins" {
		filter = "all"
	}
	search := r.URL.Query().Get("search")

	// Sort params (Task 6)
	sortBy := r.URL.Query().Get("sort_by")
	order := r.URL.Query().Get("order")

	users, total, err := s.db.Users.List(database.UserListParams{
		Filter: filter,
		Search: search,
		Limit:  limit,
		Offset: offset,
		SortBy: sortBy,
		Order:  order,
	})
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to list users")
		s.respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	stats, _ := s.db.Users.Stats(search)

	userDTOs := make([]*dto.UserDTO, len(users))
	for i, u := range users {
		userDTOs[i] = dto.UserFromModel(u)
	}

	plans, _ := s.db.Plans.List()
	planMap := make(map[int64]*database.Plan)
	for _, p := range plans {
		planMap[p.ID] = p
	}
	for i, u := range users {
		if p, ok := planMap[u.PlanID]; ok {
			userDTOs[i].Plan = dto.PlanFromModel(p)
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": userDTOs,
		"total": total,
		"page":  page,
		"limit": limit,
		"stats": stats,
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

// handleListAllTunnels returns all active tunnels for admin with optional type and user_id filters
func (s *Server) handleListAllTunnels(w http.ResponseWriter, r *http.Request) {
	if s.tunnelProvider == nil {
		s.respondJSON(w, http.StatusOK, dto.AdminTunnelsListResponse{
			Tunnels: []*dto.AdminTunnelDTO{},
			Total:   0,
		})
		return
	}

	// Get all tunnels
	tunnels := s.tunnelProvider.GetAllTunnels()

	// Task 6: Filter in-memory by type and user_id
	tunnelType := r.URL.Query().Get("type")
	userIDStr := r.URL.Query().Get("user_id")
	var filterUserID int64
	if userIDStr != "" {
		filterUserID, _ = strconv.ParseInt(userIDStr, 10, 64)
	}

	var filtered []TunnelInfo
	for _, t := range tunnels {
		if tunnelType != "" && t.Type != tunnelType {
			continue
		}
		if userIDStr != "" && t.UserID != filterUserID {
			continue
		}
		filtered = append(filtered, t)
	}

	// Batch fetch users for tunnels
	userIDs := make([]int64, 0)
	for _, t := range filtered {
		if t.UserID > 0 {
			userIDs = append(userIDs, t.UserID)
		}
	}
	usersMap, _ := s.db.Users.GetByIDs(userIDs)

	tunnelDTOs := make([]*dto.AdminTunnelDTO, len(filtered))
	for i, t := range filtered {
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
	if req.PlanID != nil {
		user.PlanID = *req.PlanID
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
	if req.PlanID != nil {
		details["plan_id"] = *req.PlanID
	}
	_ = s.db.Audit.Log(&currentUser.ID, database.ActionUserUpdated, details, ipAddress)

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
	_ = s.db.Audit.Log(&currentUser.ID, database.ActionUserDeleted, map[string]interface{}{
		"deleted_user_id":    id,
		"deleted_user_email": user.Email,
	}, ipAddress)

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "user deleted successfully",
	})
}

// handleMergeUsers merges two users (admin only)
func (s *Server) handleMergeUsers(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.MergeUsersRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.PrimaryUserID == 0 || req.SecondaryUserID == 0 {
		s.respondError(w, http.StatusBadRequest, "both primary_user_id and secondary_user_id are required")
		return
	}

	if req.PrimaryUserID == req.SecondaryUserID {
		s.respondError(w, http.StatusBadRequest, "cannot merge a user with itself")
		return
	}

	if req.PrimaryUserID == currentUser.ID || req.SecondaryUserID == currentUser.ID {
		s.respondError(w, http.StatusForbidden, "cannot merge your own account")
		return
	}

	// Verify both users exist
	primaryUser, err := s.db.Users.GetByID(req.PrimaryUserID)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			s.respondError(w, http.StatusNotFound, "primary user not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, "failed to get primary user")
		return
	}

	secondaryUser, err := s.db.Users.GetByID(req.SecondaryUserID)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			s.respondError(w, http.StatusNotFound, "secondary user not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, "failed to get secondary user")
		return
	}

	if err := s.db.Users.MergeUsers(req.PrimaryUserID, req.SecondaryUserID); err != nil {
		s.log.Error().Err(err).Int64("primary", req.PrimaryUserID).Int64("secondary", req.SecondaryUserID).Msg("Failed to merge users")
		s.respondError(w, http.StatusInternalServerError, "failed to merge users")
		return
	}

	ipAddress := auth.GetClientIP(r)
	_ = s.db.Audit.Log(&currentUser.ID, database.ActionUsersMerged, map[string]interface{}{
		"primary_user_id":    req.PrimaryUserID,
		"primary_email":      primaryUser.Email,
		"secondary_user_id":  req.SecondaryUserID,
		"secondary_email":    secondaryUser.Email,
	}, ipAddress)

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "users merged successfully",
	})
}

// handleAdminResetPassword resets a user's password (admin only)
func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
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

	var req dto.ResetPasswordRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.NewPassword) < 8 || len(req.NewPassword) > 72 {
		s.respondError(w, http.StatusBadRequest, "password must be between 8 and 72 characters")
		return
	}

	// Verify user exists
	_, err = s.db.Users.GetByID(id)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			s.respondError(w, http.StatusNotFound, "user not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := s.db.Users.UpdatePassword(id, hash); err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}

	// Invalidate all existing sessions for the user
	_ = s.db.Sessions.DeleteByUserID(id)

	ipAddress := auth.GetClientIP(r)
	_ = s.db.Audit.Log(&currentUser.ID, database.ActionPasswordReset, map[string]interface{}{
		"target_user_id": id,
	}, ipAddress)

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "password reset successfully",
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

// handleListPlans returns all plans
func (s *Server) handleListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := s.db.Plans.List()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to list plans")
		return
	}
	planDTOs := make([]*dto.PlanDTO, len(plans))
	for i, p := range plans {
		planDTOs[i] = dto.PlanFromModel(p)
	}
	s.respondJSON(w, http.StatusOK, map[string]interface{}{"plans": planDTOs, "total": len(planDTOs)})
}

// handleListPublicPlans returns plans visible on landing page (public, no auth required)
func (s *Server) handleListPublicPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := s.db.Plans.ListPublic()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to list plans")
		return
	}
	planDTOs := make([]*dto.PlanDTO, len(plans))
	for i, p := range plans {
		planDTOs[i] = dto.PlanFromModel(p)
	}
	s.respondJSON(w, http.StatusOK, map[string]interface{}{"plans": planDTOs})
}

// handleCreatePlan creates a new plan
func (s *Server) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePlanRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Slug == "" || req.Name == "" {
		s.respondError(w, http.StatusBadRequest, "slug and name are required")
		return
	}
	plan := &database.Plan{
		Slug: req.Slug, Name: req.Name, Price: req.Price,
		MaxTunnels: req.MaxTunnels, MaxDomains: req.MaxDomains,
		MaxCustomDomains: req.MaxCustomDomains, MaxTokens: req.MaxTokens,
		MaxTunnelsPerToken: req.MaxTunnelsPerToken, BandwidthMbps: req.BandwidthMbps,
		InspectorEnabled: req.InspectorEnabled,
		IsPublic: req.IsPublic, IsRecommended: req.IsRecommended,
		RateLimitTCP: req.RateLimitTCP, RateLimitUDP: req.RateLimitUDP, RateLimitHTTP: req.RateLimitHTTP,
		CreemProductID: req.CreemProductID, MaxDataSessions: req.MaxDataSessions,
	}
	if err := s.db.Plans.Create(plan); err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to create plan")
		return
	}
	s.respondJSON(w, http.StatusCreated, dto.PlanFromModel(plan))
}

// handleUpdatePlan updates a plan
func (s *Server) handleUpdatePlan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid plan id")
		return
	}
	plan, err := s.db.Plans.GetByID(id)
	if err != nil {
		if errors.Is(err, database.ErrPlanNotFound) {
			s.respondError(w, http.StatusNotFound, "plan not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, "failed to get plan")
		return
	}
	var req dto.UpdatePlanRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.Price != nil {
		plan.Price = *req.Price
	}
	if req.MaxTunnels != nil {
		plan.MaxTunnels = *req.MaxTunnels
	}
	if req.MaxDomains != nil {
		plan.MaxDomains = *req.MaxDomains
	}
	if req.MaxCustomDomains != nil {
		plan.MaxCustomDomains = *req.MaxCustomDomains
	}
	if req.MaxTokens != nil {
		plan.MaxTokens = *req.MaxTokens
	}
	if req.MaxTunnelsPerToken != nil {
		plan.MaxTunnelsPerToken = *req.MaxTunnelsPerToken
	}
	if req.BandwidthMbps != nil {
		plan.BandwidthMbps = *req.BandwidthMbps
	}
	if req.InspectorEnabled != nil {
		plan.InspectorEnabled = *req.InspectorEnabled
	}
	if req.IsPublic != nil {
		plan.IsPublic = *req.IsPublic
	}
	if req.IsRecommended != nil {
		plan.IsRecommended = *req.IsRecommended
	}
	if req.RateLimitTCP != nil {
		plan.RateLimitTCP = *req.RateLimitTCP
	}
	if req.RateLimitUDP != nil {
		plan.RateLimitUDP = *req.RateLimitUDP
	}
	if req.RateLimitHTTP != nil {
		plan.RateLimitHTTP = *req.RateLimitHTTP
	}
	if req.CreemProductID != nil {
		plan.CreemProductID = *req.CreemProductID
	}
	if req.MaxDataSessions != nil {
		plan.MaxDataSessions = *req.MaxDataSessions
	}
	if err := s.db.Plans.Update(plan); err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to update plan")
		return
	}
	s.respondJSON(w, http.StatusOK, dto.PlanFromModel(plan))
}

// handleDeletePlan deletes a plan
func (s *Server) handleDeletePlan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid plan id")
		return
	}
	if err := s.db.Plans.Delete(id); err != nil {
		if errors.Is(err, database.ErrPlanNotFound) {
			s.respondError(w, http.StatusNotFound, "plan not found")
			return
		}
		if errors.Is(err, database.ErrPlanHasUsers) {
			s.respondError(w, http.StatusConflict, "cannot delete plan with active users")
			return
		}
		s.respondError(w, http.StatusInternalServerError, "failed to delete plan")
		return
	}
	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{Success: true, Message: "plan deleted"})
}

// handleAdminListSubscriptions returns all subscriptions for admin
func (s *Server) handleAdminListSubscriptions(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	subs, total, err := s.db.Subscriptions.ListAll(limit, offset)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to list subscriptions")
		s.respondError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}

	// Batch fetch users and plans
	userIDs := make([]int64, 0)
	planIDs := make(map[int64]bool)
	for _, sub := range subs {
		userIDs = append(userIDs, sub.UserID)
		planIDs[sub.PlanID] = true
		if sub.NextPlanID != nil {
			planIDs[*sub.NextPlanID] = true
		}
	}
	usersMap, _ := s.db.Users.GetByIDs(userIDs)
	plans, _ := s.db.Plans.List()
	planMap := make(map[int64]*database.Plan)
	for _, p := range plans {
		planMap[p.ID] = p
	}

	subDTOs := make([]*dto.AdminSubscriptionDTO, len(subs))
	for i, sub := range subs {
		var userPhone, userEmail string
		if user, ok := usersMap[sub.UserID]; ok {
			userPhone = user.Phone
			userEmail = user.Email
		}

		subDTO := &dto.AdminSubscriptionDTO{
			ID:                 sub.ID,
			UserID:             sub.UserID,
			UserPhone:          userPhone,
			UserEmail:          userEmail,
			PlanID:             sub.PlanID,
			Status:             string(sub.Status),
			Recurring:          sub.Recurring,
			CurrentPeriodStart: sub.CurrentPeriodStart,
			CurrentPeriodEnd:   sub.CurrentPeriodEnd,
			CreatedAt:          sub.CreatedAt,
		}

		if plan, ok := planMap[sub.PlanID]; ok {
			subDTO.Plan = dto.PlanFromModel(plan)
		}
		if sub.NextPlanID != nil {
			if plan, ok := planMap[*sub.NextPlanID]; ok {
				subDTO.NextPlan = dto.PlanFromModel(plan)
			}
		}

		subDTOs[i] = subDTO
	}

	s.respondJSON(w, http.StatusOK, dto.AdminSubscriptionsListResponse{
		Subscriptions: subDTOs,
		Total:         total,
		Page:          page,
		Limit:         limit,
	})
}

// handleAdminListPayments returns all payments for admin
func (s *Server) handleAdminListPayments(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	payments, total, err := s.db.Payments.ListAll(limit, offset)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to list payments")
		s.respondError(w, http.StatusInternalServerError, "failed to list payments")
		return
	}

	// Batch fetch users
	userIDs := make([]int64, 0)
	for _, p := range payments {
		userIDs = append(userIDs, p.UserID)
	}
	usersMap, _ := s.db.Users.GetByIDs(userIDs)

	paymentDTOs := make([]*dto.AdminPaymentDTO, len(payments))
	for i, p := range payments {
		var userPhone, userEmail string
		if user, ok := usersMap[p.UserID]; ok {
			userPhone = user.Phone
			userEmail = user.Email
		}

		paymentDTOs[i] = &dto.AdminPaymentDTO{
			ID:             p.ID,
			UserID:         p.UserID,
			UserPhone:      userPhone,
			UserEmail:      userEmail,
			SubscriptionID: p.SubscriptionID,
			InvoiceID:      p.InvoiceID,
			Amount:         p.Amount,
			Status:         string(p.Status),
			IsRecurring:    p.IsRecurring,
			CreatedAt:      p.CreatedAt,
		}
	}

	s.respondJSON(w, http.StatusOK, dto.AdminPaymentsListResponse{
		Payments: paymentDTOs,
		Total:    total,
		Page:     page,
		Limit:    limit,
	})
}

// handleAdminCancelSubscription cancels a user's subscription (admin)
func (s *Server) handleAdminCancelSubscription(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid subscription id")
		return
	}

	sub, err := s.db.Subscriptions.GetByID(id)
	if err != nil || sub == nil {
		s.respondError(w, http.StatusNotFound, "subscription not found")
		return
	}

	sub.Status = database.SubscriptionStatusCancelled
	sub.Recurring = false
	if err := s.db.Subscriptions.Update(sub); err != nil {
		s.log.Error().Err(err).Msg("Failed to cancel subscription")
		s.respondError(w, http.StatusInternalServerError, "failed to cancel subscription")
		return
	}

	// Update user's plan to Free immediately
	freePlan, err := s.db.Plans.GetDefault()
	if err == nil && freePlan != nil {
		if user, err := s.db.Users.GetByID(sub.UserID); err == nil && user != nil {
			user.PlanID = freePlan.ID
			if err := s.db.Users.Update(user); err != nil {
				s.log.Error().Err(err).Int64("user_id", user.ID).Msg("Failed to update user plan to free")
			}
		}
	}

	// Log audit
	_ = s.db.Audit.Log(&currentUser.ID, "admin_subscription_cancelled", map[string]interface{}{
		"subscription_id": sub.ID,
		"user_id":         sub.UserID,
	}, auth.GetClientIP(r))

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "subscription cancelled",
	})
}

// handleAdminExtendSubscription extends a subscription period
func (s *Server) handleAdminExtendSubscription(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid subscription id")
		return
	}

	var req dto.ExtendSubscriptionRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Days <= 0 {
		s.respondError(w, http.StatusBadRequest, "days must be positive")
		return
	}

	sub, err := s.db.Subscriptions.GetByID(id)
	if err != nil || sub == nil {
		s.respondError(w, http.StatusNotFound, "subscription not found")
		return
	}

	// Extend period
	now := time.Now()
	if sub.CurrentPeriodEnd == nil || sub.CurrentPeriodEnd.Before(now) {
		sub.CurrentPeriodStart = &now
		newEnd := now.AddDate(0, 0, req.Days)
		sub.CurrentPeriodEnd = &newEnd
	} else {
		newEnd := sub.CurrentPeriodEnd.AddDate(0, 0, req.Days)
		sub.CurrentPeriodEnd = &newEnd
	}

	// Reactivate if expired or cancelled
	if sub.Status == database.SubscriptionStatusExpired || sub.Status == database.SubscriptionStatusCancelled {
		sub.Status = database.SubscriptionStatusActive
	}

	if err := s.db.Subscriptions.Update(sub); err != nil {
		s.log.Error().Err(err).Msg("Failed to extend subscription")
		s.respondError(w, http.StatusInternalServerError, "failed to extend subscription")
		return
	}

	// Update user plan
	if user, err := s.db.Users.GetByID(sub.UserID); err == nil && user != nil {
		user.PlanID = sub.PlanID
		_ = s.db.Users.Update(user)
	}

	// Log audit
	_ = s.db.Audit.Log(&currentUser.ID, "admin_subscription_extended", map[string]interface{}{
		"subscription_id": sub.ID,
		"user_id":         sub.UserID,
		"days":            req.Days,
		"new_end":         sub.CurrentPeriodEnd,
	}, auth.GetClientIP(r))

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "subscription extended",
	})
}

// handleAdminGrantSubscription grants a free subscription to a user
func (s *Server) handleAdminGrantSubscription(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req dto.GrantSubscriptionRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Months <= 0 || req.Months > 60 {
		s.respondError(w, http.StatusBadRequest, "months must be between 1 and 60")
		return
	}

	// Check user exists
	user, err := s.db.Users.GetByID(userID)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			s.respondError(w, http.StatusNotFound, "user not found")
			return
		}
		s.log.Error().Err(err).Msg("Failed to get user")
		s.respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	// Check plan exists
	plan, err := s.db.Plans.GetByID(req.PlanID)
	if err != nil {
		if errors.Is(err, database.ErrPlanNotFound) {
			s.respondError(w, http.StatusNotFound, "plan not found")
			return
		}
		s.log.Error().Err(err).Msg("Failed to get plan")
		s.respondError(w, http.StatusInternalServerError, "failed to get plan")
		return
	}

	now := time.Now()
	periodEnd := now.AddDate(0, req.Months, 0)

	// Look for existing active subscription
	sub, err := s.db.Subscriptions.GetByUserID(userID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get subscription")
		s.respondError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}

	if sub != nil && sub.IsActive() {
		// Extend existing active subscription
		if sub.CurrentPeriodEnd != nil && sub.CurrentPeriodEnd.After(now) {
			periodEnd = sub.CurrentPeriodEnd.AddDate(0, req.Months, 0)
		}
		sub.PlanID = plan.ID
		sub.CurrentPeriodEnd = &periodEnd
		if err := s.db.Subscriptions.Update(sub); err != nil {
			s.log.Error().Err(err).Msg("Failed to update subscription")
			s.respondError(w, http.StatusInternalServerError, "failed to update subscription")
			return
		}
	} else {
		// Create new subscription
		sub = &database.Subscription{
			UserID:             userID,
			PlanID:             plan.ID,
			Status:             database.SubscriptionStatusActive,
			Recurring:          false,
			CurrentPeriodStart: &now,
			CurrentPeriodEnd:   &periodEnd,
		}
		if err := s.db.Subscriptions.Create(sub); err != nil {
			s.log.Error().Err(err).Msg("Failed to create subscription")
			s.respondError(w, http.StatusInternalServerError, "failed to create subscription")
			return
		}
	}

	// Update user plan
	user.PlanID = plan.ID
	if err := s.db.Users.Update(user); err != nil {
		s.log.Error().Err(err).Msg("Failed to update user plan")
		// Non-fatal: subscription was already created/updated
	}

	// Audit log
	_ = s.db.Audit.Log(&currentUser.ID, "admin_grant_subscription", map[string]interface{}{
		"target_user_id":  userID,
		"plan_id":         plan.ID,
		"plan_name":       plan.Name,
		"months":          req.Months,
		"subscription_id": sub.ID,
		"period_end":      sub.CurrentPeriodEnd,
	}, auth.GetClientIP(r))

	s.respondJSON(w, http.StatusOK, sub)
}

// handleGetUserDetail returns detailed user info with payments, subscriptions, and tunnel history
func (s *Server) handleGetUserDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid user id")
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

	userDTO := dto.UserFromModel(user)
	if plan, err := s.db.Plans.GetByID(user.PlanID); err == nil {
		userDTO.Plan = dto.PlanFromModel(plan)
	}

	// Payments (last 50)
	payments, _, _ := s.db.Payments.GetByUserID(id, 50, 0)
	paymentDTOs := make([]*dto.PaymentDTO, 0, len(payments))
	for _, p := range payments {
		paymentDTOs = append(paymentDTOs, dto.PaymentFromModel(p))
	}

	// Subscriptions (all)
	subs, _ := s.db.Subscriptions.ListByUserID(id)
	plans, _ := s.db.Plans.List()
	planMap := make(map[int64]*database.Plan)
	for _, p := range plans {
		planMap[p.ID] = p
	}
	subDTOs := make([]*dto.AdminSubscriptionDTO, 0, len(subs))
	for _, sub := range subs {
		subDTO := &dto.AdminSubscriptionDTO{
			ID:                 sub.ID,
			UserID:             sub.UserID,
			PlanID:             sub.PlanID,
			Status:             string(sub.Status),
			Recurring:          sub.Recurring,
			CurrentPeriodStart: sub.CurrentPeriodStart,
			CurrentPeriodEnd:   sub.CurrentPeriodEnd,
			CreatedAt:          sub.CreatedAt,
		}
		if plan, ok := planMap[sub.PlanID]; ok {
			subDTO.Plan = dto.PlanFromModel(plan)
		}
		subDTOs = append(subDTOs, subDTO)
	}

	// Tunnel history (last 50)
	history, _ := s.db.UserHistory.GetByUserID(id, 50, 0)
	historyDTOs := make([]*dto.TunnelHistoryDTO, 0, len(history))
	for _, h := range history {
		historyDTOs = append(historyDTOs, &dto.TunnelHistoryDTO{
			ID:             h.ID,
			BundleName:     h.BundleName,
			TunnelType:     h.TunnelType,
			LocalPort:      h.LocalPort,
			RemoteAddr:     h.RemoteAddr,
			URL:            h.URL,
			ConnectedAt:    h.ConnectedAt,
			DisconnectedAt: h.DisconnectedAt,
			BytesSent:      h.BytesSent,
			BytesReceived:  h.BytesReceived,
		})
	}

	// Tunnel stats
	var tunnelStats *dto.TunnelHistoryStatsDTO
	if stats, err := s.db.UserHistory.GetStats(id); err == nil {
		tunnelStats = &dto.TunnelHistoryStatsDTO{
			TotalConnections:   stats.TotalConnections,
			TotalBytesSent:     stats.TotalBytesSent,
			TotalBytesReceived: stats.TotalBytesReceived,
		}
	}

	// Counts
	tokenCount := 0
	if tokens, err := s.db.Tokens.GetByUserID(id); err == nil {
		tokenCount = len(tokens)
	}
	domainCount := 0
	if domains, err := s.db.Domains.GetByUserID(id); err == nil {
		domainCount = len(domains)
	}

	s.respondJSON(w, http.StatusOK, dto.AdminUserDetailResponse{
		User:          userDTO,
		Payments:      paymentDTOs,
		Subscriptions: subDTOs,
		TunnelHistory: historyDTOs,
		TunnelStats:   tunnelStats,
		TokenCount:    tokenCount,
		DomainCount:   domainCount,
	})
}

// ==================== Task 1: Chart data endpoint ====================

// handleGetChartData returns time-series data for admin dashboard charts
func (s *Server) handleGetChartData(w http.ResponseWriter, r *http.Request) {
	metric := r.URL.Query().Get("metric")
	period := r.URL.Query().Get("period")

	var days int
	switch period {
	case "7d":
		days = 7
	case "30d":
		days = 30
	default:
		days = 7
		period = "7d"
	}

	var points []dto.ChartDataPoint

	switch metric {
	case "registrations":
		data, err := s.db.Users.RegistrationsByDay(days)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to get registration chart data")
			s.respondError(w, http.StatusInternalServerError, "failed to get registration data")
			return
		}
		for _, d := range data {
			points = append(points, dto.ChartDataPoint{Date: d.Date.Format("2006-01-02"), Value: d.Value})
		}
	case "payments":
		data, err := s.db.Payments.PaymentsByDay(days)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to get payment chart data")
			s.respondError(w, http.StatusInternalServerError, "failed to get payment data")
			return
		}
		for _, d := range data {
			points = append(points, dto.ChartDataPoint{Date: d.Date.Format("2006-01-02"), Value: d.Value})
		}
	default:
		s.respondError(w, http.StatusBadRequest, "invalid metric: use registrations or payments")
		return
	}

	if points == nil {
		points = []dto.ChartDataPoint{}
	}

	s.respondJSON(w, http.StatusOK, dto.ChartDataResponse{
		Points: points,
		Metric: metric,
		Period: period,
	})
}

// ==================== Task 2: SSE stream endpoint ====================

// handleAdminStatsStream sends Server-Sent Events with real-time admin stats.
// Supports ?token= query param for auth since EventSource can't send headers.
func (s *Server) handleAdminStatsStream(w http.ResponseWriter, r *http.Request) {
	// Check if user came through normal auth middleware
	user := auth.GetUserFromContext(r.Context())

	// Fallback: support ?token= query param for EventSource which can't send headers.
	// Only JWT access tokens are accepted — API tokens (sk_) must NOT be used for web auth.
	if user == nil {
		tokenStr := r.URL.Query().Get("token")
		if tokenStr != "" {
			claims, err := s.authService.ValidateAccessToken(tokenStr)
			if err == nil && claims != nil && claims.IsAdmin {
				user = &auth.AuthenticatedUser{
					ID:      claims.UserID,
					Phone:   claims.Phone,
					IsAdmin: claims.IsAdmin,
				}
			}
		}
	}

	if user == nil || !user.IsAdmin {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.respondError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	rc := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send initial ping
	rc.SetWriteDeadline(time.Now().Add(120 * time.Second))
	_, _ = fmt.Fprintf(w, ": ping\n\n")
	flusher.Flush()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send initial stats immediately
	rc.SetWriteDeadline(time.Now().Add(120 * time.Second))
	s.sendAdminStatsEvent(w, flusher)

	for {
		select {
		case <-ticker.C:
			rc.SetWriteDeadline(time.Now().Add(120 * time.Second))
			s.sendAdminStatsEvent(w, flusher)
		case <-r.Context().Done():
			return
		}
	}
}

// sendAdminStatsEvent writes a single stats_update SSE event.
func (s *Server) sendAdminStatsEvent(w http.ResponseWriter, flusher http.Flusher) {
	var stats Stats
	if s.tunnelProvider != nil {
		stats = s.tunnelProvider.GetStats()
	}
	totalUsers, err := s.db.Users.Count()
	if err != nil {
		s.log.Error().Err(err).Msg("failed to count users for admin stats")
	}

	data, _ := json.Marshal(dto.StatsResponse{
		ActiveClients: stats.ActiveClients,
		ActiveTunnels: stats.ActiveTunnels,
		HTTPTunnels:   stats.HTTPTunnels,
		TCPTunnels:    stats.TCPTunnels,
		UDPTunnels:    stats.UDPTunnels,
		TotalUsers:    totalUsers,
	})

	_, _ = fmt.Fprintf(w, "event: stats_update\ndata: %s\n\n", data)
	flusher.Flush()
}

// ==================== Task 3: Bulk operations ====================

// handleBulkUsers performs bulk user operations (block/unblock/delete/change_plan)
func (s *Server) handleBulkUsers(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.BulkUsersRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.UserIDs) == 0 {
		s.respondError(w, http.StatusBadRequest, "user_ids is required")
		return
	}
	if len(req.UserIDs) > 100 {
		s.respondError(w, http.StatusBadRequest, "max 100 users per batch")
		return
	}

	// Validate action upfront
	validActions := map[string]bool{"block": true, "unblock": true, "delete": true, "change_plan": true}
	if !validActions[req.Action] {
		s.respondError(w, http.StatusBadRequest, "invalid action: use block, unblock, delete, or change_plan")
		return
	}

	if req.Action == "change_plan" && req.PlanID == nil {
		s.respondError(w, http.StatusBadRequest, "plan_id is required for change_plan action")
		return
	}

	ipAddress := auth.GetClientIP(r)
	var successCount int
	var errs []string

	switch req.Action {
	case "block":
		affected, err := s.db.Users.BulkUpdateActive(req.UserIDs, false, currentUser.ID)
		if err != nil {
			s.log.Error().Err(err).Msg("bulk block operation failed")
			s.respondError(w, http.StatusInternalServerError, "bulk operation failed")
			return
		}
		successCount = int(affected)

	case "unblock":
		affected, err := s.db.Users.BulkUpdateActive(req.UserIDs, true, currentUser.ID)
		if err != nil {
			s.log.Error().Err(err).Msg("bulk unblock operation failed")
			s.respondError(w, http.StatusInternalServerError, "bulk operation failed")
			return
		}
		successCount = int(affected)

	case "change_plan":
		// Validate plan exists
		if _, err := s.db.Plans.GetByID(*req.PlanID); err != nil {
			s.respondError(w, http.StatusBadRequest, "invalid plan_id")
			return
		}
		affected, err := s.db.Users.BulkUpdatePlan(req.UserIDs, *req.PlanID, currentUser.ID)
		if err != nil {
			s.log.Error().Err(err).Msg("bulk change_plan operation failed")
			s.respondError(w, http.StatusInternalServerError, "bulk operation failed")
			return
		}
		successCount = int(affected)

	case "delete":
		affected, deleteErrs, err := s.db.Users.BulkDelete(req.UserIDs, currentUser.ID)
		if err != nil {
			s.log.Error().Err(err).Msg("bulk delete operation failed")
			s.respondError(w, http.StatusInternalServerError, "bulk operation failed")
			return
		}
		successCount = int(affected)
		errs = deleteErrs
	}

	// Log a single audit entry for the bulk action
	auditAction := database.ActionUserUpdated
	if req.Action == "delete" {
		auditAction = database.ActionUserDeleted
	}
	_ = s.db.Audit.Log(&currentUser.ID, auditAction, map[string]interface{}{
		"bulk_action":   req.Action,
		"target_users":  req.UserIDs,
		"success_count": successCount,
	}, ipAddress)

	if errs == nil {
		errs = []string{}
	}

	s.respondJSON(w, http.StatusOK, dto.BulkOperationResponse{
		SuccessCount: successCount,
		ErrorCount:   len(errs),
		Errors:       errs,
	})
}

// handleBulkCloseTunnels closes multiple tunnels at once
func (s *Server) handleBulkCloseTunnels(w http.ResponseWriter, r *http.Request) {
	if s.tunnelProvider == nil {
		s.respondError(w, http.StatusServiceUnavailable, "tunnel provider not available")
		return
	}

	var req dto.BulkTunnelsCloseRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.TunnelIDs) == 0 {
		s.respondError(w, http.StatusBadRequest, "tunnel_ids is required")
		return
	}

	if len(req.TunnelIDs) > 100 {
		s.respondError(w, http.StatusBadRequest, "max 100 tunnels per batch")
		return
	}

	var successCount int
	var errs []string

	for _, tid := range req.TunnelIDs {
		if err := s.tunnelProvider.AdminCloseTunnel(tid); err != nil {
			s.log.Error().Err(err).Str("tunnel_id", tid).Msg("bulk close tunnel failed")
			errs = append(errs, fmt.Sprintf("tunnel %s: operation failed", tid))
			continue
		}
		successCount++
	}

	if errs == nil {
		errs = []string{}
	}

	s.respondJSON(w, http.StatusOK, dto.BulkOperationResponse{
		SuccessCount: successCount,
		ErrorCount:   len(errs),
		Errors:       errs,
	})
}

// ==================== Task 4: Settings and system info ====================

// handleGetSettings returns read-only server configuration (no secrets)
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings := map[string]interface{}{
		"server": map[string]interface{}{
			"control_port":        s.cfg.Server.ControlPort,
			"http_port":           s.cfg.Server.HTTPPort,
			"tcp_port_range_min":  s.cfg.Server.TCPPortRange.Min,
			"tcp_port_range_max":  s.cfg.Server.TCPPortRange.Max,
			"udp_port_range_min":  s.cfg.Server.UDPPortRange.Min,
			"udp_port_range_max":  s.cfg.Server.UDPPortRange.Max,
			"compression_enabled": s.cfg.Server.CompressionEnabled,
		},
		"web": map[string]interface{}{
			"port":         s.cfg.Web.Port,
			"cors_origins": s.cfg.Web.CORSOrigins,
			"rate_limit": map[string]interface{}{
				"enabled":        s.cfg.Web.RateLimit.Enabled,
				"auth_per_min":   s.cfg.Web.RateLimit.AuthPerMin,
				"global_per_min": s.cfg.Web.RateLimit.GlobalPerMin,
			},
		},
		"domain": map[string]interface{}{
			"base":     s.cfg.Domain.Base,
			"aliases":  s.cfg.Domain.Aliases,
			"wildcard": s.cfg.Domain.Wildcard,
		},
		"features": map[string]interface{}{
			"tls_enabled":           s.cfg.TLS.Enabled,
			"totp_enabled":          s.cfg.TOTP.Enabled,
			"custom_domains":        s.cfg.CustomDomains.Enabled,
			"inspect_enabled":       s.cfg.Inspect.Enabled,
			"downloads_enabled":     s.cfg.Downloads.Enabled,
			"oauth_github":          s.cfg.OAuth.GitHub.GetCredentials(s.cfg.Domain.Base) != nil,
			"oauth_google":          s.cfg.OAuth.Google.ClientID != "",
			"yookassa_enabled":      s.cfg.YooKassa.Enabled,
			"creem_enabled":         s.cfg.Creem.Enabled,
			"smtp_enabled":          s.cfg.SMTP.Enabled,
			"telegram_enabled":      s.cfg.Telegram.Enabled,
			"redis_enabled":         s.cfg.Redis.Enabled,
		},
		"mode": string(s.cfg.EffectiveMode()),
	}

	s.respondJSON(w, http.StatusOK, settings)
}

// handleGetSystemInfo returns runtime system information
func (s *Server) handleGetSystemInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"version":    s.version,
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"num_cpu":    runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
	}

	s.respondJSON(w, http.StatusOK, info)
}

// ==================== Task 5: Invite codes admin ====================

// handleListInviteCodes returns invite codes with pagination
func (s *Server) handleListInviteCodes(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	codes, total, err := s.db.InviteCodes.List(limit, offset)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to list invite codes")
		s.respondError(w, http.StatusInternalServerError, "failed to list invite codes")
		return
	}

	codeDTOs := make([]*dto.InviteCodeDTO, len(codes))
	for i, c := range codes {
		codeDTOs[i] = &dto.InviteCodeDTO{
			ID:              c.ID,
			Code:            c.Code,
			CreatedByUserID: c.CreatedByUserID,
			UsedByUserID:    c.UsedByUserID,
			UsedAt:          c.UsedAt,
			ExpiresAt:       c.ExpiresAt,
			CreatedAt:       c.CreatedAt,
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"codes": codeDTOs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// handleCreateInviteCode creates a new invite code
func (s *Server) handleCreateInviteCode(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.CreateInviteCodeRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Generate random code if not provided
	code := req.Code
	if code == "" {
		code = generateInviteCodeString()
	}

	inviteCode, err := s.db.InviteCodes.Create(code, currentUser.ID)
	if err != nil {
		s.log.Error().Err(err).Str("code_prefix", code[:4]+"...").Msg("Failed to create invite code")
		s.respondError(w, http.StatusInternalServerError, "failed to create invite code")
		return
	}

	s.respondJSON(w, http.StatusCreated, &dto.InviteCodeDTO{
		ID:              inviteCode.ID,
		Code:            inviteCode.Code,
		CreatedByUserID: inviteCode.CreatedByUserID,
		CreatedAt:       inviteCode.CreatedAt,
	})
}

// handleDeleteInviteCode deletes an invite code
func (s *Server) handleDeleteInviteCode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := s.db.InviteCodes.Delete(id); err != nil {
		if errors.Is(err, database.ErrInviteCodeNotFound) {
			s.respondError(w, http.StatusNotFound, "invite code not found")
			return
		}
		s.log.Error().Err(err).Int64("id", id).Msg("Failed to delete invite code")
		s.respondError(w, http.StatusInternalServerError, "failed to delete invite code")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "invite code deleted",
	})
}

// generateInviteCodeString generates a random 8-character hex invite code.
func generateInviteCodeString() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
