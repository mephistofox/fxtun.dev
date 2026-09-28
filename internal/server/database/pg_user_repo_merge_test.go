package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// newTestDB spins up an isolated schema on TEST_DATABASE_DSN and returns a
// migrated Database. Skips when TEST_DATABASE_DSN is unset.
func newTestDB(t *testing.T) *Database {
	t.Helper()

	baseDSN := os.Getenv("TEST_DATABASE_DSN")
	if baseDSN == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping database-dependent test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, baseDSN)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %q", schema)); err != nil {
		pool.Close()
		t.Fatalf("create schema: %v", err)
	}
	pool.Close()

	t.Cleanup(func() {
		p, err := pgxpool.New(ctx, baseDSN)
		if err == nil {
			_, _ = p.Exec(ctx, fmt.Sprintf("DROP SCHEMA %q CASCADE", schema))
			p.Close()
		}
	})

	sep := "?"
	if strings.Contains(baseDSN, "?") {
		sep = "&"
	}
	db, err := New(baseDSN+sep+"search_path="+schema, zerolog.New(os.Stderr).Level(zerolog.Disabled))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func insertUser(t *testing.T, db *Database, phone string) int64 {
	t.Helper()
	var id int64
	err := db.Pool().QueryRow(context.Background(),
		`INSERT INTO users (phone, password_hash) VALUES ($1, 'x') RETURNING id`, phone).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

// TestMergeUsers_BothHaveOAuthAndTOTP reproduces the admin merge failure:
// copying secondary's OAuth ids into primary before deleting secondary trips the
// partial UNIQUE indexes, and moving totp_secrets trips UNIQUE(user_id).
func TestMergeUsers_BothHaveOAuthAndTOTP(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	primary := insertUser(t, db, "+30000000001")
	secondary := insertUser(t, db, "+30000000002")

	// Primary has no OAuth links; secondary has all three.
	_, err := db.Pool().Exec(ctx,
		`UPDATE users SET github_id = 111, google_id = 'g-111', yandex_id = 'y-111',
		 email = 'secondary@example.com', avatar_url = 'http://a/1' WHERE id = $1`, secondary)
	if err != nil {
		t.Fatalf("seed secondary oauth: %v", err)
	}

	// Both have 2FA enabled — totp_secrets has UNIQUE(user_id).
	for _, id := range []int64{primary, secondary} {
		_, err := db.Pool().Exec(ctx,
			`INSERT INTO totp_secrets (user_id, secret_encrypted, is_enabled) VALUES ($1, 'sec', TRUE)`, id)
		if err != nil {
			t.Fatalf("seed totp: %v", err)
		}
	}

	if err := db.Users.MergeUsers(primary, secondary); err != nil {
		t.Fatalf("MergeUsers failed: %v", err)
	}

	var githubID *int64
	var googleID, yandexID, email *string
	err = db.Pool().QueryRow(ctx,
		`SELECT github_id, google_id, yandex_id, email FROM users WHERE id = $1`, primary).
		Scan(&githubID, &googleID, &yandexID, &email)
	if err != nil {
		t.Fatalf("read primary: %v", err)
	}
	if githubID == nil || *githubID != 111 {
		t.Fatalf("github_id not merged: %v", githubID)
	}
	if googleID == nil || *googleID != "g-111" {
		t.Fatalf("google_id not merged: %v", googleID)
	}
	if yandexID == nil || *yandexID != "y-111" {
		t.Fatalf("yandex_id not merged: %v", yandexID)
	}

	var remaining int
	if err := db.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE id = $1`, secondary).Scan(&remaining); err != nil {
		t.Fatalf("count secondary: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("secondary user was not deleted")
	}

	var totpRows int
	if err := db.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM totp_secrets`).Scan(&totpRows); err != nil {
		t.Fatalf("count totp: %v", err)
	}
	if totpRows != 1 {
		t.Fatalf("expected exactly one surviving totp secret, got %d", totpRows)
	}
}

// TestListWithSort_IncludesYandexID: sorted admin listing must not blank out the
// Yandex link.
func TestListWithSort_IncludesYandexID(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	id := insertUser(t, db, "+30000000003")
	if _, err := db.Pool().Exec(ctx, `UPDATE users SET yandex_id = 'y-sort' WHERE id = $1`, id); err != nil {
		t.Fatalf("seed yandex_id: %v", err)
	}

	users, _, err := db.Users.List(UserListParams{Limit: 10, Offset: 0, SortBy: "created_at", Order: "desc"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, u := range users {
		if u.ID == id {
			if u.YandexID == nil || *u.YandexID != "y-sort" {
				t.Fatalf("sorted listing dropped yandex_id: %v", u.YandexID)
			}
			return
		}
	}
	t.Fatalf("user %d not found in sorted listing", id)
}

// TestCreateWithLimit_Concurrent: the per-user row lock must serialize the
// count+insert so a plan limit cannot be overrun by parallel callers.
func TestCreateWithLimit_Concurrent(t *testing.T) {
	db := newTestDB(t)
	userID := insertUser(t, db, "+30000000004")

	const n = 20
	var wg sync.WaitGroup
	start := make(chan struct{})
	okCount := make([]bool, n)

	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := db.Domains.CreateWithLimit(&ReservedDomain{
				UserID:    userID,
				Subdomain: fmt.Sprintf("conc%d", i),
			}, 1)
			okCount[i] = err == nil
		}()
	}
	close(start)
	wg.Wait()

	created := 0
	for _, ok := range okCount {
		if ok {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("expected exactly 1 successful reservation, got %d", created)
	}

	var rows int
	if err := db.Pool().QueryRow(context.Background(),
		`SELECT COUNT(*) FROM reserved_domains WHERE user_id = $1`, userID).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 reserved_domains row, got %d", rows)
	}
}

// insertSubscription creates a subscription row directly and returns its id.
func insertSubscription(t *testing.T, db *Database, userID int64, status string) int64 {
	t.Helper()
	var id int64
	err := db.Pool().QueryRow(context.Background(),
		`INSERT INTO subscriptions (user_id, plan_id, status, recurring, current_period_end)
		 VALUES ($1, 2, $2, TRUE, NOW() + INTERVAL '10 days') RETURNING id`, userID, status).Scan(&id)
	if err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
	return id
}

// insertPayment creates a payment row directly and returns its id.
func insertPayment(t *testing.T, db *Database, userID, subID, invoiceID int64) int64 {
	t.Helper()
	var id int64
	err := db.Pool().QueryRow(context.Background(),
		`INSERT INTO payments (user_id, subscription_id, invoice_id, amount, status, provider)
		 VALUES ($1, $2, $3, 250, 'success', 'yookassa') RETURNING id`, userID, subID, invoiceID).Scan(&id)
	if err != nil {
		t.Fatalf("insert payment: %v", err)
	}
	return id
}

// TestMergeUsers_TransfersPaymentsAndSubscriptions: payments and subscriptions
// cascade on user delete, so a merge that leaves them behind silently destroys
// the secondary account's financial history along with its live subscription.
func TestMergeUsers_TransfersPaymentsAndSubscriptions(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	primary := insertUser(t, db, "+30000000011")
	secondary := insertUser(t, db, "+30000000012")

	subID := insertSubscription(t, db, secondary, "active")
	payID := insertPayment(t, db, secondary, subID, 900011)

	if err := db.Users.MergeUsers(primary, secondary); err != nil {
		t.Fatalf("MergeUsers failed: %v", err)
	}

	var subOwner int64
	if err := db.Pool().QueryRow(ctx, `SELECT user_id FROM subscriptions WHERE id = $1`, subID).Scan(&subOwner); err != nil {
		t.Fatalf("subscription lost by merge: %v", err)
	}
	if subOwner != primary {
		t.Errorf("subscription owner = %d, want %d", subOwner, primary)
	}

	var payOwner int64
	if err := db.Pool().QueryRow(ctx, `SELECT user_id FROM payments WHERE id = $1`, payID).Scan(&payOwner); err != nil {
		t.Fatalf("payment lost by merge: %v", err)
	}
	if payOwner != primary {
		t.Errorf("payment owner = %d, want %d", payOwner, primary)
	}
}

// TestMergeUsers_KeepsPrimaryActiveSubscription: when both accounts carry a live
// subscription the primary's stays the effective one; the secondary's row is
// preserved for history but must not stay live, or renewals bill twice.
func TestMergeUsers_KeepsPrimaryActiveSubscription(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	primary := insertUser(t, db, "+30000000013")
	secondary := insertUser(t, db, "+30000000014")

	primarySub := insertSubscription(t, db, primary, "active")
	secondarySub := insertSubscription(t, db, secondary, "active")

	if err := db.Users.MergeUsers(primary, secondary); err != nil {
		t.Fatalf("MergeUsers failed: %v", err)
	}

	var primaryStatus, secondaryStatus string
	var secondaryOwner int64
	if err := db.Pool().QueryRow(ctx, `SELECT status FROM subscriptions WHERE id = $1`, primarySub).Scan(&primaryStatus); err != nil {
		t.Fatalf("read primary subscription: %v", err)
	}
	if err := db.Pool().QueryRow(ctx, `SELECT status, user_id FROM subscriptions WHERE id = $1`, secondarySub).Scan(&secondaryStatus, &secondaryOwner); err != nil {
		t.Fatalf("secondary subscription lost by merge: %v", err)
	}

	if primaryStatus != "active" {
		t.Errorf("primary subscription status = %q, want active", primaryStatus)
	}
	if secondaryOwner != primary {
		t.Errorf("secondary subscription owner = %d, want %d", secondaryOwner, primary)
	}
	if secondaryStatus == "active" {
		t.Error("both subscriptions are active after merge: the user would be billed twice")
	}
}

// TestMergeUsers_PendingSubscriptionsDoNotCollide: the partial unique index
// allows one pending subscription per user, so transferring the secondary's
// pending checkout onto a primary that already has one must not fail the merge.
func TestMergeUsers_PendingSubscriptionsDoNotCollide(t *testing.T) {
	db := newTestDB(t)

	primary := insertUser(t, db, "+30000000015")
	secondary := insertUser(t, db, "+30000000016")

	insertSubscription(t, db, primary, "pending")
	insertSubscription(t, db, secondary, "pending")

	if err := db.Users.MergeUsers(primary, secondary); err != nil {
		t.Fatalf("MergeUsers failed: %v", err)
	}
}
