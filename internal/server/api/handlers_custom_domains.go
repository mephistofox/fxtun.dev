package api

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtun.dev/internal/server/auth"
	"github.com/mephistofox/fxtun.dev/internal/server/database"
	fxdns "github.com/mephistofox/fxtun.dev/internal/server/dns"
	fxtls "github.com/mephistofox/fxtun.dev/internal/server/tls"
)

// verifyCooldown throttles custom-domain verification per domain. Every attempt
// performs outbound TXT/host lookups against a name the caller chose (turning
// the server into a DNS reflector) and, on success, starts an ACME order, so it
// must not be callable in a tight loop.
//
// ponytail: process-local map — fine for the single API node we run; move it to
// the existing Redis rate limiter if the API is ever scaled out.
var verifyCooldown sync.Map // domain -> time.Time of the last attempt

const verifyCooldownWindow = 30 * time.Second

// reservedDomains lists hostnames this installation owns: the base domain, its
// aliases and every authoritative DNS zone we serve. None of them (nor their
// subdomains) may be claimed as a tenant's "custom" domain.
func (s *Server) reservedDomains() []string {
	reserved := make([]string, 0, 4+len(s.cfg.Domain.Aliases))
	reserved = append(reserved, s.baseDomain)
	reserved = append(reserved, s.cfg.Domain.Aliases...)
	if s.cfg.DNS.ZoneFile != "" {
		if zf, err := fxdns.LoadZoneFile(s.cfg.DNS.ZoneFile); err == nil {
			for _, z := range zf.Zones {
				reserved = append(reserved, z.Name)
			}
		} else {
			s.log.Warn().Err(err).Msg("Failed to load DNS zone file for custom domain validation")
		}
	}
	return reserved
}

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
		"domains": domains,
		"total":   len(domains),
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

	// Normalize BEFORE validating and storing. The runtime routing map
	// lowercases on lookup and removal, so a row kept as "EXAMPLE.com" would
	// evict the entry of whoever owns "example.com".
	reqDomain := fxtls.NormalizeDomain(req.Domain)

	if err := fxtls.ValidateCustomDomain(reqDomain, s.reservedDomains()...); err != nil {
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

	// Issue a unique ownership-proof token. The domain is NOT auto-verified:
	// pointing an A-record at the shared server IP does not prove ownership
	// (any tenant can do that). The user must publish the token as a TXT
	// record and then call verify.
	domain := &database.CustomDomain{
		UserID:            user.ID,
		Domain:            reqDomain,
		TargetSubdomain:   req.TargetSubdomain,
		VerificationToken: "fxtunnel-verify=" + randomHex(24),
		Verified:          false,
	}

	if err := s.db.CustomDomains.Create(domain); err != nil {
		if errors.Is(err, database.ErrCustomDomainAlreadyExists) {
			s.respondErrorWithCode(w, http.StatusConflict, "DOMAIN_TAKEN", "domain already registered")
			return
		}
		s.respondError(w, http.StatusInternalServerError, "failed to create custom domain")
		return
	}

	ipAddress := auth.GetClientIP(r)
	_ = s.db.Audit.Log(&user.ID, "custom_domain_added", map[string]interface{}{
		"domain":           reqDomain,
		"target_subdomain": req.TargetSubdomain,
		"verified":         false,
	}, ipAddress)

	s.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"domain":           domain,
		"txt_record_name":  fxtls.ChallengeRecordName(domain.Domain),
		"txt_record_value": domain.VerificationToken,
		"target":           req.TargetSubdomain + "." + s.baseDomain,
	})
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

	// Already verified — nothing to look up, nothing to issue.
	if domain.Verified {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{"verified": true})
		return
	}

	if last, ok := verifyCooldown.Load(domain.Domain); ok {
		if elapsed := time.Since(last.(time.Time)); elapsed < verifyCooldownWindow {
			w.Header().Set("Retry-After", strconv.Itoa(int((verifyCooldownWindow-elapsed).Seconds())+1))
			s.respondErrorWithCode(w, http.StatusTooManyRequests, "VERIFY_COOLDOWN", "verification attempted too recently, try again shortly")
			return
		}
	}
	verifyCooldown.Store(domain.Domain, time.Now())

	// Legacy rows (created before TXT verification) have no token — issue one
	// lazily so the owner can complete verification instead of being stranded.
	if domain.VerificationToken == "" {
		domain.VerificationToken = "fxtunnel-verify=" + randomHex(24)
		if err := s.db.CustomDomains.SetVerificationToken(domain.ID, domain.VerificationToken); err != nil {
			s.respondError(w, http.StatusInternalServerError, "failed to issue verification token")
			return
		}
	}

	// Ownership proof MUST come first: the TXT challenge proves the user
	// controls the domain's DNS. Without it, an A-record pointing at the
	// shared server IP would let one tenant claim another tenant's domain.
	if err := fxtls.VerifyTXT(domain.Domain, domain.VerificationToken); err != nil {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"verified":         false,
			"error":            err.Error(),
			"txt_record_name":  fxtls.ChallengeRecordName(domain.Domain),
			"txt_record_value": domain.VerificationToken,
		})
		return
	}

	// Ownership confirmed — now check the routing record (CNAME/A) so traffic
	// actually reaches the user's tunnel.
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
