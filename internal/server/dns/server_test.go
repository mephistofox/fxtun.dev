package dns

import (
	"net"
	"strings"
	"testing"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

// captureWriter is a dns.ResponseWriter that records the last written message.
type captureWriter struct{ msg *dns.Msg }

func (c *captureWriter) LocalAddr() net.Addr       { return &net.UDPAddr{} }
func (c *captureWriter) RemoteAddr() net.Addr      { return &net.UDPAddr{} }
func (c *captureWriter) WriteMsg(m *dns.Msg) error { c.msg = m; return nil }
func (c *captureWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
func (c *captureWriter) Close() error        { return nil }
func (c *captureWriter) TsigStatus() error   { return nil }
func (c *captureWriter) TsigTimersOnly(bool) {}
func (c *captureWriter) Hijack()             {}

// stubTunnels is a TunnelLookup backed by an in-memory subdomain→IP map.
type stubTunnels map[string]string

func (s stubTunnels) LookupBySubdomain(subdomain string) (*store.TunnelEntry, error) {
	ip, ok := s[subdomain]
	if !ok {
		return nil, nil
	}
	return &store.TunnelEntry{Subdomain: subdomain, ServerID: ip + ":10080"}, nil
}

// newTestServer builds a Server from in-memory zones, mirroring New().
func newTestServer(tunnels TunnelLookup, zoneList ...Zone) *Server {
	zones := make(map[string]*Zone, len(zoneList))
	for i := range zoneList {
		z := &zoneList[i]
		name := strings.ToLower(strings.TrimSuffix(z.Name, ".")) + "."
		for _, rec := range z.Records {
			if rec.Name == "*" {
				z.hasWildcard = true
				break
			}
		}
		zones[name] = z
	}
	return &Server{
		cfg:     Config{Listen: ":0"},
		zones:   zones,
		tunnels: tunnels,
		log:     zerolog.Nop(),
	}
}

func query(s *Server, name string, qtype uint16) *dns.Msg {
	r := new(dns.Msg)
	r.SetQuestion(dns.Fqdn(name), qtype)
	w := &captureWriter{}
	s.handle(w, r)
	return w.msg
}

// zoneWithWildcard mirrors the fxtun.dev prod zone shape: static infra records
// plus a wildcard A fallback, tunnels enabled.
func zoneWithWildcard() Zone {
	return Zone{
		Name:           "fxtun.dev",
		TunnelsEnabled: true,
		TTL:            300,
		Records: []Record{
			{Name: "@", Type: "A", Value: "95.181.173.114"},
			{Name: "www", Type: "A", Value: "95.181.173.114"},
			{Name: "@", Type: "NS", Value: "ns1.fxtun.dev."},
			{Name: "*", Type: "A", Value: "95.181.173.114"},
		},
	}
}

// zoneNoWildcard mirrors the staging mfdev.ru zone: no wildcard, tunnels enabled.
func zoneNoWildcard() Zone {
	return Zone{
		Name:           "mfdev.ru",
		TunnelsEnabled: true,
		TTL:            300,
		Records: []Record{
			{Name: "@", Type: "A", Value: "155.212.137.199"},
			{Name: "www", Type: "A", Value: "87.236.16.192"},
		},
	}
}

func TestNoDataNotNxDomain(t *testing.T) {
	tests := []struct {
		name       string
		zones      []Zone
		tunnels    stubTunnels
		qname      string
		qtype      uint16
		wantRcode  int
		wantAnswer bool // expect at least one answer RR
		wantSOA    bool // expect SOA in authority section
	}{
		{
			name:      "AAAA on wildcard-covered name is NODATA not NXDOMAIN",
			zones:     []Zone{zoneWithWildcard()},
			qname:     "tunnel.fxtun.dev",
			qtype:     dns.TypeAAAA,
			wantRcode: dns.RcodeSuccess,
			wantSOA:   true,
		},
		{
			name:       "A on wildcard-covered name resolves",
			zones:      []Zone{zoneWithWildcard()},
			qname:      "tunnel.fxtun.dev",
			qtype:      dns.TypeA,
			wantRcode:  dns.RcodeSuccess,
			wantAnswer: true,
		},
		{
			name:      "AAAA on static A subdomain is NODATA",
			zones:     []Zone{zoneWithWildcard()},
			qname:     "www.fxtun.dev",
			qtype:     dns.TypeAAAA,
			wantRcode: dns.RcodeSuccess,
			wantSOA:   true,
		},
		{
			name:      "AAAA on apex is NODATA",
			zones:     []Zone{zoneWithWildcard()},
			qname:     "fxtun.dev",
			qtype:     dns.TypeAAAA,
			wantRcode: dns.RcodeSuccess,
			wantSOA:   true,
		},
		{
			name:      "AAAA on live tunnel without wildcard is NODATA",
			zones:     []Zone{zoneNoWildcard()},
			tunnels:   stubTunnels{"live": "155.212.137.199"},
			qname:     "live.mfdev.ru",
			qtype:     dns.TypeAAAA,
			wantRcode: dns.RcodeSuccess,
			wantSOA:   true,
		},
		{
			name:       "A on live tunnel without wildcard resolves",
			zones:      []Zone{zoneNoWildcard()},
			tunnels:    stubTunnels{"live": "155.212.137.199"},
			qname:      "live.mfdev.ru",
			qtype:      dns.TypeA,
			wantRcode:  dns.RcodeSuccess,
			wantAnswer: true,
		},
		{
			name:      "truly nonexistent name without wildcard is NXDOMAIN",
			zones:     []Zone{zoneNoWildcard()},
			qname:     "ghost.mfdev.ru",
			qtype:     dns.TypeA,
			wantRcode: dns.RcodeNameError,
			wantSOA:   true,
		},
		{
			name:      "MX on wildcard-covered name is NODATA",
			zones:     []Zone{zoneWithWildcard()},
			qname:     "tunnel.fxtun.dev",
			qtype:     dns.TypeMX,
			wantRcode: dns.RcodeSuccess,
			wantSOA:   true,
		},
		{
			name:       "ANY on wildcard-covered name resolves via wildcard A",
			zones:      []Zone{zoneWithWildcard()},
			qname:      "tunnel.fxtun.dev",
			qtype:      dns.TypeANY,
			wantRcode:  dns.RcodeSuccess,
			wantAnswer: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tunnels TunnelLookup
			if tt.tunnels != nil {
				tunnels = tt.tunnels
			}
			s := newTestServer(tunnels, tt.zones...)
			m := query(s, tt.qname, tt.qtype)
			if m == nil {
				t.Fatal("no response written")
			}
			if !m.Authoritative {
				t.Error("response is not authoritative")
			}
			if m.Rcode != tt.wantRcode {
				t.Errorf("rcode = %s, want %s", dns.RcodeToString[m.Rcode], dns.RcodeToString[tt.wantRcode])
			}
			if got := len(m.Answer) > 0; got != tt.wantAnswer {
				t.Errorf("has answer = %v, want %v (answer=%v)", got, tt.wantAnswer, m.Answer)
			}
			if tt.wantSOA {
				hasSOA := false
				for _, rr := range m.Ns {
					if _, ok := rr.(*dns.SOA); ok {
						hasSOA = true
					}
				}
				if !hasSOA {
					t.Errorf("expected SOA in authority section, got %v", m.Ns)
				}
			}
		})
	}
}

// TestTunnelShadowsWildcard verifies that a registered tunnel wins over the
// zone's wildcard A record: an A query returns exactly the tunnel's node IP,
// not the wildcard IP, and not both. This guards the removal of the tunnel
// path's early return — the tunnel must still shadow the wildcard fallback.
func TestTunnelShadowsWildcard(t *testing.T) {
	const tunnelIP = "203.0.113.7"
	s := newTestServer(stubTunnels{"live": tunnelIP}, zoneWithWildcard())

	m := query(s, "live.fxtun.dev", dns.TypeA)
	if m == nil {
		t.Fatal("no response written")
	}
	if len(m.Answer) != 1 {
		t.Fatalf("expected exactly one answer, got %d: %v", len(m.Answer), m.Answer)
	}
	a, ok := m.Answer[0].(*dns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", m.Answer[0])
	}
	if a.A.String() != tunnelIP {
		t.Errorf("answer = %s, want tunnel IP %s (wildcard leaked?)", a.A, tunnelIP)
	}
}
