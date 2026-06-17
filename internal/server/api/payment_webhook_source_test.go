package api

import "testing"

func TestWebhookSourceAllowed(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		testMode bool
		want     bool
	}{
		{"public attacker, prod", "203.0.113.7:443", false, false},
		{"public attacker, test mode", "203.0.113.7:443", true, false},
		{"loopback, prod rejected", "127.0.0.1:5000", false, false},
		{"loopback, test mode allowed", "127.0.0.1:5000", true, true},
		{"private, test mode allowed", "10.1.2.3:5000", true, true},
		{"private, prod rejected", "10.1.2.3:5000", false, false},
		{"garbage addr", "not-an-addr", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := webhookSourceAllowed(tt.addr, tt.testMode); got != tt.want {
				t.Errorf("webhookSourceAllowed(%q, test=%v) = %v, want %v", tt.addr, tt.testMode, got, tt.want)
			}
		})
	}
}
