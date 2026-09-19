package api

import (
	"sync"
	"testing"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/database"
	"github.com/mephistofox/fxtunnel/internal/server/exchange"
	"github.com/mephistofox/fxtunnel/internal/server/payment"
)

// countAuditAction returns how many audit entries with the given action exist
// for a user.
func countAuditAction(t *testing.T, env *testEnv, userID int64, action string) int {
	t.Helper()
	logs, _, err := env.DB.Audit.GetByUserID(userID, 100, 0)
	if err != nil {
		t.Fatalf("audit logs: %v", err)
	}
	n := 0
	for _, l := range logs {
		if l.Action == action {
			n++
		}
	}
	return n
}

// TestCreemPaymentSucceededActivatesOnce pins that two concurrent deliveries of
// the same Creem success event activate the subscription exactly once. The
// read-then-update path lets both deliveries observe "pending" and both
// activate, which double-counts the billing period.
func TestCreemPaymentSucceededActivatesOnce(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+30000000021", "password123", "Creem User")

	_, pmt := newPendingCheckout(t, env, user.User.ID, 999021)

	evt := payment.WebhookEvent{
		Type:                   payment.WebhookEventPaymentSucceeded,
		InvoiceID:              pmt.InvoiceID,
		ProviderPaymentID:      "creem-pay-1",
		ProviderSubscriptionID: "creem-sub-1",
		ProviderData:           map[string]interface{}{"event_type": "checkout.completed"},
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			env.APIServer.handleCreemPaymentSucceeded(evt)
		}()
	}
	close(start)
	wg.Wait()

	if n := countAuditAction(t, env, user.User.ID, "subscription_activated"); n != 1 {
		t.Errorf("subscription activated %d times, want 1", n)
	}
}

// TestCreemSubscriptionDeletedKeepsNewerActivePlan pins that deleting a stale
// Creem subscription does not downgrade a user who already paid for a newer,
// still-running subscription.
func TestCreemSubscriptionDeletedKeepsNewerActivePlan(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+30000000022", "password123", "Creem User 2")

	pro, err := env.DB.Plans.GetBySlug("pro")
	if err != nil || pro == nil {
		t.Fatalf("pro plan: %v", err)
	}
	dbUser, err := env.DB.Users.GetByID(user.User.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	dbUser.PlanID = pro.ID
	if err := env.DB.Users.Update(dbUser); err != nil {
		t.Fatalf("update user: %v", err)
	}

	creemSubID := "creem-sub-stale"
	oldEnd := time.Now().Add(-time.Hour)
	oldSub := &database.Subscription{
		UserID: user.User.ID, PlanID: pro.ID, Status: database.SubscriptionStatusCancelled,
		CurrentPeriodEnd: &oldEnd, CreemSubscriptionID: &creemSubID,
	}
	if err := env.DB.Subscriptions.Create(oldSub); err != nil {
		t.Fatalf("old sub: %v", err)
	}

	newEnd := time.Now().Add(29 * 24 * time.Hour)
	newSub := &database.Subscription{
		UserID: user.User.ID, PlanID: pro.ID, Status: database.SubscriptionStatusActive,
		CurrentPeriodEnd: &newEnd,
	}
	if err := env.DB.Subscriptions.Create(newSub); err != nil {
		t.Fatalf("new sub: %v", err)
	}

	env.APIServer.handleCreemSubscriptionDeleted(payment.WebhookEvent{
		Type:                   payment.WebhookEventSubscriptionDeleted,
		ProviderSubscriptionID: creemSubID,
	})

	got, err := env.DB.Users.GetByID(user.User.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.PlanID != pro.ID {
		t.Errorf("user plan_id = %d, want %d: paid subscription was downgraded", got.PlanID, pro.ID)
	}
}

// TestRecoverPaymentFromWebhookUsesPlanPrice pins that a payment recovered from
// webhook metadata gets its amount from the plan, not from the webhook body.
// Taking it from the webhook makes the later ±1% amount check self-referential,
// so a forged webhook can activate a paid plan for one rouble.
func TestRecoverPaymentFromWebhookUsesPlanPrice(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+30000000023", "password123", "Recover User")

	pro, err := env.DB.Plans.GetBySlug("pro")
	if err != nil || pro == nil {
		t.Fatalf("pro plan: %v", err)
	}

	pmt, err := env.APIServer.recoverPaymentFromWebhook(999023, user.User.ID, 0, pro.ID)
	if err != nil {
		t.Fatalf("recoverPaymentFromWebhook: %v", err)
	}

	want := exchange.ConvertUSDToRUB(pro.Price)
	if pmt.Amount != want {
		t.Errorf("recovered payment amount = %v, want %v (plan price)", pmt.Amount, want)
	}
}
