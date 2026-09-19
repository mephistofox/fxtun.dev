package api

import "testing"

// TestEffectiveRecurring guards the checkout gate. Subscriptions are monthly
// with auto-renewal only — there is no one-time option in the UI, and the API
// must not offer one either, so a YooKassa checkout follows the shop flag alone
// and ignores what the client asked for. The flag still matters: a production
// shop rejects save_payment_method with "This store can't make recurring
// payments" until autopayments are approved. Creem manages its own
// subscriptions and stays always-recurring.
func TestEffectiveRecurring(t *testing.T) {
	cases := []struct {
		name            string
		provider        string
		requested       bool
		yookassaEnabled bool
		want            bool
	}{
		{"yookassa requested but shop disabled -> one-time", "yookassa", true, false, false},
		{"yookassa requested and shop enabled -> recurring", "yookassa", true, true, true},
		{"yookassa opt-out ignored while shop enabled -> recurring", "yookassa", false, true, true},
		{"yookassa opt-out with shop disabled -> one-time", "yookassa", false, false, false},
		{"creem always recurring even if shop flag off", "creem", false, false, true},
		{"creem always recurring", "creem", true, false, true},
		{"unknown provider honours request true", "other", true, false, true},
		{"unknown provider honours request false", "other", false, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectiveRecurring(tc.provider, tc.requested, tc.yookassaEnabled); got != tc.want {
				t.Errorf("effectiveRecurring(%q, %v, %v) = %v, want %v",
					tc.provider, tc.requested, tc.yookassaEnabled, got, tc.want)
			}
		})
	}
}
