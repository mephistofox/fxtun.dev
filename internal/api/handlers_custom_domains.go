package api

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/database"
	fxtls "github.com/mephistofox/fxtunnel/internal/tls"
)

func (s *Server) handleListCustomDomains(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	domains, err := s.db.CustomDomains.GetByUserID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to list custom domains")
		s.respondError(w, http.StatusInternalServerError, "failed to list custom domains")
		return
	}

	serverIP := ""
	if ips, err := net.LookupHost(s.baseDomain); err == nil && len(ips) > 0 {
		serverIP = ips[0]
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"domains":     domains,
		"total":       len(domains),
		"max_domains": func() int {
			if user.Plan != nil {
				return user.Plan.MaxCustomDomains
			}
			return 0
		}(),
		"base_domain": s.baseDomain,
		"server_ip":   serverIP,
	})
}

func (s *Server) handleAddCustomDomain(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Domain          string `json:"domain"`
		TargetSubdomain string `json:"target_subdomain"`
	}
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := fxtls.ValidateCustomDomain(req.Domain, s.baseDomain); err != nil {
		s.respondErrorWithCode(w, http.StatusBadRequest, "INVALID_DOMAIN", err.Error())
		return
	}

	owned, err := s.db.Domains.IsOwnedByUser(req.TargetSubdomain, user.ID)
	if err != nil || !owned {
		s.respondErrorWithCode(w, http.StatusBadRequest, "INVALID_SUBDOMAIN", "target subdomain not owned by you")
		return
	}

	count, err := s.db.CustomDomains.CountByUserID(user.ID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to check limit")
		return
	}
	maxCustomDomains := 0
	if user.Plan != nil {
		maxCustomDomains = user.Plan.MaxCustomDomains
	}
	if maxCustomDomains >= 0 && count >= maxCustomDomains {
		s.respondErrorWithCode(w, http.StatusConflict, "LIMIT_REACHED", "custom domain limit reached")
		return
	}

	expectedTarget := req.TargetSubdomain + "." + s.baseDomain
	verified := fxtls.VerifyDNS(req.Domain, expectedTarget) == nil

	domain := &database.CustomDomain{
		UserID:          user.ID,
		Domain:          req.Domain,
		TargetSubdomain: req.TargetSubdomain,
		Verified:        verified,
	}

	if err := s.db.CustomDomains.Create(domain); err != nil {
		if errors.Is(err, database.ErrCustomDomainAlreadyExists) {
			s.respondErrorWithCode(w, http.StatusConflict, "DOMAIN_TAKEN", "domain already registered")
			return
		}
		s.respondError(w, http.StatusInternalServerError, "failed to create custom domain")
		return
	}

	if verified && s.customDomainManager != nil {
		s.customDomainManager.AddCustomDomain(domain)
		if cm := s.customDomainManager.CertManager(); cm != nil {
			cm.ObtainCert(domain.Domain)
		}
	}

	ipAddress := auth.GetClientIP(r)
	_ = s.db.Audit.Log(&user.ID, "custom_domain_added", map[string]interface{}{
		"domain":           req.Domain,
		"target_subdomain": req.TargetSubdomain,
		"verified":         verified,
	}, ipAddress)

	s.respondJSON(w, http.StatusCreated, domain)
}

func (s *Server) handleDeleteCustomDomain(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	domain, err := s.db.CustomDomains.GetByID(id)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "custom domain not found")
		return
	}
	if domain.UserID != user.ID {
		s.respondError(w, http.StatusForbidden, "access denied")
		return
	}

	if err := s.db.CustomDomains.Delete(id); err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to delete")
		return
	}

	if s.customDomainManager != nil {
		s.customDomainManager.RemoveCustomDomain(domain.Domain)
		if cm := s.customDomainManager.CertManager(); cm != nil {
			cm.RemoveCert(domain.Domain)
		}
	}

	ipAddress := auth.GetClientIP(r)
	_ = s.db.Audit.Log(&user.ID, "custom_domain_removed", map[string]interface{}{
		"domain": domain.Domain,
	}, ipAddress)

	s.respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (s *Server) handleVerifyCustomDomain(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	domain, err := s.db.CustomDomains.GetByID(id)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "custom domain not found")
		return
	}
	if domain.UserID != user.ID {
		s.respondError(w, http.StatusForbidden, "access denied")
		return
	}

	expectedTarget := domain.TargetSubdomain + "." + s.baseDomain
	if err := fxtls.VerifyDNS(domain.Domain, expectedTarget); err != nil {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"verified": false,
			"error":    err.Error(),
			"expected": expectedTarget,
		})
		return
	}

	if err := s.db.CustomDomains.SetVerified(id, true); err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to update")
		return
	}

	domain.Verified = true
	if s.customDomainManager != nil {
		s.customDomainManager.AddCustomDomain(domain)
		if cm := s.customDomainManager.CertManager(); cm != nil {
			cm.ObtainCert(domain.Domain)
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"verified": true,
	})
}

// Admin handlers

func (s *Server) handleAdminListCustomDomains(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	domains, total, err := s.db.CustomDomains.GetAll(limit, offset)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to list custom domains")
		return
	}

	userIDs := make([]int64, 0, len(domains))
	for _, d := range domains {
		userIDs = append(userIDs, d.UserID)
	}
	usersMap, _ := s.db.Users.GetByIDs(userIDs)

	type adminDomain struct {
		*database.CustomDomain
		UserPhone string  `json:"user_phone"`
		TLSExpiry *string `json:"tls_expiry,omitempty"`
	}

	result := make([]adminDomain, len(domains))
	for i, d := range domains {
		ad := adminDomain{CustomDomain: d}
		if u, ok := usersMap[d.UserID]; ok {
			ad.UserPhone = u.Phone
		}
		if cert, err := s.db.TLSCerts.GetByDomain(d.Domain); err == nil {
			exp := cert.ExpiresAt.Format(time.RFC3339)
			ad.TLSExpiry = &exp
		}
		result[i] = ad
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"domains": result,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

func (s *Server) handleAdminDeleteCustomDomain(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	domain, err := s.db.CustomDomains.GetByID(id)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "custom domain not found")
		return
	}

	if err := s.db.CustomDomains.Delete(id); err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to delete")
		return
	}

	if s.customDomainManager != nil {
		s.customDomainManager.RemoveCustomDomain(domain.Domain)
		if cm := s.customDomainManager.CertManager(); cm != nil {
			cm.RemoveCert(domain.Domain)
		}
	}

	user := auth.GetUserFromContext(r.Context())
	ipAddress := auth.GetClientIP(r)
	_ = s.db.Audit.Log(&user.ID, "admin_custom_domain_removed", map[string]interface{}{
		"domain":  domain.Domain,
		"user_id": domain.UserID,
	}, ipAddress)

	s.respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
