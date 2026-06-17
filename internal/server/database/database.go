package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/server/database/sqlc"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Database holds the database connection pool and repositories.
type Database struct {
	pool          *pgxpool.Pool
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
	EdgeNodes     *EdgeNodeRepository
	InviteCodes   *InviteCodeRepository
}

// New creates a new PostgreSQL database connection pool and initializes repositories.
func New(dsn string, log zerolog.Logger) (*Database, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Run goose migrations
	if err := runMigrations(dsn); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	q := sqlc.New(pool)
	lg := log.With().Str("component", "database").Logger()

	database := &Database{
		pool:          pool,
		log:           lg,
		CustomDomains: &CustomDomainRepository{q: q},
		TLSCerts:      &TLSCertRepository{q: q},
		Users:         &UserRepository{q: q, pool: pool},
		Sessions:      &SessionRepository{q: q},
		Tokens:        &APITokenRepository{q: q},
		Domains:       &DomainRepository{q: q},
		TOTP:          &TOTPRepository{q: q},
		Audit:         &AuditRepository{q: q},
		UserBundles:   &UserBundleRepository{q: q},
		UserHistory:   &UserHistoryRepository{q: q},
		UserSettings:  &UserSettingsRepository{q: q},
		Plans:         &PlanRepository{q: q},
		Subscriptions: &SubscriptionRepository{q: q},
		Payments:      &PaymentRepository{q: q, pool: pool},
		Exchanges:     &ExchangeRepository{q: q},
		EdgeNodes:     &EdgeNodeRepository{pool: pool},
		InviteCodes:   &InviteCodeRepository{pool: pool},
	}

	lg.Info().Msg("Database initialized")
	return database, nil
}

// Close closes the database connection pool.
func (d *Database) Close() error {
	d.pool.Close()
	return nil
}

// Pool returns the underlying pgxpool.Pool for direct access (e.g. transactions).
func (d *Database) Pool() *pgxpool.Pool {
	return d.pool
}

// runMigrations uses goose to apply embedded SQL migrations.
func runMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open for migrations: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
