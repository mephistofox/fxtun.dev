package core

import (
	"strings"
	"testing"
)

// TestSelfUpdateRefusesUnapprovedURL: a valid signature proves a binary is
// genuine, not that it is newer. SelfUpdate must refuse a binary that no
// CheckUpdate in this process offered as an upgrade, so an old (still validly
// signed) release cannot be replayed at us.
func TestSelfUpdateRefusesUnapprovedURL(t *testing.T) {
	t.Cleanup(clearApprovedUpdate)
	clearApprovedUpdate()

	err := SelfUpdate("https://github.com/mephistofox/fxtunnel/releases/download/v0.0.1/fxtunnel-linux-amd64")
	if err == nil {
		t.Fatal("expected refusal of an update that was never offered")
	}
	if !strings.Contains(err.Error(), "not offered") {
		t.Fatalf("expected a downgrade refusal, got %v", err)
	}
}

func TestUpdateApproval(t *testing.T) {
	t.Cleanup(clearApprovedUpdate)

	const newer = "https://github.com/fx/v2"
	clearApprovedUpdate()
	if updateApproved(newer) {
		t.Fatal("nothing must be approved before a check")
	}

	approveUpdate(newer)
	if !updateApproved(newer) {
		t.Fatal("the offered upgrade must be installable")
	}
	if updateApproved("https://github.com/fx/v1") {
		t.Fatal("another binary must not ride on that approval")
	}
}
