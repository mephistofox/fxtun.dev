package exchange

import "testing"

// TestPlausibleUSDRate pins the sanity corridor for the USD/RUB rate. The
// source is an unofficial CBR mirror; a bogus value such as 1.0 would price a
// $20 plan at 20 RUB, so out-of-corridor values must be rejected outright.
func TestPlausibleUSDRate(t *testing.T) {
	cases := []struct {
		rate float64
		want bool
	}{
		{0, false},
		{-5, false},
		{1, false},
		{29.99, false},
		{30, true},
		{80, true},
		{500, true},
		{500.01, false},
		{100000, false},
	}
	for _, tc := range cases {
		if got := plausibleUSDRate(tc.rate); got != tc.want {
			t.Errorf("plausibleUSDRate(%v) = %v, want %v", tc.rate, got, tc.want)
		}
	}
}
