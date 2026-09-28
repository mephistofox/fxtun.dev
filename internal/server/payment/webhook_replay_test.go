package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

// signedCreemRequest builds a Creem webhook request with a valid HMAC signature
// over the exact body, i.e. what an attacker replaying a captured request has.
func signedCreemRequest(t *testing.T, secret, body string) *http.Request {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	r := httptest.NewRequest(http.MethodPost, "/api/payments/webhook/creem", bytes.NewReader([]byte(body)))
	r.Header.Set("creem-signature", hex.EncodeToString(mac.Sum(nil)))
	return r
}

// TestCreemHandleWebhookIgnoresReplay pins replay protection: a captured
// body+signature resent verbatim must not yield a second event, otherwise a
// subscription.paid replay extends the paid period without a payment.
func TestCreemHandleWebhookIgnoresReplay(t *testing.T) {
	c := NewCreem(CreemConfig{WebhookSecret: "s3cret", TestMode: true})
	body := `{"id":"evt_replay_1","eventType":"subscription.paid","object":{"id":"sub_1","subscription_id":"sub_1","metadata":{"invoice_id":"42"}}}`

	events, err := c.HandleWebhook(signedCreemRequest(t, "s3cret", body))
	if err != nil {
		t.Fatalf("first HandleWebhook: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("first delivery produced %d events, want 1", len(events))
	}

	events, err = c.HandleWebhook(signedCreemRequest(t, "s3cret", body))
	if err != nil {
		t.Fatalf("replayed HandleWebhook: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("replayed delivery produced %d events, want 0", len(events))
	}
}

// TestCreemHandleWebhookDistinctEvents guards against the replay cache
// swallowing genuinely different deliveries.
func TestCreemHandleWebhookDistinctEvents(t *testing.T) {
	c := NewCreem(CreemConfig{WebhookSecret: "s3cret", TestMode: true})

	for _, id := range []string{"evt_a", "evt_b"} {
		body := `{"id":"` + id + `","eventType":"subscription.paid","object":{"id":"sub_1","subscription_id":"sub_1"}}`
		events, err := c.HandleWebhook(signedCreemRequest(t, "s3cret", body))
		if err != nil {
			t.Fatalf("HandleWebhook(%s): %v", id, err)
		}
		if len(events) != 1 {
			t.Fatalf("HandleWebhook(%s) produced %d events, want 1", id, len(events))
		}
	}
}

// TestParseWebhookEventRejectsMissingObject pins the nil guard: a webhook body
// without an "object" must be an error, not a nil dereference in the handler.
func TestParseWebhookEventRejectsMissingObject(t *testing.T) {
	if _, err := ParseWebhookEvent([]byte(`{"type":"notification","event":"payment.succeeded"}`)); err == nil {
		t.Fatal("expected error for webhook without object, got nil")
	}
	if _, err := ParseWebhookEvent([]byte(`{"type":"notification","event":"payment.succeeded","object":null}`)); err == nil {
		t.Fatal("expected error for webhook with null object, got nil")
	}
}
