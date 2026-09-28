package payment

import "testing"

// TestSavedCardLast4 pins that only a genuine 4-digit bank-card tail is stored.
// Webhooks are IP-verified but unsigned, so any unexpected value must be dropped
// rather than displayed as the user's saved card.
func TestSavedCardLast4(t *testing.T) {
	card := func(last4 string) *Payment {
		p := &Payment{PaymentMethod: &PaymentMethod{Type: "bank_card", Saved: true}}
		p.PaymentMethod.Card = &struct {
			First6      string `json:"first6,omitempty"`
			Last4       string `json:"last4,omitempty"`
			ExpiryYear  string `json:"expiry_year,omitempty"`
			ExpiryMonth string `json:"expiry_month,omitempty"`
			CardType    string `json:"card_type,omitempty"`
		}{Last4: last4}
		return p
	}

	cases := []struct {
		name string
		pmt  *Payment
		want string
	}{
		{"valid 4 digits", card("4242"), "4242"},
		{"empty", card(""), ""},
		{"too short", card("42"), ""},
		{"too long", card("42424"), ""},
		{"non-digit", card("4a42"), ""},
		{"injection attempt", card("<b>x"), ""},
		{"nil payment", nil, ""},
		{"not saved", &Payment{PaymentMethod: &PaymentMethod{Saved: false}}, ""},
		{"no card (e.g. yoo_money)", &Payment{PaymentMethod: &PaymentMethod{Type: "yoo_money", Saved: true}}, ""},
		{"no payment method", &Payment{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.pmt.SavedCardLast4(); got != tc.want {
				t.Errorf("SavedCardLast4() = %q, want %q", got, tc.want)
			}
		})
	}
}
