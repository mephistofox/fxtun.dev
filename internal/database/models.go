package database

import (
	"net"
	"time"
)

// User represents a registered user
type User struct {
	ID           int64      `json:"id"`
	Phone        string     `json:"phone"`
	PasswordHash string     `json:"-"`
	DisplayName  string     `json:"display_name"`
	IsAdmin      bool       `json:"is_admin"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	GitHubID     *int64     `json:"github_id,omitempty"`
	GoogleID     *string    `json:"google_id,omitempty"`
	Email        string     `json:"email,omitempty"`
	AvatarURL    string     `json:"avatar_url,omitempty"`
	PlanID       int64      `json:"plan_id"`
}

// Plan represents a subscription plan
type Plan struct {
	ID                 int64   `json:"id"`
	Slug               string  `json:"slug"`
	Name               string  `json:"name"`
	Price              float64 `json:"price"`
	MaxTunnels         int     `json:"max_tunnels"`
	MaxDomains         int     `json:"max_domains"`
	MaxCustomDomains   int     `json:"max_custom_domains"`
	MaxTokens          int     `json:"max_tokens"`
	MaxTunnelsPerToken int     `json:"max_tunnels_per_token"`
	InspectorEnabled   bool    `json:"inspector_enabled"`
	IsPublic           bool    `json:"is_public"`
	IsRecommended      bool    `json:"is_recommended"`
}

// InviteCode represents a one-time invitation code
type InviteCode struct {
	ID              int64      `json:"id"`
	Code            string     `json:"code"`
	CreatedByUserID *int64     `json:"created_by_user_id,omitempty"`
	UsedByUserID    *int64     `json:"used_by_user_id,omitempty"`
	UsedAt          *time.Time `json:"used_at,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// IsUsed returns true if the invite code has been used
func (i *InviteCode) IsUsed() bool {
	return i.UsedByUserID != nil
}

// IsExpired returns true if the invite code has expired
func (i *InviteCode) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*i.ExpiresAt)
}

// ReservedDomain represents a subdomain reserved by a user
type ReservedDomain struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Subdomain string    `json:"subdomain"`
	CreatedAt time.Time `json:"created_at"`
}

// Session represents a user session (refresh token)
type Session struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	RefreshTokenHash string    `json:"-"`
	UserAgent        string    `json:"user_agent,omitempty"`
	IPAddress        string    `json:"ip_address,omitempty"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// IsExpired returns true if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// APIToken represents an API token for CLI authentication
type APIToken struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	TokenHash         string     `json:"-"`
	Name              string     `json:"name"`
	AllowedSubdomains []string   `json:"allowed_subdomains"`
	MaxTunnels        int        `json:"max_tunnels"`
	AllowedIPs        []string   `json:"allowed_ips,omitempty"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// CanUseSubdomain checks if the token allows using a specific subdomain
func (t *APIToken) CanUseSubdomain(subdomain string) bool {
	for _, pattern := range t.AllowedSubdomains {
		if pattern == "*" {
			return true
		}
		if matchWildcard(pattern, subdomain) {
			return true
		}
	}
	return false
}

// matchWildcard matches a pattern like "dev-*" against a subdomain
func matchWildcard(pattern, subdomain string) bool {
	if len(pattern) == 0 {
		return len(subdomain) == 0
	}

	// Handle wildcard at the end
	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(subdomain) >= len(prefix) && subdomain[:len(prefix)] == prefix
	}

	// Handle wildcard at the beginning
	if pattern[0] == '*' {
		suffix := pattern[1:]
		return len(subdomain) >= len(suffix) && subdomain[len(subdomain)-len(suffix):] == suffix
	}

	return pattern == subdomain
}

// IsIPAllowed checks if the given IP is allowed for this token.
// Empty AllowedIPs means all IPs are allowed.
func (t *APIToken) IsIPAllowed(ip string) bool {
	if len(t.AllowedIPs) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		host = ip // no port, use as-is
	}
	for _, allowed := range t.AllowedIPs {
		if allowed == host {
			return true
		}
	}
	return false
}

// TOTPSecret represents TOTP 2FA settings for a user
type TOTPSecret struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	SecretEncrypted string    `json:"-"`
	IsEnabled       bool      `json:"is_enabled"`
	BackupCodes     []string  `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        int64                  `json:"id"`
	UserID    *int64                 `json:"user_id,omitempty"`
	Action    string                 `json:"action"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// Audit log action constants
const (
	ActionLogin           = "login"
	ActionLogout          = "logout"
	ActionRegister        = "register"
	ActionPasswordChange  = "password_change"
	ActionTokenCreated    = "token_created"
	ActionTokenDeleted    = "token_deleted"
	ActionDomainReserved  = "domain_reserved"
	ActionDomainReleased  = "domain_released"
	ActionTunnelCreated   = "tunnel_created"
	ActionTunnelClosed    = "tunnel_closed"
	ActionTOTPEnabled     = "totp_enabled"
	ActionTOTPDisabled    = "totp_disabled"
	ActionInviteCreated   = "invite_created"
	ActionInviteUsed      = "invite_used"
	ActionUserUpdated     = "user_updated"
	ActionUserDeleted     = "user_deleted"
	ActionUsersMerged     = "users_merged"
	ActionPasswordReset   = "password_reset"
)

// CustomDomain represents a user-bound custom domain
type CustomDomain struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	Domain          string     `json:"domain"`
	TargetSubdomain string     `json:"target_subdomain"`
	Verified        bool       `json:"verified"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// TLSCertificate represents a stored TLS certificate
type TLSCertificate struct {
	ID        int64     `json:"id"`
	Domain    string    `json:"domain"`
	CertPEM   []byte    `json:"-"`
	KeyPEM    []byte    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
	CreatedAt time.Time `json:"created_at"`
}

// UserBundle represents a tunnel configuration bundle for a user
type UserBundle struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	LocalPort   int       `json:"local_port"`
	Subdomain   string    `json:"subdomain,omitempty"`
	RemotePort  int       `json:"remote_port,omitempty"`
	AutoConnect bool      `json:"auto_connect"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserHistoryEntry represents a connection history entry for a user
type UserHistoryEntry struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	BundleName     string     `json:"bundle_name,omitempty"`
	TunnelType     string     `json:"tunnel_type"`
	LocalPort      int        `json:"local_port"`
	RemoteAddr     string     `json:"remote_addr,omitempty"`
	URL            string     `json:"url,omitempty"`
	ConnectedAt    time.Time  `json:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at,omitempty"`
	BytesSent      int64      `json:"bytes_sent"`
	BytesReceived  int64      `json:"bytes_received"`
}

// UserSetting represents a user setting key-value pair
type UserSetting struct {
	UserID    int64     `json:"user_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
