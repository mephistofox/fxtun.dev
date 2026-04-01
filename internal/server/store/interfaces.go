package store

import (
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/database"
)

// SessionStore manages user refresh-token sessions.
type SessionStore interface {
	Create(session *database.Session) error
	GetByTokenHash(tokenHash string) (*database.Session, error)
	GetByUserID(userID int64) ([]*database.Session, error)
	Delete(id int64) error
	DeleteByTokenHash(tokenHash string) error
	DeleteByUserID(userID int64) error
	DeleteExpired() (int64, error)
}

// DeviceSession represents a device login flow session.
type DeviceSession struct {
	ID        string
	Status    string // "pending", "authorized", "expired"
	Token     string
	CreatedAt time.Time
}

// DeviceStore manages device login sessions.
type DeviceStore interface {
	Create() (*DeviceSession, error)
	Get(id string) *DeviceSession
	Authorize(id, token string) bool
	Delete(id string)
}

// OAuthStateEntry holds in-flight OAuth state.
type OAuthStateEntry struct {
	Purpose         string // "login" or "link"
	UserID          int64
	DesktopRedirect string
}

// OAuthCodeEntry holds a one-time authorization code bundle.
type OAuthCodeEntry struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// OAuthStore manages OAuth flow state and authorization codes.
type OAuthStore interface {
	CreateState(entry *OAuthStateEntry) (string, error)
	ConsumeState(nonce string) *OAuthStateEntry
	CreateCode(accessToken, refreshToken string, expiresIn int64) (string, error)
	ExchangeCode(code string) *OAuthCodeEntry
}

// RateChecker checks if a request from an IP should be allowed.
type RateChecker interface {
	Allow(ip string) bool
}

// TunnelEntry describes a tunnel registered in the cross-server registry.
type TunnelEntry struct {
	TunnelID   string
	Type       string
	Name       string
	Subdomain  string
	RemotePort int
	LocalPort  int
	ClientID   string
	UserID     int64
	ServerID   string
	CreatedAt  time.Time
}

// TunnelRegistry provides cross-server tunnel discovery.
type TunnelRegistry interface {
	Register(entry TunnelEntry) error
	Unregister(tunnelID string) error
	LookupBySubdomain(subdomain string) (*TunnelEntry, error)
	ListByUser(userID int64) ([]TunnelEntry, error)
	Heartbeat(tunnelID string) error
}

// TLSCache provides a shared TLS certificate cache.
type TLSCache interface {
	Get(domain string) (certPEM, keyPEM []byte, err error)
	Put(domain string, certPEM, keyPEM []byte, expiresAt time.Time) error
	Delete(domain string) error
}
