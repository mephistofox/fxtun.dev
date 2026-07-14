package api

import "testing"

// TestEffectiveRecurring guards the checkout gate that keeps payments working
// before the YooKassa shop is approved for autopayments. A production shop
// rejects save_payment_method with "This store can't make recurring payments",
// so YooKassa checkouts must be forced to one-time until the shop is enabled,
// while Creem (which manages its own subscriptions) stays always-recurring.
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
		{"yookassa not requested, shop enabled -> one-time", "yookassa", false, true, false},
		{"yookassa not requested, shop disabled -> one-time", "yookassa", false, false, false},
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
