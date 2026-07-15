package store

import (
	"errors"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/server/database"
)

// ErrSubdomainTaken is returned when registering a tunnel whose subdomain is
// already claimed by a different user, preventing cross-node subdomain hijack.
var ErrSubdomainTaken = errors.New("subdomain already claimed by another user")

// RotatedTokenTracker is an optional capability of a SessionStore. It remembers
// recently rotated refresh-token hashes so that presenting an already-rotated
// token (a sign of token theft) can be detected as reuse. A SessionStore that
// does not implement it simply has no reuse detection.
type RotatedTokenTracker interface {
	// MarkRotated records that tokenHash was rotated away for userID, retained
	// for ttl (the remaining refresh-token lifetime).
	MarkRotated(tokenHash string, userID int64, ttl time.Duration) error
	// RotatedOwner returns the user a rotated token belonged to. found is false
	// when the hash was never recorded (or has since expired).
	RotatedOwner(tokenHash string) (userID int64, found bool, err error)
}

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

// MagicLinkEntry holds an in-flight passwordless email verification.
type MagicLinkEntry struct {
	Email    string // normalized, lowercased recipient address
	CodeHash string // hash of the 6-digit fallback code
	IP       string // requester IP (for audit)
	Lang     string // "ru" or "en", for the verification response
}

// MagicLinkStore manages passwordless magic-link tokens, the 6-digit code
// alternative, per-email send throttling, and code-attempt limiting.
type MagicLinkStore interface {
	// Create stores a verification entry keyed by a fresh opaque token (returned)
	// plus an email->token index, both retained for ttl. A new request for an
	// email replaces any previous pending entry.
	Create(entry *MagicLinkEntry, ttl time.Duration) (token string, err error)
	// ConsumeToken atomically retrieves and deletes the entry for token (and its
	// email index). Returns nil if not found or expired.
	ConsumeToken(token string) *MagicLinkEntry
	// LookupByToken returns the entry for a link token without consuming it, so
	// callers can run extra checks (e.g. TOTP) before burning the one-time link.
	LookupByToken(token string) (entry *MagicLinkEntry, ok bool)
	// LookupByEmail returns the active token and entry for an email (used by the
	// 6-digit code path) without consuming it. ok is false if none is pending.
	LookupByEmail(email string) (token string, entry *MagicLinkEntry, ok bool)
	// IncrAttempt increments and returns the failed-code attempt counter for
	// email, retained for ttl.
	IncrAttempt(email string, ttl time.Duration) int
	// AllowSend reports whether a new link may be sent to email now; when it
	// returns true it records a cooldown of the given duration.
	AllowSend(email string, cooldown time.Duration) bool
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

// NodeEntry describes an edge node registered in the cluster.
type NodeEntry struct {
	NodeID      string
	Name        string
	Region      string
	PublicAddr  string // control-plane address for client connections (host:port)
	HTTPAddr    string // HTTP address for inter-node reverse proxy (host:port)
	Status      string // "active"
	TunnelCount int
	ClientCount int
	UpdatedAt   time.Time
}

// NodeRegistry provides cross-server node discovery.
type NodeRegistry interface {
	RegisterNode(entry NodeEntry) error
	UnregisterNode(nodeID string) error
	GetNode(nodeID string) (*NodeEntry, error)
	ListActiveNodes() ([]NodeEntry, error)
	HeartbeatNode(nodeID string, tunnelCount, clientCount int) error
}

// TLSCache provides a shared TLS certificate cache.
type TLSCache interface {
	Get(domain string) (certPEM, keyPEM []byte, err error)
	Put(domain string, certPEM, keyPEM []byte, expiresAt time.Time) error
	Delete(domain string) error
}

// IPBanEntry describes an active IP ban.
type IPBanEntry struct {
	IP        string
	Reason    string
	BannedAt  time.Time
	ExpiresAt time.Time
}

// IPBanStore manages temporary IP bans (e.g. from tarpit hits).
type IPBanStore interface {
	// Ban records the IP as banned for ttl. Returns true if this is a new ban,
	// false if the IP was already banned (the TTL is refreshed in both cases).
	Ban(ip, reason string, ttl time.Duration) (isNew bool, err error)
	// IsBanned returns whether the IP is currently banned and the ban reason.
	IsBanned(ip string) (banned bool, reason string, err error)
	// Unban removes an active ban.
	Unban(ip string) error
	// List returns all currently active bans (best-effort; may be paginated under the hood).
	List() ([]IPBanEntry, error)
}
