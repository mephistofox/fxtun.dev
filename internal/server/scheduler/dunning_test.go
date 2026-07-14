package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/database"
)

// TestAutopayRetryBackoff pins the dunning retry spacing: no wait before the
// first attempt, then increasing gaps so a declining card is not charged every
// hourly tick, while still fitting several attempts inside the 7-day grace.
func TestAutopayRetryBackoff(t *testing.T) {
	cases := []struct {
		failedAttempts int
		want           time.Duration
	}{
		{0, 0},
		{1, 24 * time.Hour},
		{2, 48 * time.Hour},
		{3, 48 * time.Hour},
		{5, 48 * time.Hour},
	}
	for _, tc := range cases {
		if got := autopayRetryBackoff(tc.failedAttempts); got != tc.want {
			t.Errorf("autopayRetryBackoff(%d) = %v, want %v", tc.failedAttempts, got, tc.want)
		}
	}

	// The scheduled attempts must all land inside the grace window, otherwise a
	// card would be detached before every retry was tried.
	var elapsed time.Duration
	for n := 1; n <= 3; n++ {
		elapsed += autopayRetryBackoff(n)
	}
	if elapsed >= renewalGracePeriod {
		t.Errorf("cumulative retry backoff %v exceeds grace period %v", elapsed, renewalGracePeriod)
	}
}

// TestListFailedRecurringSince verifies the query backing dunning retry spacing
// counts only failed autopayments for this subscription within the current
// cycle: successful renewals, one-off payments, and older attempts are ignored.
func TestListFailedRecurringSince(t *testing.T) {
	db := setupTestDB(t)

	pro, err := db.Plans.GetBySlug("pro")
	if err != nil {
		t.Fatalf("pro plan: %v", err)
	}
	user := &database.User{Phone: "+79995550001", PasswordHash: "h", PlanID: pro.ID, IsActive: true}
	if err := db.Users.Create(user); err != nil {
		t.Fatalf("user: %v", err)
	}
	now := time.Now()
	start := now.Add(-30 * 24 * time.Hour)
	sub := &database.Subscription{
		UserID: user.ID, PlanID: pro.ID, Status: database.SubscriptionStatusActive,
		Recurring: true, CurrentPeriodStart: &start, CurrentPeriodEnd: &now,
	}
	if err := db.Subscriptions.Create(sub); err != nil {
		t.Fatalf("sub: %v", err)
	}

	mkPayment := func(invoice int64, status database.PaymentStatus, recurring bool) {
		p := &database.Payment{
			UserID: user.ID, SubscriptionID: &sub.ID, InvoiceID: invoice,
			Amount: 5, Status: status, IsRecurring: recurring, Provider: "yookassa",
		}
		if err := db.Payments.Create(p); err != nil {
			t.Fatalf("payment %d: %v", invoice, err)
		}
	}

	mkPayment(200001, database.PaymentStatusFailed, true)  // counts
	mkPayment(200002, database.PaymentStatusFailed, true)  // counts
	mkPayment(200003, database.PaymentStatusSuccess, true) // ignored: succeeded
	mkPayment(200004, database.PaymentStatusFailed, false) // ignored: not recurring
	mkPayment(200005, database.PaymentStatusPending, true) // ignored: still pending

	// A failed recurring attempt from a previous cycle must be excluded by `since`.
	if _, err := db.Pool().Exec(context.Background(),
		`INSERT INTO payments (user_id, subscription_id, invoice_id, amount, status, is_recurring, provider, created_at)
		 VALUES ($1, $2, $3, $4, 'failed', TRUE, 'yookassa', $5)`,
		user.ID, sub.ID, int64(200006), 5.0, now.Add(-40*24*time.Hour)); err != nil {
		t.Fatalf("insert old payment: %v", err)
	}

	got, err := db.Payments.ListFailedRecurringSince(sub.ID, start)
	if err != nil {
		t.Fatalf("ListFailedRecurringSince: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 failed recurring attempts in cycle, got %d", len(got))
	}
	// Newest first, so the two in-cycle failures are returned.
	for _, p := range got {
		if !p.IsRecurring || p.Status != database.PaymentStatusFailed {
			t.Errorf("unexpected payment in result: recurring=%v status=%s", p.IsRecurring, p.Status)
		}
	}
}
