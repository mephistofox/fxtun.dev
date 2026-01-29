package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
)

// Database holds the database connection and repositories
type Database struct {
	db           *sql.DB
	log          zerolog.Logger
	Users        *UserRepository
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

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
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

// migrate runs all database migrations
func (d *Database) migrate() error {
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
	}

	for i, migration := range migrations {
		if _, err := d.db.Exec(migration); err != nil {
			// Ignore "duplicate column" errors from ALTER TABLE migrations
			if strings.Contains(err.Error(), "duplicate column") {
				d.log.Debug().Int("migration", i+1).Msg("Migration already applied, skipping")
				continue
			}
			return fmt.Errorf("migration %d failed: %w", i+1, err)
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
