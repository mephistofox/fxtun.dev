package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
)

// Database holds the database connection and repositories
type Database struct {
	db           *sql.DB
	log          zerolog.Logger
	CustomDomains *CustomDomainRepository
	TLSCerts      *TLSCertRepository
	Users         *UserRepository
	Sessions     *SessionRepository
	Tokens       *APITokenRepository
	Domains      *DomainRepository
	Invites      *InviteRepository
	TOTP         *TOTPRepository
	Audit        *AuditRepository
	UserBundles  *UserBundleRepository
	UserHistory  *UserHistoryRepository
	UserSettings *UserSettingsRepository
}

// New creates a new database connection and initializes repositories
func New(dbPath string, log zerolog.Logger) (*Database, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// SQLite only supports a single writer at a time. Allowing multiple open
	// connections would cause concurrent writes to fail with "database is locked"
	// errors, so we limit the pool to one connection.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	database := &Database{
		db:  db,
		log: log.With().Str("component", "database").Logger(),
	}

	// Run migrations
	if err := database.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// Initialize repositories
	database.CustomDomains = NewCustomDomainRepository(db)
	database.TLSCerts = NewTLSCertRepository(db)
	database.Users = NewUserRepository(db)
	database.Sessions = NewSessionRepository(db)
	database.Tokens = NewAPITokenRepository(db)
	database.Domains = NewDomainRepository(db)
	database.Invites = NewInviteRepository(db)
	database.TOTP = NewTOTPRepository(db)
	database.Audit = NewAuditRepository(db)
	database.UserBundles = NewUserBundleRepository(db)
	database.UserHistory = NewUserHistoryRepository(db)
	database.UserSettings = NewUserSettingsRepository(db)

	log.Info().Str("path", dbPath).Msg("Database initialized")

	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// DB returns the underlying sql.DB for transactions
func (d *Database) DB() *sql.DB {
	return d.db
}

// migrate runs all database migrations with version tracking
func (d *Database) migrate() error {
	// Create schema_migrations table
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	migrations := []string{
		migrationCreateUsers,
		migrationCreateInviteCodes,
		migrationCreateReservedDomains,
		migrationCreateSessions,
		migrationCreateAPITokens,
		migrationCreateTOTPSecrets,
		migrationCreateAuditLogs,
		migrationCreateIndexes,
		migrationCreateUserBundles,
		migrationCreateUserHistory,
		migrationCreateUserSettings,
		migrationAddAllowedIPs,
		migrationAddTokenAndSessionIndexes,
		migrationCreateCustomDomains,
		migrationCreateTLSCertificates,
	}

	// Bootstrap: if users table exists but schema_migrations is empty,
	// mark all existing migrations as applied (upgrade from pre-tracking database)
	var tableCount int
	_ = d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableCount)
	if tableCount > 0 {
		var migrationCount int
		_ = d.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount)
		if migrationCount == 0 {
			for i := range migrations {
				_, _ = d.db.Exec("INSERT OR IGNORE INTO schema_migrations (version) VALUES (?)", i+1)
			}
			d.log.Debug().Msg("Bootstrapped migration tracking for existing database")
			return nil
		}
	}

	for i, migration := range migrations {
		version := i + 1

		// Check if already applied
		var count int
		if err := d.db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if count > 0 {
			continue
		}

		// Apply migration
		if _, err := d.db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", version, err)
		}

		// Record version
		if _, err := d.db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			return fmt.Errorf("record migration %d: %w", version, err)
		}
	}

	d.log.Debug().Msg("Database migrations completed")
	return nil
}

// Migration SQL statements
const migrationCreateUsers = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phone VARCHAR(20) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(100),
    is_admin BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP
);
`

const migrationCreateInviteCodes = `
CREATE TABLE IF NOT EXISTS invite_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code VARCHAR(32) UNIQUE NOT NULL,
    created_by_user_id INTEGER,
    used_by_user_id INTEGER,
    used_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (used_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);
`

const migrationCreateReservedDomains = `
CREATE TABLE IF NOT EXISTS reserved_domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    subdomain VARCHAR(63) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`

const migrationCreateSessions = `
CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    refresh_token_hash VARCHAR(255) NOT NULL,
    user_agent VARCHAR(255),
    ip_address VARCHAR(45),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`

const migrationCreateAPITokens = `
CREATE TABLE IF NOT EXISTS api_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    allowed_subdomains TEXT,
    max_tunnels INTEGER DEFAULT 10,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`

const migrationCreateTOTPSecrets = `
CREATE TABLE IF NOT EXISTS totp_secrets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER UNIQUE NOT NULL,
    secret_encrypted VARCHAR(255) NOT NULL,
    is_enabled BOOLEAN DEFAULT FALSE,
    backup_codes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`

const migrationCreateAuditLogs = `
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    action VARCHAR(50) NOT NULL,
    details TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);
`

const migrationCreateIndexes = `
CREATE INDEX IF NOT EXISTS idx_reserved_domains_user ON reserved_domains(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_api_tokens_user ON api_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_invite_codes_code ON invite_codes(code);
`

const migrationCreateUserBundles = `
CREATE TABLE IF NOT EXISTS user_bundles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('http', 'tcp', 'udp')),
    local_port INTEGER NOT NULL,
    subdomain TEXT,
    remote_port INTEGER,
    auto_connect BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, name),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_user_bundles_user ON user_bundles(user_id);
`

const migrationCreateUserHistory = `
CREATE TABLE IF NOT EXISTS user_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    bundle_name TEXT,
    tunnel_type TEXT NOT NULL,
    local_port INTEGER NOT NULL,
    remote_addr TEXT,
    url TEXT,
    connected_at TIMESTAMP NOT NULL,
    disconnected_at TIMESTAMP,
    bytes_sent INTEGER DEFAULT 0,
    bytes_received INTEGER DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_user_history_user ON user_history(user_id);
CREATE INDEX IF NOT EXISTS idx_user_history_connected ON user_history(connected_at);
`

const migrationCreateUserSettings = `
CREATE TABLE IF NOT EXISTS user_settings (
    user_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(user_id, key),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`

const migrationAddAllowedIPs = `
ALTER TABLE api_tokens ADD COLUMN allowed_ips TEXT DEFAULT '[]';
`

//nolint:gosec // not credentials, just index names
const migrationAddTokenAndSessionIndexes = `
CREATE INDEX IF NOT EXISTS idx_api_tokens_token_hash ON api_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_token_hash ON sessions(refresh_token_hash);
`

const migrationCreateCustomDomains = `
CREATE TABLE IF NOT EXISTS custom_domains (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	domain TEXT UNIQUE NOT NULL,
	target_subdomain TEXT NOT NULL,
	verified BOOLEAN DEFAULT FALSE,
	verified_at TIMESTAMP,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_custom_domains_user ON custom_domains(user_id);
CREATE INDEX IF NOT EXISTS idx_custom_domains_domain ON custom_domains(domain);
`

const migrationCreateTLSCertificates = `
CREATE TABLE IF NOT EXISTS tls_certificates (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	domain TEXT UNIQUE NOT NULL,
	cert_pem BLOB NOT NULL,
	key_pem BLOB NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	issued_at TIMESTAMP NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`
