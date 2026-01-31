package dto

import (
	"time"

	"github.com/mephistofox/fxtunnel/internal/database"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// UserDTO represents a user in API responses
type UserDTO struct {
	ID          int64      `json:"id"`
	Phone       string     `json:"phone"`
	DisplayName string     `json:"display_name"`
	IsAdmin     bool       `json:"is_admin"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// UserFromModel converts a database User to UserDTO
func UserFromModel(u *database.User) *UserDTO {
	return &UserDTO{
		ID:          u.ID,
		Phone:       u.Phone,
		DisplayName: u.DisplayName,
		IsAdmin:     u.IsAdmin,
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt,
		LastLoginAt: u.LastLoginAt,
	}
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User         *UserDTO `json:"user"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"`
}

// ProfileResponse represents a user profile response
type ProfileResponse struct {
	User            *UserDTO          `json:"user"`
	TOTPEnabled     bool              `json:"totp_enabled"`
	ReservedDomains []*DomainDTO      `json:"reserved_domains"`
	MaxDomains      int               `json:"max_domains"`
	TokenCount      int               `json:"token_count"`
}

// TokenDTO represents an API token in API responses
type TokenDTO struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	AllowedSubdomains []string   `json:"allowed_subdomains"`
	AllowedIPs        []string   `json:"allowed_ips,omitempty"`
	MaxTunnels        int        `json:"max_tunnels"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// TokenFromModel converts a database APIToken to TokenDTO
func TokenFromModel(t *database.APIToken) *TokenDTO {
	return &TokenDTO{
		ID:                t.ID,
		Name:              t.Name,
		AllowedSubdomains: t.AllowedSubdomains,
		AllowedIPs:        t.AllowedIPs,
		MaxTunnels:        t.MaxTunnels,
		LastUsedAt:        t.LastUsedAt,
		CreatedAt:         t.CreatedAt,
	}
}

// CreateTokenResponse represents a token creation response
type CreateTokenResponse struct {
	Token string    `json:"token"` // Plain text token - shown only once!
	Info  *TokenDTO `json:"info"`
}

// TokensListResponse represents a list of tokens
type TokensListResponse struct {
	Tokens []*TokenDTO `json:"tokens"`
	Total  int         `json:"total"`
}

// DomainDTO represents a reserved domain in API responses
type DomainDTO struct {
	ID        int64     `json:"id"`
	Subdomain string    `json:"subdomain"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

// DomainFromModel converts a database ReservedDomain to DomainDTO
func DomainFromModel(d *database.ReservedDomain, baseDomain string) *DomainDTO {
	return &DomainDTO{
		ID:        d.ID,
		Subdomain: d.Subdomain,
		URL:       "https://" + d.Subdomain + "." + baseDomain,
		CreatedAt: d.CreatedAt,
	}
}

// DomainsListResponse represents a list of domains
type DomainsListResponse struct {
	Domains    []*DomainDTO `json:"domains"`
	Total      int          `json:"total"`
	MaxDomains int          `json:"max_domains"`
}

// DomainCheckResponse represents a domain availability check response
type DomainCheckResponse struct {
	Subdomain string `json:"subdomain"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"` // "taken", "reserved", "invalid"
}

// TunnelDTO represents a tunnel in API responses
type TunnelDTO struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"` // http, tcp, udp
	Name       string    `json:"name"`
	Subdomain  string    `json:"subdomain,omitempty"`
	RemotePort int       `json:"remote_port,omitempty"`
	LocalPort  int       `json:"local_port"`
	URL        string    `json:"url,omitempty"`
	ClientID   string    `json:"client_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// TunnelsListResponse represents a list of tunnels
type TunnelsListResponse struct {
	Tunnels []*TunnelDTO `json:"tunnels"`
	Total   int          `json:"total"`
}

// InviteCodeDTO represents an invite code in API responses
type InviteCodeDTO struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	Used      bool       `json:"used"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// InviteCodeFromModel converts a database InviteCode to InviteCodeDTO
func InviteCodeFromModel(i *database.InviteCode) *InviteCodeDTO {
	return &InviteCodeDTO{
		ID:        i.ID,
		Code:      i.Code,
		Used:      i.IsUsed(),
		UsedAt:    i.UsedAt,
		ExpiresAt: i.ExpiresAt,
		CreatedAt: i.CreatedAt,
	}
}

// InviteCodesListResponse represents a list of invite codes
type InviteCodesListResponse struct {
	Codes []*InviteCodeDTO `json:"codes"`
	Total int              `json:"total"`
}

// TOTPEnableResponse represents a TOTP enable response
type TOTPEnableResponse struct {
	Secret      string   `json:"secret"`
	QRCode      string   `json:"qr_code"` // Data URL
	BackupCodes []string `json:"backup_codes"`
}

// DownloadDTO represents a client download in API responses
type DownloadDTO struct {
	Platform   string `json:"platform"`    // linux-amd64, darwin-arm64, windows-amd64
	OS         string `json:"os"`          // Linux, macOS, Windows
	Arch       string `json:"arch"`        // amd64, arm64
	Size       int64  `json:"size"`        // bytes
	URL        string `json:"url"`         // /api/downloads/:platform
	ClientType string `json:"client_type"` // cli, gui
}

// DownloadsListResponse represents a list of available downloads
type DownloadsListResponse struct {
	Clients    []*DownloadDTO `json:"clients"`     // CLI clients (deprecated, use cli field)
	CLI        []*DownloadDTO `json:"cli"`         // CLI clients
	GUI        []*DownloadDTO `json:"gui"`         // GUI clients
}

// StatsResponse represents server statistics
type StatsResponse struct {
	ActiveClients    int   `json:"active_clients"`
	ActiveTunnels    int   `json:"active_tunnels"`
	HTTPTunnels      int   `json:"http_tunnels"`
	TCPTunnels       int   `json:"tcp_tunnels"`
	UDPTunnels       int   `json:"udp_tunnels"`
	TotalUsers       int   `json:"total_users"`
	TotalConnections int64 `json:"total_connections"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
}

// AuditLogDTO represents an audit log entry in API responses
type AuditLogDTO struct {
	ID        int64                  `json:"id"`
	UserID    *int64                 `json:"user_id,omitempty"`
	UserPhone string                 `json:"user_phone,omitempty"`
	Action    string                 `json:"action"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IPAddress string                 `json:"ip_address"`
	CreatedAt time.Time              `json:"created_at"`
}

// AuditLogFromModel converts a database AuditLog to AuditLogDTO
func AuditLogFromModel(a *database.AuditLog, userPhone string) *AuditLogDTO {
	return &AuditLogDTO{
		ID:        a.ID,
		UserID:    a.UserID,
		UserPhone: userPhone,
		Action:    a.Action,
		Details:   a.Details,
		IPAddress: a.IPAddress,
		CreatedAt: a.CreatedAt,
	}
}

// AuditLogsListResponse represents a list of audit logs
type AuditLogsListResponse struct {
	Logs  []*AuditLogDTO `json:"logs"`
	Total int            `json:"total"`
}

// AdminTunnelDTO represents a tunnel with owner info in API responses
type AdminTunnelDTO struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Name       string    `json:"name"`
	Subdomain  string    `json:"subdomain,omitempty"`
	RemotePort int       `json:"remote_port,omitempty"`
	LocalPort  int       `json:"local_port"`
	URL        string    `json:"url,omitempty"`
	ClientID   string    `json:"client_id"`
	UserID     int64     `json:"user_id"`
	UserPhone  string    `json:"user_phone"`
	CreatedAt  time.Time `json:"created_at"`
}

// AdminTunnelsListResponse represents a list of all tunnels for admin
type AdminTunnelsListResponse struct {
	Tunnels []*AdminTunnelDTO `json:"tunnels"`
	Total   int               `json:"total"`
}

// DeviceCodeResponse represents a device flow code response
type DeviceCodeResponse struct {
	SessionID string `json:"session_id"`
	AuthURL   string `json:"auth_url"`
	ExpiresIn int    `json:"expires_in"`
}

// DevicePollResponse represents a device flow poll response
type DevicePollResponse struct {
	Status string `json:"status"`
	Token  string `json:"token,omitempty"`
}

// UsersListResponse represents a list of users for admin
type UsersListResponse struct {
	Users []*UserDTO `json:"users"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}
