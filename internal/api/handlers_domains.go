package api

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/database"
)

var subdomainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,30}[a-z0-9])?$`)

// handleListDomains returns the user's reserved domains
func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	domains, err := s.db.Domains.GetByUserID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get domains")
		s.respondError(w, http.StatusInternalServerError, "failed to get domains")
		return
	}

	domainDTOs := make([]*dto.DomainDTO, len(domains))
	for i, d := range domains {
		domainDTOs[i] = dto.DomainFromModel(d, s.baseDomain)
	}

	s.respondJSON(w, http.StatusOK, dto.DomainsListResponse{
		Domains:    domainDTOs,
		Total:      len(domainDTOs),
		MaxDomains: s.authService.GetMaxDomains(),
	})
}

// handleReserveDomain reserves a subdomain for the user
func (s *Server) handleReserveDomain(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ReserveDomainRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Subdomain == "" {
		s.respondError(w, http.StatusBadRequest, "subdomain is required")
		return
	}

	// Validate subdomain format
	if !subdomainRegex.MatchString(req.Subdomain) {
		s.respondErrorWithCode(w, http.StatusBadRequest, "INVALID_SUBDOMAIN", "subdomain must be 3-32 characters, alphanumeric and hyphens only")
		return
	}

	// Check max domains limit
	count, err := s.db.Domains.Count(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to count domains")
		s.respondError(w, http.StatusInternalServerError, "failed to reserve domain")
		return
	}

	if count >= s.authService.GetMaxDomains() {
		s.respondErrorWithCode(w, http.StatusForbidden, "MAX_DOMAINS", "maximum domains reached")
		return
	}

	// Check if subdomain is available
	available, err := s.db.Domains.IsAvailable(req.Subdomain)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to check domain availability")
		s.respondError(w, http.StatusInternalServerError, "failed to reserve domain")
		return
	}

	if !available {
		s.respondErrorWithCode(w, http.StatusConflict, "SUBDOMAIN_TAKEN", "subdomain is already reserved")
		return
	}

	// Create reservation
	domain := &database.ReservedDomain{
		UserID:    user.ID,
		Subdomain: req.Subdomain,
	}

	if err := s.db.Domains.Create(domain); err != nil {
		if errors.Is(err, database.ErrDomainAlreadyExists) {
			s.respondErrorWithCode(w, http.StatusConflict, "SUBDOMAIN_TAKEN", "subdomain is already reserved")
			return
		}
		s.log.Error().Err(err).Msg("Failed to create domain")
		s.respondError(w, http.StatusInternalServerError, "failed to reserve domain")
		return
	}

	// Log audit
	ipAddress := auth.GetClientIP(r)
	s.db.Audit.Log(&user.ID, database.ActionDomainReserved, map[string]interface{}{
		"subdomain": req.Subdomain,
	}, ipAddress)

	s.respondJSON(w, http.StatusCreated, dto.DomainFromModel(domain, s.baseDomain))
}

// handleReleaseDomain releases a reserved domain
func (s *Server) handleReleaseDomain(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid domain id")
		return
	}

	// Get domain to verify ownership
	domain, err := s.db.Domains.GetByID(id)
	if err != nil {
		if errors.Is(err, database.ErrDomainNotFound) {
			s.respondError(w, http.StatusNotFound, "domain not found")
			return
		}
		s.log.Error().Err(err).Msg("Failed to get domain")
		s.respondError(w, http.StatusInternalServerError, "failed to release domain")
		return
	}

	// Check ownership
	if domain.UserID != user.ID && !user.IsAdmin {
		s.respondError(w, http.StatusForbidden, "access denied")
		return
	}

	// Delete domain
	if err := s.db.Domains.Delete(id); err != nil {
		s.log.Error().Err(err).Msg("Failed to delete domain")
		s.respondError(w, http.StatusInternalServerError, "failed to release domain")
		return
	}

	// Log audit
	ipAddress := auth.GetClientIP(r)
	s.db.Audit.Log(&user.ID, database.ActionDomainReleased, map[string]interface{}{
		"subdomain": domain.Subdomain,
	}, ipAddress)

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "domain released successfully",
	})
}

// handleCheckDomain checks if a subdomain is available
func (s *Server) handleCheckDomain(w http.ResponseWriter, r *http.Request) {
	subdomain := chi.URLParam(r, "subdomain")

	// Validate subdomain format
	if !subdomainRegex.MatchString(subdomain) {
		s.respondJSON(w, http.StatusOK, dto.DomainCheckResponse{
			Subdomain: subdomain,
			Available: false,
			Reason:    "invalid",
		})
		return
	}

	available, err := s.db.Domains.IsAvailable(subdomain)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to check domain availability")
		s.respondError(w, http.StatusInternalServerError, "failed to check domain")
		return
	}

	response := dto.DomainCheckResponse{
		Subdomain: subdomain,
		Available: available,
	}

	if !available {
		response.Reason = "reserved"
	}

	s.respondJSON(w, http.StatusOK, response)
}
