package core

import "testing"

func TestPortBlocked(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		isAdmin bool
		blocked map[int]bool
		want    bool
	}{
		{"udp dns non-admin blocked", 53, false, blockedUDPPorts, true},
		{"udp dns admin allowed", 53, true, blockedUDPPorts, false},
		{"udp ntp non-admin blocked", 123, false, blockedUDPPorts, true},
		{"udp arbitrary non-admin allowed", 40000, false, blockedUDPPorts, false},
		{"udp auto-allocate (0) allowed", 0, false, blockedUDPPorts, false},
		{"tcp ssh non-admin blocked", 22, false, blockedTCPPorts, true},
		{"tcp postgres non-admin blocked", 5432, false, blockedTCPPorts, true},
		{"tcp ssh admin allowed", 22, true, blockedTCPPorts, false},
		{"tcp arbitrary non-admin allowed", 8080, false, blockedTCPPorts, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := portBlocked(tt.port, tt.isAdmin, tt.blocked); got != tt.want {
				t.Errorf("portBlocked(%d, admin=%v) = %v, want %v", tt.port, tt.isAdmin, got, tt.want)
			}
		})
	}
}

func TestBlockedUDPPortsCoversDNS(t *testing.T) {
	// Port 53 (the server's own DNS for ACME/wildcard) MUST be blocked for UDP.
	if !blockedUDPPorts[53] {
		t.Fatal("blockedUDPPorts must contain 53 (server DNS) to prevent shadowing")
	}
}
