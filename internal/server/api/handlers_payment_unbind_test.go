package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/api/dto"
	"github.com/mephistofox/fxtunnel/internal/server/database"
)

// createActiveSubWithCard creates an active, recurring subscription with a saved
// YooKassa card for the given user and returns it.
func createActiveSubWithCard(t *testing.T, env *testEnv, userID int64) *database.Subscription {
	t.Helper()
	pmID := "pm_test_123"
	last4 := "4242"
	now := time.Now()
	end := now.AddDate(0, 1, 0)
	sub := &database.Subscription{
		UserID:                  userID,
		PlanID:                  1, // seeded Free plan; any valid FK works for this test
		Status:                  database.SubscriptionStatusActive,
		Recurring:               true,
		CurrentPeriodStart:      &now,
		CurrentPeriodEnd:        &end,
		YooKassaPaymentMethodID: &pmID,
		YooKassaCardLast4:       &last4,
	}
	if err := env.DB.Subscriptions.Create(sub); err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	return sub
}

func doAuthedPost(t *testing.T, env *testEnv, path, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, env.Server.URL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", path, err)
	}
	return resp
}

// TestUnbindCard_DetachesCardAndStopsRenewal verifies the core YooKassa
// compliance requirement: a user can detach their saved card themselves. The
// subscription must stop auto-renewing but remain active until period end.
func TestUnbindCard_DetachesCardAndStopsRenewal(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79001112233", "password123", "Card User")
	sub := createActiveSubWithCard(t, env, user.User.ID)

	resp := doAuthedPost(t, env, "/api/subscription/unbind-card", user.AccessToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unbind status = %d, want 200", resp.StatusCode)
	}

	got, err := env.DB.Subscriptions.GetByID(sub.ID)
	if err != nil {
		t.Fatalf("reload subscription: %v", err)
	}
	if got.YooKassaPaymentMethodID != nil {
		t.Errorf("YooKassaPaymentMethodID = %v, want nil", *got.YooKassaPaymentMethodID)
	}
	if got.YooKassaCardLast4 != nil {
		t.Errorf("YooKassaCardLast4 = %v, want nil", *got.YooKassaCardLast4)
	}
	if got.Recurring {
		t.Errorf("Recurring = true, want false")
	}
	if got.Status != database.SubscriptionStatusActive {
		t.Errorf("Status = %s, want active (access kept until period end)", got.Status)
	}
}

// TestGetSubscription_CardBoundFlag verifies the profile-facing DTO exposes the
// bound card so the UI can render "•••• 4242" and an unbind control.
func TestGetSubscription_CardBoundFlag(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79004445566", "password123", "Flag User")
	createActiveSubWithCard(t, env, user.User.ID)

	req, _ := http.NewRequest(http.MethodGet, env.Server.URL+"/api/subscription", nil)
	req.Header.Set("Authorization", "Bearer "+user.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get subscription: %v", err)
	}
	defer resp.Body.Close()

	var body dto.SubscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Subscription == nil {
		t.Fatal("subscription is nil")
	}
	if !body.Subscription.CardBound {
		t.Error("CardBound = false, want true")
	}
	if body.Subscription.CardLast4 != "4242" {
		t.Errorf("CardLast4 = %q, want 4242", body.Subscription.CardLast4)
	}
}

// TestUnbindCard_Idempotent verifies unbinding twice is safe and the second call
// reports there is nothing bound rather than erroring.
func TestUnbindCard_Idempotent(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79007778899", "password123", "Idem User")
	createActiveSubWithCard(t, env, user.User.ID)

	first := doAuthedPost(t, env, "/api/subscription/unbind-card", user.AccessToken)
	first.Body.Close()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first unbind = %d, want 200", first.StatusCode)
	}

	second := doAuthedPost(t, env, "/api/subscription/unbind-card", user.AccessToken)
	defer second.Body.Close()
	if second.StatusCode != http.StatusOK {
		t.Fatalf("second unbind = %d, want 200", second.StatusCode)
	}
}

// TestUnbindCard_NoSubscription returns 404 when the user has no subscription.
func TestUnbindCard_NoSubscription(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+79001010101", "password123", "NoSub User")

	resp := doAuthedPost(t, env, "/api/subscription/unbind-card", user.AccessToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unbind status = %d, want 404", resp.StatusCode)
	}
}
