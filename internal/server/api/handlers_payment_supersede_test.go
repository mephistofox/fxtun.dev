package api

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/mephistofox/fxtun.dev/internal/server/database"
	"github.com/mephistofox/fxtun.dev/internal/server/payment"
)

// newPendingCheckout creates a pending subscription + pending payment for a user,
// mirroring what handleCheckout persists, and returns them.
func newPendingCheckout(t *testing.T, env *testEnv, userID, invoiceID int64) (*database.Subscription, *database.Payment) {
	t.Helper()

	sub := &database.Subscription{
		UserID:    userID,
		PlanID:    1, // seeded "free" plan — only needs to satisfy the FK
		Status:    database.SubscriptionStatusPending,
		Recurring: false,
	}
	if err := env.DB.Subscriptions.Create(sub); err != nil {
		t.Fatalf("create pending subscription: %v", err)
	}

	pd, _ := json.Marshal(map[string]string{"provider_payment_id": "yoo-test-1"})
	pmt := &database.Payment{
		UserID:         userID,
		SubscriptionID: &sub.ID,
		InvoiceID:      invoiceID,
		Amount:         200,
		Status:         database.PaymentStatusPending,
		Provider:       "yookassa",
		ProviderData:   string(pd),
	}
	if err := env.DB.Payments.Create(pmt); err != nil {
		t.Fatalf("create pending payment: %v", err)
	}

	return sub, pmt
}

// TestSupersedePendingCheckout verifies that superseding an abandoned checkout
// expires the pending subscription and fails its pending payment, so a fresh
// checkout is no longer blocked by "pending payment already exists".
func TestSupersedePendingCheckout(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+30000000001", "password123", "Pay User")

	sub, _ := newPendingCheckout(t, env, user.User.ID, 999001)

	if err := env.APIServer.supersedePendingCheckout(sub); err != nil {
		t.Fatalf("supersedePendingCheckout: %v", err)
	}

	gotSub, err := env.DB.Subscriptions.GetByID(sub.ID)
	if err != nil {
		t.Fatalf("get subscription: %v", err)
	}
	if gotSub.Status != database.SubscriptionStatusExpired {
		t.Errorf("subscription status = %q, want %q", gotSub.Status, database.SubscriptionStatusExpired)
	}

	gotPmt, err := env.DB.Payments.GetByInvoiceID(999001)
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}
	if gotPmt.Status != database.PaymentStatusFailed {
		t.Errorf("payment status = %q, want %q", gotPmt.Status, database.PaymentStatusFailed)
	}

	// The user must now be free to start a new checkout.
	pending, err := env.DB.Subscriptions.GetPendingByUserID(user.User.ID)
	if err != nil {
		t.Fatalf("get pending subscription: %v", err)
	}
	if pending != nil {
		t.Errorf("expected no pending subscription after supersede, got id=%d", pending.ID)
	}
}

// TestPendingSubscriptionUniquePerUser guards the TOCTOU fix: the DB enforces at
// most one pending subscription per user, so two concurrent checkouts cannot each
// create a pending subscription (and a live provider payment) for the same user.
func TestPendingSubscriptionUniquePerUser(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+30000000003", "password123", "Pay User 3")

	first := &database.Subscription{
		UserID:    user.User.ID,
		PlanID:    1,
		Status:    database.SubscriptionStatusPending,
		Recurring: false,
	}
	if err := env.DB.Subscriptions.Create(first); err != nil {
		t.Fatalf("create first pending subscription: %v", err)
	}

	// A second pending subscription for the same user must be rejected with the
	// typed sentinel so the handler can translate it into a 409 retry.
	second := &database.Subscription{
		UserID:    user.User.ID,
		PlanID:    1,
		Status:    database.SubscriptionStatusPending,
		Recurring: false,
	}
	err := env.DB.Subscriptions.Create(second)
	if !errors.Is(err, database.ErrPendingSubscriptionExists) {
		t.Fatalf("second pending subscription: got err %v, want ErrPendingSubscriptionExists", err)
	}

	// A non-pending subscription (e.g. active) is unaffected by the partial index.
	active := &database.Subscription{
		UserID:    user.User.ID,
		PlanID:    1,
		Status:    database.SubscriptionStatusActive,
		Recurring: false,
	}
	if err := env.DB.Subscriptions.Create(active); err != nil {
		t.Fatalf("create active subscription alongside pending: %v", err)
	}
}

// TestHandlePaymentSucceeded_IgnoresSuperseded is the double-charge guard: a
// succeeded webhook for a payment that was superseded (marked failed) must NOT
// activate its subscription. Otherwise a user who abandoned a checkout, started
// a new one, then completed the old provider page would be charged twice and get
// two active subscriptions.
func TestHandlePaymentSucceeded_IgnoresSuperseded(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+30000000002", "password123", "Pay User 2")

	sub, _ := newPendingCheckout(t, env, user.User.ID, 999002)
	// Simulate that this checkout was superseded by a newer one.
	if err := env.APIServer.supersedePendingCheckout(sub); err != nil {
		t.Fatalf("supersedePendingCheckout: %v", err)
	}

	// The abandoned YooKassa payment is completed anyway and fires a webhook.
	yooPayment := &payment.Payment{
		ID:     "yoo-test-1",
		Status: "succeeded",
		Amount: payment.Amount{Value: "200.00", Currency: "RUB"},
		Metadata: map[string]string{
			"invoice_id":      "999002",
			"user_id":         strconv.FormatInt(user.User.ID, 10),
			"subscription_id": strconv.FormatInt(sub.ID, 10),
			"plan_id":         "1",
		},
	}

	w := httptest.NewRecorder()
	env.APIServer.handlePaymentSucceeded(w, yooPayment)

	// Subscription must stay expired (not reactivated).
	gotSub, err := env.DB.Subscriptions.GetByID(sub.ID)
	if err != nil {
		t.Fatalf("get subscription: %v", err)
	}
	if gotSub.Status == database.SubscriptionStatusActive {
		t.Errorf("superseded subscription was reactivated: status=%q", gotSub.Status)
	}

	// Payment must stay failed (not flipped to success).
	gotPmt, err := env.DB.Payments.GetByInvoiceID(999002)
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}
	if gotPmt.Status == database.PaymentStatusSuccess {
		t.Errorf("superseded payment was marked success")
	}

	// User plan must be unchanged (still the default free plan).
	gotUser, err := env.DB.Users.GetByID(user.User.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if gotUser.PlanID != user.User.PlanID {
		t.Errorf("user plan changed from %d to %d despite superseded payment", user.User.PlanID, gotUser.PlanID)
	}
}
