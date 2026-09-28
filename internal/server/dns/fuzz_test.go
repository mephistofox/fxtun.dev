package dns

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/miekg/dns"
)

// fuzzZones is the zone set every DNS fuzz target answers from: one zone with
// tunnels + a wildcard, one plain zone, mirroring the production zone file.
func fuzzZones() []Zone {
	return []Zone{
		{
			Name:           "fxtun.dev",
			TunnelsEnabled: true,
			TTL:            300,
			Records: []Record{
				{Name: "@", Type: "A", Value: "45.12.74.91"},
				{Name: "*", Type: "A", Value: "45.12.74.91"},
				{Name: "ns1", Type: "A", Value: "45.12.74.91"},
				{Name: "@", Type: "MX", Value: "mail.fxtun.dev", Priority: 10},
				{Name: "@", Type: "TXT", Value: "v=spf1 -all"},
				{Name: "@", Type: "CAA", Value: `0 issue "letsencrypt.org"`},
				{Name: "_svc", Type: "SRV", Value: "svc.fxtun.dev", Priority: 1, Weight: 2, Port: 443},
			},
		},
		{
			Name:    "fxtun.ru",
			TTL:     300,
			Records: []Record{{Name: "@", Type: "AAAA", Value: "2a03::1"}},
		},
	}
}

// FuzzHandle drives the query handler with arbitrary wire-format packets.
//
// Invariants: the handler never panics, the reply it produces must be packable
// (an unpackable reply is silently dropped in production), and a plain UDP
// query with no EDNS0 must never be answered with more than 512 bytes — that
// bound is what keeps an authoritative server from being an amplifier.
func FuzzHandle(f *testing.F) {
	seed := func(name string, qtype uint16, edns bool) {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(name), qtype)
		if edns {
			m.SetEdns0(4096, false)
		}
		if b, err := m.Pack(); err == nil {
			f.Add(b)
		}
	}
	seed("fxtun.dev", dns.TypeSOA, false)
	seed("fxtun.dev", dns.TypeANY, true)
	seed("myapp.fxtun.dev", dns.TypeA, false)
	seed("myapp.fxtun.dev", dns.TypeAAAA, false)
	seed("nope.example.org", dns.TypeA, false)
	seed("_svc.fxtun.dev", dns.TypeSRV, true)
	seed("fxtun.ru", dns.TypeTXT, false)
	f.Add([]byte{})
	f.Add([]byte{0x00})

	f.Fuzz(func(t *testing.T, packet []byte) {
		r := new(dns.Msg)
		if err := r.Unpack(packet); err != nil {
			return
		}

		// A fresh server per iteration: the rate limiter is stateful and would
		// otherwise start dropping queries a few hundred execs in.
		s := newTestServer(stubTunnels{"myapp": "45.12.74.91"}, fuzzZones()...)
		w := &captureWriter{}
		s.handle(w, r)

		if w.msg == nil {
			return // dropped, e.g. over quota — nothing to check
		}
		out, err := w.msg.Pack()
		if err != nil {
			t.Fatalf("reply is not packable: %v\nreply: %v", err, w.msg)
		}
		if r.IsEdns0() == nil && len(out) > dns.MinMsgSize {
			t.Fatalf("UDP reply without EDNS0 is %d bytes (> %d)", len(out), dns.MinMsgSize)
		}
	})
}

// FuzzBuildRR converts a fuzzed zone record into a resource record and packs
// it, exercising the int→uint16 narrowing in the SRV/MX paths.
func FuzzBuildRR(f *testing.F) {
	f.Add("A", "1.2.3.4", 0, 0, 0, uint32(300))
	f.Add("SRV", "svc.example.com", 65536, -1, 70000, uint32(0))
	f.Add("MX", "mail.example.com", -32769, 0, 0, uint32(4294967295))
	f.Add("CAA", `0 issue "letsencrypt.org"`, 0, 0, 0, uint32(1))
	f.Add("TXT", "x", 0, 0, 0, uint32(1))
	f.Add("CNAME", "\x00..", 0, 0, 0, uint32(1))

	f.Fuzz(func(t *testing.T, typ, value string, prio, weight, port int, ttl uint32) {
		rec := Record{Name: "x", Type: typ, Value: value, Priority: prio, Weight: weight, Port: port, TTL: ttl}
		rr := buildRR(rec, "x.fxtun.dev.", ttl)
		if rr == nil {
			return
		}
		m := new(dns.Msg)
		m.SetQuestion("x.fxtun.dev.", dns.TypeA)
		m.Answer = append(m.Answer, rr)
		if _, err := m.Pack(); err != nil {
			t.Fatalf("record %+v built an unpackable RR: %v", rec, err)
		}
		// Narrowing must not wrap: a value that does not fit the wire field
		// has to be rejected, not silently turned into a different number.
		switch v := rr.(type) {
		case *dns.SRV:
			if int(v.Priority) != prio || int(v.Weight) != weight || int(v.Port) != port {
				t.Fatalf("SRV fields wrapped: %d/%d/%d from %d/%d/%d",
					v.Priority, v.Weight, v.Port, prio, weight, port)
			}
		case *dns.MX:
			if int(v.Preference) != prio {
				t.Fatalf("MX preference wrapped: %d from %d", v.Preference, prio)
			}
		}
	})
}

// FuzzFindZone checks the longest-suffix zone match against arbitrary names.
func FuzzFindZone(f *testing.F) {
	f.Add("fxtun.dev.")
	f.Add("a.b.c.fxtun.dev.")
	f.Add(".")
	f.Add("")
	f.Add("xfxtun.dev.")

	s := newTestServer(nil, fuzzZones()...)
	f.Fuzz(func(t *testing.T, qname string) {
		zone, sub := s.findZone(qname)
		if zone == nil {
			return
		}
		// The subdomain must be what is left of the query name once the zone
		// suffix is removed — the tunnel registry is keyed on it.
		if sub == "" {
			return
		}
		full := sub + "." + zone.Name + "."
		if full != dns.Fqdn(qname) {
			t.Fatalf("findZone(%q) = zone %q sub %q, recombines to %q", qname, zone.Name, sub, full)
		}
	})
}

// FuzzLoadZoneFile parses arbitrary YAML as a zone file.
func FuzzLoadZoneFile(f *testing.F) {
	f.Add([]byte("zones:\n  - name: fxtun.dev\n    records:\n      - {name: '@', type: A, value: 1.2.3.4}\n"))
	f.Add([]byte("zones: []"))
	f.Add([]byte("zones:\n  - name: x\n    ttl: 99999999999999999999\n"))

	dir := f.TempDir()
	path := filepath.Join(dir, "zones.yaml")
	f.Fuzz(func(t *testing.T, data []byte) {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		zf, err := LoadZoneFile(path)
		if err != nil {
			return
		}
		for i := range zf.Zones {
			for _, rec := range zf.Zones[i].Records {
				_ = rec.FullName(zf.Zones[i].Name)
			}
		}
	})
}
