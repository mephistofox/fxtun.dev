package api

import (
	"testing"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/database"
	"github.com/mephistofox/fxtunnel/internal/server/payment"
)

// makePendingSubAndPayment creates a pending subscription + pending payment for a
// user on the given plan and returns them, mirroring what handleCheckout persists
// before the provider confirms payment.
func makePendingSubAndPayment(t *testing.T, env *testEnv, userID, planID int64) (*database.Subscription, *database.Payment) {
	t.Helper()
	sub := &database.Subscription{
		UserID:    userID,
		PlanID:    planID,
		Status:    database.SubscriptionStatusPending,
		Recurring: false,
	}
	if err := env.DB.Subscriptions.Create(sub); err != nil {
		t.Fatalf("create sub: %v", err)
	}
	invoice, err := env.DB.Payments.GetNextInvoiceID()
	if err != nil {
		t.Fatalf("invoice id: %v", err)
	}
	pmt := &database.Payment{
		UserID:         userID,
		SubscriptionID: &sub.ID,
		InvoiceID:      invoice,
		Amount:         200,
		Status:         database.PaymentStatusPending,
		Provider:       "yookassa",
	}
	if err := env.DB.Payments.Create(pmt); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	return sub, pmt
}

// succeededYoo builds a succeeded YooKassa payment object for the given invoice
// amount, optionally with a saved card.
func succeededYoo(invoice int64, amount string, savedCardLast4 string) *payment.Payment {
	p := &payment.Payment{
		ID:       "yoo_" + amount,
		Status:   "succeeded",
		Paid:     true,
		Amount:   payment.Amount{Value: amount, Currency: "RUB"},
		Metadata: map[string]string{"invoice_id": itoa(invoice)},
	}
	if savedCardLast4 != "" {
		pm := &payment.PaymentMethod{Type: "bank_card", ID: "pm_saved_1", Saved: true}
		pm.Card = &struct {
			First6      string `json:"first6,omitempty"`
			Last4       string `json:"last4,omitempty"`
			ExpiryYear  string `json:"expiry_year,omitempty"`
			ExpiryMonth string `json:"expiry_month,omitempty"`
			CardType    string `json:"card_type,omitempty"`
		}{Last4: savedCardLast4}
		p.PaymentMethod = pm
	}
	return p
}

func itoa(v int64) string {
	// small local helper to avoid importing strconv just for tests
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// TestApplySucceededPayment_Activates is the core reconciler/webhook path: a
// pending payment marked succeeded activates the subscription and upgrades the
// user, independent of any webhook.
func TestApplySucceededPayment_Activates(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79001110001", "password123", "Recon User")
	plan, err := env.DB.Plans.GetBySlug("base")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	sub, pmt := makePendingSubAndPayment(t, env, user.User.ID, plan.ID)

	res, err := env.APIServer.applySucceededPayment(pmt, succeededYoo(pmt.InvoiceID, "200.00", ""), plan.ID, "webhook")
	if err != nil {
		t.Fatalf("applySucceededPayment: %v", err)
	}
	if res != paymentApplied {
		t.Fatalf("result = %v, want paymentApplied", res)
	}

	gotSub, _ := env.DB.Subscriptions.GetByID(sub.ID)
	if gotSub.Status != database.SubscriptionStatusActive {
		t.Errorf("sub status = %s, want active", gotSub.Status)
	}
	if gotSub.CurrentPeriodEnd == nil || !gotSub.CurrentPeriodEnd.After(time.Now()) {
		t.Errorf("period end not set in the future: %v", gotSub.CurrentPeriodEnd)
	}
	gotPmt, _ := env.DB.Payments.GetByID(pmt.ID)
	if gotPmt.Status != database.PaymentStatusSuccess {
		t.Errorf("payment status = %s, want success", gotPmt.Status)
	}
	if u, _ := env.DB.Users.GetByID(user.User.ID); u.PlanID != plan.ID {
		t.Errorf("user plan = %d, want %d", u.PlanID, plan.ID)
	}
}

// TestApplySucceededPayment_Idempotent verifies a second application is a no-op,
// so the webhook and the reconciler can both run without double-activating.
func TestApplySucceededPayment_Idempotent(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79001110002", "password123", "Idem User")
	plan, _ := env.DB.Plans.GetBySlug("base")
	_, pmt := makePendingSubAndPayment(t, env, user.User.ID, plan.ID)

	if _, err := env.APIServer.applySucceededPayment(pmt, succeededYoo(pmt.InvoiceID, "200.00", ""), plan.ID, "webhook"); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	// Reload to reflect the persisted success status.
	reloaded, _ := env.DB.Payments.GetByID(pmt.ID)
	res, err := env.APIServer.applySucceededPayment(reloaded, succeededYoo(pmt.InvoiceID, "200.00", ""), plan.ID, "webhook")
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if res != paymentAlreadyDone {
		t.Errorf("result = %v, want paymentAlreadyDone", res)
	}
}

// TestApplySucceededPayment_SupersededNotActivated verifies a payment already
// marked failed (superseded / stale-cleaned) is NOT activated — guards against
// granting a second subscription / double-charge.
func TestApplySucceededPayment_SupersededNotActivated(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79001110003", "password123", "Super User")
	plan, _ := env.DB.Plans.GetBySlug("base")
	sub, pmt := makePendingSubAndPayment(t, env, user.User.ID, plan.ID)
	pmt.Status = database.PaymentStatusFailed
	if err := env.DB.Payments.Update(pmt); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	res, err := env.APIServer.applySucceededPayment(pmt, succeededYoo(pmt.InvoiceID, "200.00", ""), plan.ID, "webhook")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res != paymentSuperseded {
		t.Errorf("result = %v, want paymentSuperseded", res)
	}
	if gotSub, _ := env.DB.Subscriptions.GetByID(sub.ID); gotSub.Status == database.SubscriptionStatusActive {
		t.Error("superseded payment must not activate the subscription")
	}
}

// TestApplySucceededPayment_StaleSnapshotSuperseded exercises the TOCTOU guard:
// the caller holds a pending in-memory snapshot while the DB row was concurrently
// marked failed (a supersede racing a slow provider poll). The atomic claim must
// lose and the payment must NOT be activated (no second subscription).
func TestApplySucceededPayment_StaleSnapshotSuperseded(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79001110007", "password123", "Race User")
	plan, _ := env.DB.Plans.GetBySlug("base")
	sub, pmt := makePendingSubAndPayment(t, env, user.User.ID, plan.ID)

	// Concurrent supersede: another loader marks the DB row failed. pmt stays a
	// stale "pending" snapshot in this caller's hand.
	other, _ := env.DB.Payments.GetByID(pmt.ID)
	other.Status = database.PaymentStatusFailed
	if err := env.DB.Payments.Update(other); err != nil {
		t.Fatalf("supersede: %v", err)
	}

	res, err := env.APIServer.applySucceededPayment(pmt, succeededYoo(pmt.InvoiceID, "200.00", ""), plan.ID, "reconciler")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res != paymentSuperseded {
		t.Errorf("result = %v, want paymentSuperseded (stale snapshot must lose the claim)", res)
	}
	if gotSub, _ := env.DB.Subscriptions.GetByID(sub.ID); gotSub.Status == database.SubscriptionStatusActive {
		t.Error("stale-snapshot payment must not activate the subscription")
	}
}

// TestApplySucceededPayment_AmountMismatch rejects a provider amount outside the
// tolerance (forged/incorrect webhook or payment).
func TestApplySucceededPayment_AmountMismatch(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79001110004", "password123", "Amount User")
	plan, _ := env.DB.Plans.GetBySlug("base")
	_, pmt := makePendingSubAndPayment(t, env, user.User.ID, plan.ID)

	_, err := env.APIServer.applySucceededPayment(pmt, succeededYoo(pmt.InvoiceID, "5.00", ""), plan.ID, "webhook")
	if err != errPaymentAmountMismatch {
		t.Fatalf("err = %v, want errPaymentAmountMismatch", err)
	}
	if gotPmt, _ := env.DB.Payments.GetByID(pmt.ID); gotPmt.Status == database.PaymentStatusSuccess {
		t.Error("amount-mismatched payment must not be marked success")
	}
}

// TestApplySucceededPayment_BindsCard verifies a saved card is stored on the
// subscription so future autopayments have a method.
func TestApplySucceededPayment_BindsCard(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79001110005", "password123", "Card User")
	plan, _ := env.DB.Plans.GetBySlug("base")
	sub, pmt := makePendingSubAndPayment(t, env, user.User.ID, plan.ID)

	if _, err := env.APIServer.applySucceededPayment(pmt, succeededYoo(pmt.InvoiceID, "200.00", "4242"), plan.ID, "webhook"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gotSub, _ := env.DB.Subscriptions.GetByID(sub.ID)
	if gotSub.YooKassaPaymentMethodID == nil || *gotSub.YooKassaPaymentMethodID != "pm_saved_1" {
		t.Errorf("payment method id not bound: %v", gotSub.YooKassaPaymentMethodID)
	}
	if gotSub.YooKassaCardLast4 == nil || *gotSub.YooKassaCardLast4 != "4242" {
		t.Errorf("card last4 not bound: %v", gotSub.YooKassaCardLast4)
	}
}

// TestListPendingByProviderInWindow verifies the reconciler query returns only
// pending YooKassa payments whose age is inside [min, max], excluding fresh ones
// (webhook's job) and old ones (already stale-cleaned).
func TestListPendingByProviderInWindow(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79001110006", "password123", "Window User")
	plan, _ := env.DB.Plans.GetBySlug("base")
	_, pmt := makePendingSubAndPayment(t, env, user.User.ID, plan.ID)

	now := time.Now()
	// Fresh (just created) → excluded by the 3-min lower bound.
	fresh, err := env.DB.Payments.ListPendingByProviderInWindow("yookassa", now.Add(-reconcileMaxAge), now.Add(-reconcileMinAge))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, p := range fresh {
		if p.ID == pmt.ID {
			t.Error("freshly created payment must be excluded from the reconcile window")
		}
	}
	// Wide window including now → included.
	all, err := env.DB.Payments.ListPendingByProviderInWindow("yookassa", now.Add(-reconcileMaxAge), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	found := false
	for _, p := range all {
		if p.ID == pmt.ID {
			found = true
		}
	}
	if !found {
		t.Error("pending payment should be listed within a window covering now")
	}
}
