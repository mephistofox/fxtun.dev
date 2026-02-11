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
	db            *sql.DB
	log           zerolog.Logger
	CustomDomains *CustomDomainRepository
	TLSCerts      *TLSCertRepository
	Users         *UserRepository
	Sessions      *SessionRepository
	Tokens        *APITokenRepository
	Domains       *DomainRepository
	TOTP          *TOTPRepository
	Audit         *AuditRepository
	UserBundles   *UserBundleRepository
	UserHistory   *UserHistoryRepository
	UserSettings  *UserSettingsRepository
	Plans         *PlanRepository
	Subscriptions *SubscriptionRepository
	Payments      *PaymentRepository
	Exchanges     *ExchangeRepository
}

// New creates a new database connection and initializes repositories
func New(dbPath string, log zerolog.Logger) (*Database, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
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
	database.TOTP = NewTOTPRepository(db)
	database.Audit = NewAuditRepository(db)
	database.UserBundles = NewUserBundleRepository(db)
	database.UserHistory = NewUserHistoryRepository(db)
	database.UserSettings = NewUserSettingsRepository(db)
	database.Plans = NewPlanRepository(db)
	database.Subscriptions = NewSubscriptionRepository(db)
	database.Payments = NewPaymentRepository(db)
	database.Exchanges = NewExchangeRepository(db)

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
		migrationAddOAuthFields,
		migrationAddGoogleOAuth,
		migrationMakePhoneNullable,
		migrationCreatePlans,
		migrationAddPlanVisibility,
		migrationCreateSubscriptions,
		migrationCreatePayments,
		migrationRenameToYooKassa,
		migrationCreateInspectExchanges,
		migrationAddInspectHostUserIndex,
		migrationAddPlanBandwidth,
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

		// Apply migration inside a transaction
		tx, err := d.db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration tx: %w", err)
		}
		if _, err := tx.Exec(migration); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d failed: %w", version, err)
		}
		// Record version inside same transaction
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
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

const migrationAddOAuthFields = `
ALTER TABLE users ADD COLUMN github_id INTEGER;
ALTER TABLE users ADD COLUMN email VARCHAR(255);
ALTER TABLE users ADD COLUMN avatar_url TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_github_id ON users(github_id) WHERE github_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL;
`

const migrationAddGoogleOAuth = `
ALTER TABLE users ADD COLUMN google_id VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL;
`

const migrationMakePhoneNullable = `
-- Recreate users table with phone nullable for OAuth users
CREATE TABLE users_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phone VARCHAR(20),
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    display_name VARCHAR(100),
    is_admin BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    github_id INTEGER,
    email VARCHAR(255),
    avatar_url TEXT,
    google_id VARCHAR(255)
);
INSERT INTO users_new SELECT id, CASE WHEN phone = '' THEN NULL ELSE phone END, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone ON users(phone) WHERE phone IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_github_id ON users(github_id) WHERE github_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL AND email != '';
`

const migrationCreatePlans = `
CREATE TABLE plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    price REAL NOT NULL DEFAULT 0,
    max_tunnels INTEGER NOT NULL DEFAULT 3,
    max_domains INTEGER NOT NULL DEFAULT 1,
    max_custom_domains INTEGER NOT NULL DEFAULT 0,
    max_tokens INTEGER NOT NULL DEFAULT 1,
    max_tunnels_per_token INTEGER NOT NULL DEFAULT 3,
    inspector_enabled BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO plans (slug, name, price, max_tunnels, max_domains, max_custom_domains, max_tokens, max_tunnels_per_token, inspector_enabled) VALUES
    ('free', 'Free', 0, 3, 1, 0, 1, 3, 0),
    ('base', 'Base', 5, 5, 5, 1, 5, 5, 1),
    ('pro', 'Pro', 10, 15, 15, 5, 10, 10, 1),
    ('business', 'Business', 20, 50, 50, 50, 50, 50, 1),
    ('admin', 'Admin', 0, -1, -1, -1, -1, -1, 1);

ALTER TABLE users ADD COLUMN plan_id INTEGER REFERENCES plans(id);
UPDATE users SET plan_id = (SELECT id FROM plans WHERE slug = 'admin') WHERE is_admin = 1;
UPDATE users SET plan_id = (SELECT id FROM plans WHERE slug = 'free') WHERE plan_id IS NULL;
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

const migrationAddPlanVisibility = `
ALTER TABLE plans ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE plans ADD COLUMN is_recommended BOOLEAN NOT NULL DEFAULT 0;
`

const migrationCreateSubscriptions = `
CREATE TABLE subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    plan_id INTEGER NOT NULL,
    next_plan_id INTEGER,
    status TEXT NOT NULL DEFAULT 'pending',
    recurring BOOLEAN NOT NULL DEFAULT 1,
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    robokassa_invoice_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (plan_id) REFERENCES plans(id),
    FOREIGN KEY (next_plan_id) REFERENCES plans(id)
);
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_period_end ON subscriptions(current_period_end);
`

const migrationCreatePayments = `
CREATE TABLE payments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    subscription_id INTEGER,
    invoice_id INTEGER NOT NULL UNIQUE,
    amount REAL NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    is_recurring BOOLEAN NOT NULL DEFAULT 0,
    robokassa_data TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE SET NULL
);
CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_invoice_id ON payments(invoice_id);
CREATE INDEX idx_payments_status ON payments(status);
`

const migrationRenameToYooKassa = `
-- Add new YooKassa columns to subscriptions
ALTER TABLE subscriptions ADD COLUMN yookassa_payment_method_id TEXT;

-- Add new YooKassa columns to payments (rename robokassa_data to yookassa_data)
ALTER TABLE payments ADD COLUMN yookassa_data TEXT;
-- Copy data from old column to new
UPDATE payments SET yookassa_data = robokassa_data WHERE robokassa_data IS NOT NULL;
`

const migrationCreateInspectExchanges = `
CREATE TABLE IF NOT EXISTS inspect_exchanges (
    id TEXT PRIMARY KEY,
    tunnel_id TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    trace_id TEXT,
    replay_ref TEXT,
    timestamp DATETIME NOT NULL,
    duration_ns INTEGER NOT NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    host TEXT NOT NULL,
    request_headers TEXT,
    request_body BLOB,
    request_body_size INTEGER NOT NULL DEFAULT 0,
    response_headers TEXT,
    response_body BLOB,
    response_body_size INTEGER NOT NULL DEFAULT 0,
    status_code INTEGER NOT NULL,
    remote_addr TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_inspect_exch_tunnel ON inspect_exchanges(tunnel_id, timestamp DESC);
CREATE INDEX idx_inspect_exch_created ON inspect_exchanges(created_at);
`

const migrationAddInspectHostUserIndex = `
CREATE INDEX IF NOT EXISTS idx_inspect_exch_host_user ON inspect_exchanges(host, user_id, timestamp DESC);
`

const migrationAddPlanBandwidth = `
ALTER TABLE plans ADD COLUMN bandwidth_mbps INTEGER NOT NULL DEFAULT 0;
UPDATE plans SET bandwidth_mbps = 10 WHERE slug = 'free';
UPDATE plans SET bandwidth_mbps = 50 WHERE slug = 'base';
UPDATE plans SET bandwidth_mbps = 100 WHERE slug = 'pro';
UPDATE plans SET bandwidth_mbps = 0 WHERE slug = 'admin';
`
