package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/server/database"
	"github.com/mephistofox/fxtunnel/internal/server/payment"
)

// newRenewalScheduler builds a scheduler whose YooKassa provider is bound to an
// unroutable source address, so the autopayment API call fails immediately
// without touching the network. The payment record is still written first,
// which is what these tests inspect.
func newRenewalScheduler(t *testing.T, db *database.Database) *Scheduler {
	t.Helper()
	providers := payment.NewRegistry()
	providers.Register(payment.NewYooKassa(payment.YooKassaConfig{
		ShopID:    "test",
		SecretKey: "test_secret",
		TestMode:  true,
		SourceIP:  "203.0.113.1", // TEST-NET-3: never assigned locally, bind fails at once
	}))
	return New(db, &config.ServerConfig{}, providers, zerolog.New(zerolog.NewTestWriter(t)))
}

// TestRecurringRenewalPaymentHasProvider pins that the autopayment record is
// tagged with its provider. An empty provider overrides the column default and
// hides the payment from the reconciler's ListPendingByProviderInWindow sweep,
// so a renewal whose webhook is lost is never activated while the card is
// charged.
func TestRecurringRenewalPaymentHasProvider(t *testing.T) {
	db := setupTestDB(t)

	pro, err := db.Plans.GetBySlug("pro")
	if err != nil || pro == nil {
		t.Fatalf("pro plan: %v", err)
	}
	user := &database.User{Phone: "+79995550101", PasswordHash: "h", PlanID: pro.ID, IsActive: true}
	if err := db.Users.Create(user); err != nil {
		t.Fatalf("user: %v", err)
	}

	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now().Add(30 * time.Minute)
	methodID := "pm-test-1"
	sub := &database.Subscription{
		UserID: user.ID, PlanID: pro.ID, Status: database.SubscriptionStatusActive,
		Recurring: true, CurrentPeriodStart: &start, CurrentPeriodEnd: &end,
		YooKassaPaymentMethodID: &methodID,
	}
	if err := db.Subscriptions.Create(sub); err != nil {
		t.Fatalf("sub: %v", err)
	}

	newRenewalScheduler(t, db).processRecurringRenewals()

	payments, _, err := db.Payments.GetByUserID(user.ID, 10, 0)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if len(payments) != 1 {
		t.Fatalf("expected 1 renewal payment, got %d", len(payments))
	}
	if payments[0].Provider != "yookassa" {
		t.Errorf("renewal payment provider = %q, want %q", payments[0].Provider, "yookassa")
	}
}

// TestExpiredSubscriptionKeepsPaidPlanWhenNewerOneIsActive pins that expiring a
// stale subscription does not downgrade a user who already paid for a newer,
// still-running one.
func TestExpiredSubscriptionKeepsPaidPlanWhenNewerOneIsActive(t *testing.T) {
	db := setupTestDB(t)

	free, err := db.Plans.GetBySlug("free")
	if err != nil || free == nil {
		t.Fatalf("free plan: %v", err)
	}
	pro, err := db.Plans.GetBySlug("pro")
	if err != nil || pro == nil {
		t.Fatalf("pro plan: %v", err)
	}
	user := &database.User{Phone: "+79995550102", PasswordHash: "h", PlanID: pro.ID, IsActive: true}
	if err := db.Users.Create(user); err != nil {
		t.Fatalf("user: %v", err)
	}

	// Old cancelled subscription whose period has already ended.
	oldStart := time.Now().Add(-60 * 24 * time.Hour)
	oldEnd := time.Now().Add(-1 * time.Hour)
	oldSub := &database.Subscription{
		UserID: user.ID, PlanID: pro.ID, Status: database.SubscriptionStatusCancelled,
		CurrentPeriodStart: &oldStart, CurrentPeriodEnd: &oldEnd,
	}
	if err := db.Subscriptions.Create(oldSub); err != nil {
		t.Fatalf("old sub: %v", err)
	}

	// Newer, paid-for subscription that is still running.
	newStart := time.Now().Add(-1 * time.Hour)
	newEnd := time.Now().Add(29 * 24 * time.Hour)
	newSub := &database.Subscription{
		UserID: user.ID, PlanID: pro.ID, Status: database.SubscriptionStatusActive,
		CurrentPeriodStart: &newStart, CurrentPeriodEnd: &newEnd,
	}
	if err := db.Subscriptions.Create(newSub); err != nil {
		t.Fatalf("new sub: %v", err)
	}

	New(db, &config.ServerConfig{}, nil, zerolog.New(zerolog.NewTestWriter(t))).processExpiredSubscriptions()

	got, err := db.Users.GetByID(user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.PlanID != pro.ID {
		t.Errorf("user plan_id = %d (free=%d), want %d: paid subscription was downgraded", got.PlanID, free.ID, pro.ID)
	}

	stillActive, err := db.Subscriptions.GetByID(newSub.ID)
	if err != nil {
		t.Fatalf("get new sub: %v", err)
	}
	if stillActive.Status != database.SubscriptionStatusActive {
		t.Errorf("newer subscription status = %q, want %q", stillActive.Status, database.SubscriptionStatusActive)
	}
}

// TestExpiredSubscriptionDowngradesWhenNoOtherActive keeps the happy path
// honest: with nothing else to fall back on, the user still drops to free.
func TestExpiredSubscriptionDowngradesWhenNoOtherActive(t *testing.T) {
	db := setupTestDB(t)

	free, err := db.Plans.GetBySlug("free")
	if err != nil || free == nil {
		t.Fatalf("free plan: %v", err)
	}
	pro, err := db.Plans.GetBySlug("pro")
	if err != nil || pro == nil {
		t.Fatalf("pro plan: %v", err)
	}
	user := &database.User{Phone: "+79995550103", PasswordHash: "h", PlanID: pro.ID, IsActive: true}
	if err := db.Users.Create(user); err != nil {
		t.Fatalf("user: %v", err)
	}

	start := time.Now().Add(-60 * 24 * time.Hour)
	end := time.Now().Add(-1 * time.Hour)
	sub := &database.Subscription{
		UserID: user.ID, PlanID: pro.ID, Status: database.SubscriptionStatusActive,
		CurrentPeriodStart: &start, CurrentPeriodEnd: &end,
	}
	if err := db.Subscriptions.Create(sub); err != nil {
		t.Fatalf("sub: %v", err)
	}

	New(db, &config.ServerConfig{}, nil, zerolog.New(zerolog.NewTestWriter(t))).processExpiredSubscriptions()

	got, err := db.Users.GetByID(user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.PlanID != free.ID {
		t.Errorf("user plan_id = %d, want free plan %d", got.PlanID, free.ID)
	}
}

// TestStaleCleanupLeavesYooKassaConfirmationWindow pins that the stale-pending
// sweep does not fail a YooKassa payment that is still inside the provider's
// ~1h confirmation window plus the reconciler's lookback: failing it early
// turns a later "succeeded" webhook into a superseded payment (money taken,
// subscription not activated).
func TestStaleCleanupLeavesYooKassaConfirmationWindow(t *testing.T) {
	db := setupTestDB(t)

	pro, err := db.Plans.GetBySlug("pro")
	if err != nil || pro == nil {
		t.Fatalf("pro plan: %v", err)
	}
	user := &database.User{Phone: "+79995550104", PasswordHash: "h", PlanID: pro.ID, IsActive: true}
	if err := db.Users.Create(user); err != nil {
		t.Fatalf("user: %v", err)
	}

	if _, err := db.Pool().Exec(context.Background(),
		`INSERT INTO payments (user_id, invoice_id, amount, status, is_recurring, provider, created_at)
		 VALUES ($1, $2, $3, 'pending', FALSE, 'yookassa', $4)`,
		user.ID, int64(300001), 200.0, time.Now().Add(-2*time.Hour)); err != nil {
		t.Fatalf("insert pending payment: %v", err)
	}

	New(db, &config.ServerConfig{}, nil, zerolog.New(zerolog.NewTestWriter(t))).cleanupStalePendingPayments()

	got, err := db.Payments.GetByInvoiceID(300001)
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}
	if got.Status != database.PaymentStatusPending {
		t.Errorf("payment status = %q, want %q: a 2h-old YooKassa payment is still confirmable", got.Status, database.PaymentStatusPending)
	}
}
