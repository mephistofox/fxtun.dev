package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/server/database"
	"github.com/mephistofox/fxtun.dev/internal/server/payment"
)

// stubProvider is a payment.Provider that returns a fixed checkout URL, so the
// checkout handler can be exercised without talking to a real provider.
type stubProvider struct {
	name      string
	cancelled []string
}

func (p *stubProvider) Name() string { return p.name }

func (p *stubProvider) CreateCheckoutSession(params payment.CheckoutParams) (*payment.CheckoutResult, error) {
	return &payment.CheckoutResult{
		PaymentURL:        "https://pay.example/checkout",
		ProviderPaymentID: "stub-payment-1",
	}, nil
}

func (p *stubProvider) HandleWebhook(r *http.Request) ([]payment.WebhookEvent, error) {
	return nil, nil
}

func (p *stubProvider) CancelSubscription(providerSubscriptionID string) error {
	p.cancelled = append(p.cancelled, providerSubscriptionID)
	return nil
}

func (p *stubProvider) CancelPayment(providerPaymentID string) error { return nil }

// enableStubPayments wires stub YooKassa and Creem providers into the test
// server and returns the Creem stub, which records provider-side cancellations.
func enableStubPayments(t *testing.T, env *testEnv) *stubProvider {
	t.Helper()
	creem := &stubProvider{name: "creem"}
	reg := payment.NewRegistry()
	reg.Register(&stubProvider{name: "yookassa"})
	reg.Register(creem)
	env.APIServer.SetPaymentProviders(reg)
	env.APIServer.cfg.YooKassa.Enabled = true
	return creem
}

// seedActiveSubscription creates an active, recurring subscription with a saved
// card, as a paying user would have.
func seedActiveSubscription(t *testing.T, env *testEnv, userID, planID int64) *database.Subscription {
	t.Helper()
	start := time.Now().Add(-10 * 24 * time.Hour)
	end := start.AddDate(0, 1, 0)
	method := "pm-old"
	last4 := "4242"
	sub := &database.Subscription{
		UserID:                  userID,
		PlanID:                  planID,
		Status:                  database.SubscriptionStatusActive,
		Recurring:               true,
		CurrentPeriodStart:      &start,
		CurrentPeriodEnd:        &end,
		YooKassaPaymentMethodID: &method,
		YooKassaCardLast4:       &last4,
	}
	if err := env.DB.Subscriptions.Create(sub); err != nil {
		t.Fatalf("create active subscription: %v", err)
	}
	return sub
}

func postCheckout(t *testing.T, env *testEnv, token string, planID int64) *http.Response {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"plan_id": planID})
	req, _ := http.NewRequest("POST", env.Server.URL+"/api/subscription/checkout", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("checkout request failed: %v", err)
	}
	return resp
}

// TestCheckoutAllowsUpgradeWithActiveSubscription: a user on a cheaper plan must
// be able to pay for a pricier one without cancelling first. Refusing here is a
// straight revenue loss — the only alternative left to the user is to cancel,
// which also unbinds their card.
func TestCheckoutAllowsUpgradeWithActiveSubscription(t *testing.T) {
	env := setupTestEnv(t)
	enableStubPayments(t, env)
	user := env.createTestUser(t, "+30000000021", "password123", "Upgrade User")

	seedActiveSubscription(t, env, user.User.ID, 2) // base ($2.50)

	resp := postCheckout(t, env, user.AccessToken, 3) // pro ($5.00)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upgrade checkout: expected 200, got %d", resp.StatusCode)
	}

	pending, err := env.DB.Subscriptions.GetPendingByUserID(user.User.ID)
	if err != nil {
		t.Fatalf("get pending subscription: %v", err)
	}
	if pending == nil || pending.PlanID != 3 {
		t.Fatalf("expected a pending subscription for plan 3, got %+v", pending)
	}
}

// TestCheckoutRejectsDowngradeWithActiveSubscription: a downgrade must still go
// through the scheduled plan change, not a fresh paid checkout.
func TestCheckoutRejectsDowngradeWithActiveSubscription(t *testing.T) {
	env := setupTestEnv(t)
	enableStubPayments(t, env)
	user := env.createTestUser(t, "+30000000022", "password123", "Downgrade User")

	seedActiveSubscription(t, env, user.User.ID, 4) // business ($7.50)

	resp := postCheckout(t, env, user.AccessToken, 2) // base ($2.50)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("downgrade checkout: expected 400, got %d", resp.StatusCode)
	}
}

// TestActivateSubscriptionSupersedesPreviousActive: an upgrade leaves the old
// subscription alive next to the new one. Activating the new subscription must
// retire the old one, or the renewal jobs bill the user for both plans.
func TestActivateSubscriptionSupersedesPreviousActive(t *testing.T) {
	env := setupTestEnv(t)
	prov := enableStubPayments(t, env)
	user := env.createTestUser(t, "+30000000023", "password123", "Activate User")

	old := seedActiveSubscription(t, env, user.User.ID, 2)
	creemID := "creem-sub-old"
	old.CreemSubscriptionID = &creemID
	if err := env.DB.Subscriptions.Update(old); err != nil {
		t.Fatalf("seed creem id: %v", err)
	}

	newSub := &database.Subscription{
		UserID:    user.User.ID,
		PlanID:    3,
		Status:    database.SubscriptionStatusPending,
		Recurring: true,
	}
	if err := env.DB.Subscriptions.Create(newSub); err != nil {
		t.Fatalf("create pending subscription: %v", err)
	}
	pmt := &database.Payment{
		UserID:         user.User.ID,
		SubscriptionID: &newSub.ID,
		InvoiceID:      999021,
		Amount:         500,
		Status:         database.PaymentStatusSuccess,
		Provider:       "yookassa",
	}
	if err := env.DB.Payments.Create(pmt); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	env.APIServer.activateSubscription(newSub, pmt, "yookassa", "test")

	gotOld, err := env.DB.Subscriptions.GetByID(old.ID)
	if err != nil {
		t.Fatalf("get old subscription: %v", err)
	}
	if gotOld.Status != database.SubscriptionStatusExpired {
		t.Errorf("old subscription status = %q, want %q", gotOld.Status, database.SubscriptionStatusExpired)
	}
	if gotOld.Recurring {
		t.Error("old subscription is still recurring: it will renew and double-bill")
	}
	if gotOld.YooKassaPaymentMethodID != nil {
		t.Error("old subscription still holds a saved payment method")
	}
	if len(prov.cancelled) != 1 || prov.cancelled[0] != creemID {
		t.Errorf("provider-side subscription not cancelled, cancelled=%v", prov.cancelled)
	}

	live, err := env.DB.Subscriptions.GetByUserID(user.User.ID)
	if err != nil {
		t.Fatalf("get live subscription: %v", err)
	}
	if live == nil || live.ID != newSub.ID {
		t.Fatalf("live subscription = %+v, want id=%d", live, newSub.ID)
	}

	dbUser, err := env.DB.Users.GetByID(user.User.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if dbUser.PlanID != 3 {
		t.Errorf("user plan = %d, want 3", dbUser.PlanID)
	}
}
