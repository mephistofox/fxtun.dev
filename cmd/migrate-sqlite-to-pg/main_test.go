package main

import "testing"

// TestValidateFlags_DryRunWithTruncate: --dry-run must never be combined with
// --truncate-first, which would irreversibly wipe the destination.
func TestValidateFlags_DryRunWithTruncate(t *testing.T) {
	cfg := config{srcSQLite: "a.db", dstDSN: "postgres://x", dryRun: true, truncateFirst: true}
	if err := validateFlags(cfg); err == nil {
		t.Fatal("expected --dry-run together with --truncate-first to be rejected")
	}
}

func TestValidateFlags_Accepted(t *testing.T) {
	cases := []config{
		{srcSQLite: "a.db", dstDSN: "postgres://x"},
		{srcSQLite: "a.db", dstDSN: "postgres://x", dryRun: true},
		{srcSQLite: "a.db", dstDSN: "postgres://x", truncateFirst: true},
		{srcSQLite: "a.db", dstDSN: "postgres://x", validate: true},
	}
	for i, cfg := range cases {
		if err := validateFlags(cfg); err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}
	}
}

func TestValidateFlags_RequiresSourceAndDest(t *testing.T) {
	if err := validateFlags(config{dstDSN: "postgres://x"}); err == nil {
		t.Fatal("expected missing --src-sqlite to be rejected")
	}
	if err := validateFlags(config{srcSQLite: "a.db"}); err == nil {
		t.Fatal("expected missing --dst-dsn to be rejected")
	}
}
