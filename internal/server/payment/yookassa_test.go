package payment

import (
	"net"
	"testing"
)

func TestLocalBoundDialer(t *testing.T) {
	tests := []struct {
		name     string
		sourceIP string
		wantNil  bool
		wantIP   string
	}{
		{name: "empty falls back to default source", sourceIP: "", wantNil: true},
		{name: "invalid IP falls back to default source", sourceIP: "not-an-ip", wantNil: true},
		{name: "valid IPv4 binds LocalAddr", sourceIP: "185.173.145.56", wantNil: false, wantIP: "185.173.145.56"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := localBoundDialer(tt.sourceIP)
			if tt.wantNil {
				if d != nil {
					t.Fatalf("localBoundDialer(%q) = %+v, want nil", tt.sourceIP, d)
				}
				return
			}
			if d == nil {
				t.Fatalf("localBoundDialer(%q) = nil, want dialer bound to %s", tt.sourceIP, tt.wantIP)
			}
			tcpAddr, ok := d.LocalAddr.(*net.TCPAddr)
			if !ok {
				t.Fatalf("LocalAddr type = %T, want *net.TCPAddr", d.LocalAddr)
			}
			if !tcpAddr.IP.Equal(net.ParseIP(tt.wantIP)) {
				t.Fatalf("LocalAddr IP = %s, want %s", tcpAddr.IP, tt.wantIP)
			}
		})
	}
}

func TestNewYooKassaBindsSourceIP(t *testing.T) {
	// With a source IP the client must carry a custom transport (bound dialer);
	// without one it must use the zero-config default transport.
	bound := NewYooKassa(YooKassaConfig{SourceIP: "185.173.145.56"})
	if bound.client.Transport == nil {
		t.Fatal("expected custom Transport when SourceIP is set, got nil (default)")
	}

	def := NewYooKassa(YooKassaConfig{})
	if def.client.Transport != nil {
		t.Fatal("expected default Transport when SourceIP is empty, got custom")
	}
}
