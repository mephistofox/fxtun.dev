// One-shot utility for migrating data from the legacy SQLite schema (old prod)
// into the new PostgreSQL schema (develop branch).
//
// Usage:
//   migrate-sqlite-to-pg \
//     --src-sqlite=/path/to/fxtunnel.db \
//     --dst-dsn='postgres://user:pass@127.0.0.1:5432/fxtunnel?sslmode=disable' \
//     [--dry-run] [--truncate-first] [--validate] [--skip=inspect_exchanges,audit_logs]
//
// The PG schema must already be created (goose up of internal/server/database/migrations).
// Plans seed (00002) must already be applied — we map plans by slug, not id.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	srcSQLite      string
	dstDSN         string
	dryRun         bool
	truncateFirst  bool
	validate       bool
	skip           map[string]bool
}

func main() {
	cfg := parseFlags()

	if cfg.srcSQLite == "" || cfg.dstDSN == "" {
		log.Fatal("--src-sqlite and --dst-dsn are required")
	}

	ctx := context.Background()

	// Open SQLite read-only
	src, err := sql.Open("sqlite3", "file:"+cfg.srcSQLite+"?mode=ro&_query_only=true")
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer src.Close()
	if err := src.PingContext(ctx); err != nil {
		log.Fatalf("ping sqlite: %v", err)
	}

	// Open PG
	dst, err := pgx.Connect(ctx, cfg.dstDSN)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer dst.Close(ctx)

	if cfg.validate {
		if err := runValidate(ctx, src, dst); err != nil {
			log.Fatalf("validate failed: %v", err)
		}
		return
	}

	if cfg.truncateFirst {
		if err := truncateAll(ctx, dst); err != nil {
			log.Fatalf("truncate: %v", err)
		}
	}

	if cfg.dryRun {
		if err := runDryRun(ctx, src); err != nil {
			log.Fatalf("dry-run: %v", err)
		}
		return
	}

	if err := runMigration(ctx, src, dst, cfg.skip); err != nil {
		log.Fatalf("migration: %v", err)
	}

	fmt.Println("\nMigration completed successfully.")
}

func parseFlags() config {
	var (
		srcSQLite     = flag.String("src-sqlite", "", "path to SQLite database file")
		dstDSN        = flag.String("dst-dsn", "", "Postgres DSN (e.g. postgres://user:pass@host:port/db?sslmode=disable)")
		dryRun        = flag.Bool("dry-run", false, "count rows in source, do not write to destination")
		truncateFirst = flag.Bool("truncate-first", false, "TRUNCATE all tables before importing")
		validate      = flag.Bool("validate", false, "compare row counts between source and destination")
		skip          = flag.String("skip", "", "comma-separated tables to skip (e.g. inspect_exchanges,audit_logs)")
	)
	flag.Parse()

	skipMap := map[string]bool{}
	if *skip != "" {
		for _, t := range strings.Split(*skip, ",") {
			skipMap[strings.TrimSpace(t)] = true
		}
	}

	return config{
		srcSQLite:     *srcSQLite,
		dstDSN:        *dstDSN,
		dryRun:        *dryRun,
		truncateFirst: *truncateFirst,
		validate:      *validate,
		skip:          skipMap,
	}
}

func runDryRun(ctx context.Context, src *sql.DB) error {
	fmt.Println("=== dry-run: source row counts ===")
	for _, spec := range allTables() {
		var n int
		if err := src.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+spec.name).Scan(&n); err != nil {
			fmt.Printf("  %-25s ERROR: %v\n", spec.name, err)
			continue
		}
		fmt.Printf("  %-25s %d\n", spec.name, n)
	}
	return nil
}

func runValidate(ctx context.Context, src *sql.DB, dst *pgx.Conn) error {
	fmt.Println("=== validate: row count diff (sqlite vs pg) ===")
	hasMismatch := false
	for _, spec := range allTables() {
		var srcN, dstN int
		if err := src.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+spec.name).Scan(&srcN); err != nil {
			fmt.Printf("  %-25s SQLite ERROR: %v\n", spec.name, err)
			continue
		}
		if err := dst.QueryRow(ctx, "SELECT COUNT(*) FROM "+spec.name).Scan(&dstN); err != nil {
			fmt.Printf("  %-25s PG ERROR: %v\n", spec.name, err)
			continue
		}
		marker := "✓"
		if srcN != dstN {
			marker = "✗"
			hasMismatch = true
		}
		fmt.Printf("  %s %-25s sqlite=%-8d pg=%-8d diff=%d\n", marker, spec.name, srcN, dstN, dstN-srcN)
	}
	if hasMismatch {
		return errors.New("row count mismatch — see above")
	}
	fmt.Println("All tables match.")
	return nil
}

func truncateAll(ctx context.Context, dst *pgx.Conn) error {
	fmt.Println("=== TRUNCATE ALL TABLES ===")
	// Reverse FK order: payments first, then subscriptions, ..., finally plans (kept seeded)
	tables := []string{
		"inspect_exchanges", "payments", "subscriptions",
		"tls_certificates", "custom_domains", "user_settings", "user_history",
		"user_bundles", "audit_logs", "totp_secrets", "api_tokens", "sessions",
		"reserved_domains", "invite_codes", "users",
	}
	stmt := "TRUNCATE TABLE " + strings.Join(tables, ", ") + " RESTART IDENTITY CASCADE"
	if _, err := dst.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("truncate: %w", err)
	}
	// Plans we don't truncate — they were seeded by goose 00002 and are FK targets.
	fmt.Println("  (plans table kept — seeded by goose migration)")
	return nil
}

func runMigration(ctx context.Context, src *sql.DB, dst *pgx.Conn, skip map[string]bool) error {
	tx, err := dst.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no-op if committed

	// 1. Plans first — UPSERT by slug, build slug→pg-id map.
	planMap, err := migratePlans(ctx, src, tx)
	if err != nil {
		return fmt.Errorf("plans: %w", err)
	}
	fmt.Printf("  plans: %d slugs mapped\n", len(planMap))

	// 2..n: ordered tables
	for _, spec := range allTables() {
		if spec.name == "plans" {
			continue // handled above
		}
		if skip[spec.name] {
			fmt.Printf("  %-25s SKIPPED\n", spec.name)
			continue
		}
		start := time.Now()
		n, err := migrateTable(ctx, src, tx, spec, planMap)
		if err != nil {
			return fmt.Errorf("table %s: %w", spec.name, err)
		}
		fmt.Printf("  %-25s %d rows in %s\n", spec.name, n, time.Since(start).Round(time.Millisecond))
	}

	// 3. Bump sequences to MAX(id)+1 so future inserts don't collide.
	if err := bumpSequences(ctx, tx); err != nil {
		return fmt.Errorf("bump sequences: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// bumpSequences resets every BIGSERIAL sequence so the next INSERT starts above
// the max id we imported. Without this, the next user/api_token/etc. created via
// the app would collide with imported ids.
func bumpSequences(ctx context.Context, tx pgx.Tx) error {
	seqTables := []string{
		"users", "invite_codes", "reserved_domains", "sessions", "api_tokens",
		"totp_secrets", "audit_logs", "user_bundles", "user_history",
		"custom_domains", "tls_certificates", "subscriptions", "payments",
	}
	for _, t := range seqTables {
		stmt := fmt.Sprintf(
			"SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE((SELECT MAX(id) FROM %s), 0) + 1, false)",
			t, t,
		)
		var seq int64
		if err := tx.QueryRow(ctx, stmt).Scan(&seq); err != nil {
			return fmt.Errorf("setval %s: %w", t, err)
		}
	}
	// Also plans sequence — though we map by slug, the goose seed used inserts.
	stmt := "SELECT setval(pg_get_serial_sequence('plans', 'id'), COALESCE((SELECT MAX(id) FROM plans), 0) + 1, false)"
	var seq int64
	if err := tx.QueryRow(ctx, stmt).Scan(&seq); err != nil {
		return fmt.Errorf("setval plans: %w", err)
	}
	return nil
}

