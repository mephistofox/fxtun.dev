package main

import (
	"bytes"
	"strings"
	"testing"
)

// SilenceUsage must suppress the full help text on an error returned from
// RunE (a validation failure here), while a cobra flag-parse error still
// gets it — the flag was typed wrong, not the command's logic.
func TestRootCmd_SilenceUsageOnRunEError(t *testing.T) {
	resetGlobals(t)
	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"http", "3000", "--auth", "bad"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected error from invalid --auth")
	}
	if strings.Contains(buf.String(), "Usage:") {
		t.Fatalf("usage printed despite SilenceUsage: %s", buf.String())
	}
}

func TestRootCmd_UsageShownOnFlagParseError(t *testing.T) {
	resetGlobals(t)
	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"http", "3000", "--no-such-flag"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected error from unknown flag")
	}
	if !strings.Contains(buf.String(), "Usage:") {
		t.Fatalf("usage missing on flag-parse error: %s", buf.String())
	}
}
