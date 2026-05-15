package main

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

// parseSQLiteTime parses SQLite's TIMESTAMP/DATETIME text format into a time.Time
// suitable for PG TIMESTAMPTZ. SQLite typically stores "YYYY-MM-DD HH:MM:SS" in
// UTC. Returns zero time + nil error for NULL / empty.
func parseSQLiteTime(ns sql.NullString) (any, error) {
	if !ns.Valid || ns.String == "" {
		return nil, nil
	}
	s := strings.TrimSpace(ns.String)
	// Try a few common SQLite formats.
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	// Last-resort: store as is — driver will reject if invalid, surfacing the issue early.
	return s, nil
}

// boolFromInt converts SQLite's INTEGER 0/1 to bool.
func boolFromInt(n sql.NullInt64) bool {
	return n.Valid && n.Int64 != 0
}

// nullableString returns nil for NULL / empty strings — important for
// PG partial unique indexes (e.g. idx_users_phone WHERE phone IS NOT NULL).
func nullableString(ns sql.NullString) any {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	return ns.String
}

// nullableStringKeepEmpty returns the string as-is for VARCHAR columns where
// empty string != NULL is meaningful (e.g. password_hash DEFAULT '').
func nullableStringKeepEmpty(ns sql.NullString) any {
	if !ns.Valid {
		return nil
	}
	return ns.String
}

// nullableInt64 returns nil for NULL, otherwise the int64.
func nullableInt64(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

// jsonbOrDefault validates that a TEXT column holds parseable JSON and returns
// it as a JSON string suitable for PG's JSONB column. NULL/empty → defaultJSON
// (typically "[]" or "{}" — caller decides). Invalid JSON is replaced with
// defaultJSON to avoid breaking the import.
func jsonbOrDefault(ns sql.NullString, defaultJSON string) any {
	if !ns.Valid || strings.TrimSpace(ns.String) == "" {
		if defaultJSON == "" {
			return nil
		}
		return defaultJSON
	}
	if !json.Valid([]byte(ns.String)) {
		if defaultJSON == "" {
			return nil
		}
		return defaultJSON
	}
	return ns.String
}

// jsonbNullable like jsonbOrDefault but returns nil for invalid/empty input.
// Use this for nullable JSONB columns (e.g. audit_logs.details, totp_secrets.backup_codes).
func jsonbNullable(ns sql.NullString) any {
	return jsonbOrDefault(ns, "")
}

// blobOrNil returns nil for empty BLOBs, otherwise the bytes (for PG BYTEA).
func blobOrNil(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
