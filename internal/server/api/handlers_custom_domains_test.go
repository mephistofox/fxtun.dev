package api

import (
	"context"
	"testing"
)

// Custom domains must be unique case-insensitively. Postgres' plain UNIQUE
// constraint is case-sensitive, so an attacker could insert "EXAMPLE.com"
// alongside a victim's verified "example.com" and then delete their own row:
// the handler passes the stored spelling to RemoveCustomDomain, which
// lowercases it, evicting the victim's entry from the runtime routing map.
func TestCustomDomainUniqueIsCaseInsensitive(t *testing.T) {
	env := setupTestEnv(t)
	victim := env.createTestUser(t, "+20000000101", "password123", "Victim")
	attacker := env.createTestUser(t, "+20000000102", "password123", "Attacker")

	ctx := context.Background()
	const insert = `INSERT INTO custom_domains (user_id, domain, target_subdomain, verified)
	                VALUES ($1, $2, $3, $4)`

	if _, err := env.DB.Pool().Exec(ctx, insert, victim.User.ID, "example.com", "victimapp", true); err != nil {
		t.Fatalf("failed to insert victim domain: %v", err)
	}

	if _, err := env.DB.Pool().Exec(ctx, insert, attacker.User.ID, "EXAMPLE.com", "evilapp", false); err == nil {
		t.Fatal("expected a unique violation for a case-variant of an existing custom domain")
	}

	// A trailing dot is the same DNS name too.
	if _, err := env.DB.Pool().Exec(ctx, insert, attacker.User.ID, "example.com.", "evilapp", false); err == nil {
		t.Fatal("expected a unique violation for a trailing-dot variant; handler must normalize before insert")
	}
}
