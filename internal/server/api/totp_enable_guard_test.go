package api

import (
	"testing"
)

// Re-running the TOTP setup endpoint must not silently switch 2FA off.
// Before the guard, EnableTOTP overwrote the row with is_enabled = false
// without asking for a code, so anyone holding a stolen access token or an
// sk_ API token could strip the victim's second factor (and wipe their
// backup codes) with a single unauthenticated-by-2FA call.
func TestEnableTOTP_KeepsExisting2FAEnabled(t *testing.T) {
	env := setupTestEnv(t)
	u := env.createTestUser(t, "+70000009001", "password123", "totp guard")

	if _, _, _, err := env.AuthService.EnableTOTP(u.User.ID, u.User.Phone); err != nil {
		t.Fatalf("initial EnableTOTP: %v", err)
	}
	if err := env.DB.TOTP.Enable(u.User.ID); err != nil {
		t.Fatalf("enable 2FA: %v", err)
	}

	_, _, _, err := env.AuthService.EnableTOTP(u.User.ID, u.User.Phone)
	if err == nil {
		t.Error("EnableTOTP on an account with active 2FA returned no error; it must refuse")
	}

	enabled, err := env.DB.TOTP.IsEnabled(u.User.ID)
	if err != nil {
		t.Fatalf("IsEnabled: %v", err)
	}
	if !enabled {
		t.Error("2FA was switched off by a plain EnableTOTP call — account takeover protection lost")
	}
}
