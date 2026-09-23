package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

// TunnelLookup resolves a tunnel subdomain to its registry entry.
// The entry's ServerID identifies the edge node hosting the tunnel.
type TunnelLookup interface {
	LookupBySubdomain(subdomain string) (*store.TunnelEntry, error)
}

// NodeLookup retrieves an edge node entry by its ID.
// Used to translate a tunnel's ServerID into a publicly resolvable IP.
type NodeLookup interface {
	GetNode(nodeID string) (*store.NodeEntry, error)
}

// Config holds DNS server configuration.
type Config struct {
	Enabled  bool
	Listen   string // e.g. ":53"
	ZoneFile string // path to YAML zone file
}

// Server is an authoritative DNS server backed by a static YAML zone file plus
// a dynamic Redis tunnel registry for tunnel subdomain resolution.
type Server struct {
	cfg       Config
	zones     map[string]*Zone // key: "<name>." (lowercase, trailing dot)
	tunnels   TunnelLookup
	nodes     NodeLookup
	log       zerolog.Logger
	udpServer *dns.Server
	tcpServer *dns.Server

	rlMu    sync.Mutex
	buckets map[string]*bucket // source IP -> token bucket
}

// bucket is one source IP's token bucket for the query rate limiter.
type bucket struct {
	tokens float64
	last   time.Time
}

const (
	// An authoritative server that answers every query from every source is a
	// usable amplification relay. These bounds are well above what a real
	// recursive resolver needs for our zones and well below what makes the
	// server worth pointing at a victim.
	//
	// ponytail: fixed rate, no config knob — add one when a legitimate client
	// actually trips it.
	rateBurst  = 200.0
	ratePerSec = 100.0
	// bucketsMax bounds the limiter's memory: a spoofed-source flood would
	// otherwise grow the map without limit. Overflowing drops the whole table,
	// which is cheap and self-healing.
	bucketsMax = 20000
)

// New constructs a DNS server, loading and validating the zone file.
func New(cfg Config, tunnels TunnelLookup, nodes NodeLookup, log zerolog.Logger) (*Server, error) {
	if cfg.ZoneFile == "" {
		return nil, fmt.Errorf("dns: zone_file is required")
	}
	zf, err := LoadZoneFile(cfg.ZoneFile)
	if err != nil {
		return nil, fmt.Errorf("dns: load zone file: %w", err)
	}
	zones := make(map[string]*Zone, len(zf.Zones))
	for i := range zf.Zones {
		z := &zf.Zones[i]
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
		cfg:     cfg,
		zones:   zones,
		tunnels: tunnels,
		nodes:   nodes,
		log:     log.With().Str("component", "dns").Logger(),
	}, nil
}

// Start launches the UDP and TCP DNS listeners. Non-blocking.
func (s *Server) Start() error {
	mux := dns.NewServeMux()
	mux.HandleFunc(".", s.handle)

	s.udpServer = &dns.Server{Addr: s.cfg.Listen, Net: "udp", Handler: mux}
	s.tcpServer = &dns.Server{Addr: s.cfg.Listen, Net: "tcp", Handler: mux}

	go func() {
		if err := s.udpServer.ListenAndServe(); err != nil {
			s.log.Error().Err(err).Msg("DNS UDP server error")
		}
	}()
	go func() {
		if err := s.tcpServer.ListenAndServe(); err != nil {
			s.log.Error().Err(err).Msg("DNS TCP server error")
		}
	}()

	zoneNames := make([]string, 0, len(s.zones))
	for name := range s.zones {
		zoneNames = append(zoneNames, strings.TrimSuffix(name, "."))
	}
	s.log.Info().
		Str("addr", s.cfg.Listen).
		Strs("zones", zoneNames).
		Msg("DNS server started")
	return nil
}

// Stop gracefully shuts the DNS listeners down.
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if s.udpServer != nil {
		_ = s.udpServer.ShutdownContext(ctx)
	}
	if s.tcpServer != nil {
		_ = s.tcpServer.ShutdownContext(ctx)
	}
}

// handle is the main DNS query dispatcher.
func (s *Server) handle(w dns.ResponseWriter, r *dns.Msg) {
	// Over quota: drop without answering. Replying (even with REFUSED) would
	// still hand an attacker a packet to aim at a spoofed source.
	if !s.allow(w.RemoteAddr()) {
		return
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.RecursionAvailable = false

	if len(r.Question) == 0 {
		m.SetRcode(r, dns.RcodeFormatError)
		s.writeMsg(w, r, m)
		return
	}

	q := r.Question[0]
	qName := strings.ToLower(q.Name)

	zone, subdomain := s.findZone(qName)
	if zone == nil {
		m.SetRcode(r, dns.RcodeRefused)
		s.writeMsg(w, r, m)
		return
	}

	// RFC 8482: do not expand ANY. An ANY query at the apex returns every
	// record set in one packet (~10x the query), which is the single biggest
	// amplification lever an authoritative server offers. Answer with the
	// prescribed synthesized HINFO instead.
	if q.Qtype == dns.TypeANY {
		m.Answer = append(m.Answer, &dns.HINFO{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeHINFO, Class: dns.ClassINET, Ttl: 3600},
			Cpu: "RFC8482",
		})
		s.writeMsg(w, r, m)
		return
	}

	// nameExists tracks whether the queried name owns records of ANY type. It
	// decides the empty-answer response below: a name that exists but has no
	// record of the queried type is NODATA (NOERROR), while a name that does
	// not exist at all is NXDOMAIN. Returning NXDOMAIN for an existing name
	// (e.g. an AAAA query on a name that only has an A record) poisons resolver
	// negative caches for EVERY type of that name (RFC 2308 §5), making it stop
	// resolving entirely for the negative-cache TTL.
	nameExists := subdomain == "" // the apex always exists

	// SOA query at apex — synthesize from zone.
	if subdomain == "" && (q.Qtype == dns.TypeSOA || q.Qtype == dns.TypeANY) {
		m.Answer = append(m.Answer, buildSOA(zone))
	}

	// Dynamic tunnel lookup. Only consult the registry for actual subdomains
	// (not the apex) and only when the zone enables tunnels. A registered
	// tunnel shadows any static/wildcard record for the same name and makes the
	// name exist for every query type even though we only ever answer A. We do
	// the lookup for A/ANY queries (which need the answer) and, in zones with
	// no wildcard, for other types too (needed to classify NODATA vs NXDOMAIN);
	// in a wildcard zone the wildcard already establishes existence, so we skip
	// the extra registry hit on non-address types.
	tunnelHit := false
	if zone.TunnelsEnabled && subdomain != "" &&
		(q.Qtype == dns.TypeA || q.Qtype == dns.TypeANY || !zone.hasWildcard) {
		if ip := s.lookupTunnel(subdomain); ip != "" {
			nameExists = true
			tunnelHit = true
			if q.Qtype == dns.TypeA || q.Qtype == dns.TypeANY {
				if parsed := net.ParseIP(ip).To4(); parsed != nil {
					rr := &dns.A{
						Hdr: dns.RR_Header{
							Name:   q.Name,
							Rrtype: dns.TypeA,
							Class:  dns.ClassINET,
							Ttl:    30,
						},
						A: parsed,
					}
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}

	// Static and wildcard records. A registered tunnel shadows both (matching
	// the previous tunnel-priority behavior), so only consult them when no
	// tunnel matched.
	if !tunnelHit {
		// Static records — exact match first.
		for _, rec := range zone.Records {
			if rec.Name == "*" {
				continue // wildcards handled separately below as a fallback
			}
			fullName := strings.ToLower(rec.FullName(zone.Name))
			if fullName != qName {
				continue
			}
			nameExists = true // the name owns at least one record of some type
			if !matchType(rec.Type, q.Qtype) {
				continue
			}
			ttl := rec.TTL
			if ttl == 0 {
				ttl = zone.TTL
			}
			if ttl == 0 {
				ttl = 300
			}
			if rr := buildRR(rec, q.Name, ttl); rr != nil {
				m.Answer = append(m.Answer, rr)
			}
		}

		// Wildcard fallback — only for subdomain queries (never the apex), and
		// only when no exact-match record was found. Mirrors RFC 1034 §4.3.3
		// wildcard semantics. Provides a friendly "tunnel not found" 404 from
		// the nginx + fxtunnel chain instead of a DNS error for non-existent
		// tunnel subdomains.
		if len(m.Answer) == 0 && subdomain != "" {
			for _, rec := range zone.Records {
				if rec.Name != "*" {
					continue
				}
				// A wildcard record makes the queried name exist via synthesis
				// (RFC 4592), even when its type does not match the query —
				// that yields NODATA, not NXDOMAIN.
				nameExists = true
				if !matchType(rec.Type, q.Qtype) {
					continue
				}
				ttl := rec.TTL
				if ttl == 0 {
					ttl = zone.TTL
				}
				if ttl == 0 {
					ttl = 300
				}
				if rr := buildRR(rec, q.Name, ttl); rr != nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}

	// Empty answer. Distinguish NODATA (name exists, no record of the queried
	// type) from NXDOMAIN (name does not exist at all). Both carry the SOA in
	// the authority section for RFC 2308 negative caching, but NXDOMAIN may
	// only be returned for a truly non-existent name — returning it for an
	// existing name poisons the resolver's negative cache for ALL record types
	// of that name, so the name stops resolving entirely (e.g. an AAAA query
	// dropping the cached A record under Happy Eyeballs).
	if len(m.Answer) == 0 {
		if !nameExists {
			m.SetRcode(r, dns.RcodeNameError)
		}
		m.Ns = append(m.Ns, buildSOA(zone))
	}

	s.writeMsg(w, r, m)
}

// allow applies a per-source-IP token bucket to an incoming query.
func (s *Server) allow(addr net.Addr) bool {
	ip := ""
	if addr != nil {
		ip = addr.String()
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}
	}

	now := time.Now()
	s.rlMu.Lock()
	defer s.rlMu.Unlock()

	if s.buckets == nil {
		s.buckets = make(map[string]*bucket)
	}
	b, ok := s.buckets[ip]
	if !ok {
		if len(s.buckets) >= bucketsMax {
			s.buckets = make(map[string]*bucket, bucketsMax)
		}
		b = &bucket{tokens: rateBurst, last: now}
		s.buckets[ip] = b
	}

	b.tokens += now.Sub(b.last).Seconds() * ratePerSec
	if b.tokens > rateBurst {
		b.tokens = rateBurst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// writeMsg echoes the request's EDNS0 OPT and truncates the reply to what the
// transport can actually carry before writing it.
//
// Without the OPT echo the client's advertised buffer size is lost and the
// reply is silently assumed to fit; without truncation an oversized UDP answer
// is fragmented or dropped, and an unbounded answer size turns the server into
// an amplifier. TCP has no such limit, so it is only bounded by the protocol
// maximum.
func (s *Server) writeMsg(w dns.ResponseWriter, r, m *dns.Msg) {
	size := dns.MinMsgSize
	if opt := r.IsEdns0(); opt != nil {
		m.SetEdns0(opt.UDPSize(), opt.Do())
		if advertised := int(opt.UDPSize()); advertised > size {
			size = advertised
		}
		if size > dns.MaxMsgSize {
			size = dns.MaxMsgSize
		}
	}
	if _, isTCP := w.RemoteAddr().(*net.TCPAddr); isTCP {
		size = dns.MaxMsgSize
	}
	m.Truncate(size)
	_ = w.WriteMsg(m)
}

// findZone selects the zone whose name is the longest suffix of qName, and
// returns the subdomain portion (relative to that zone, "" for the apex).
//
// qName "myapp.mfdev.ru." with zone "mfdev.ru" → subdomain "myapp".
// qName "mfdev.ru." with zone "mfdev.ru" → subdomain "".
func (s *Server) findZone(qName string) (*Zone, string) {
	name := strings.TrimSuffix(qName, ".")
	var bestZone *Zone
	var bestSub string
	bestLen := -1
	for zoneName, zone := range s.zones {
		zn := strings.TrimSuffix(zoneName, ".")
		switch {
		case name == zn:
			if len(zn) > bestLen {
				bestZone = zone
				bestSub = ""
				bestLen = len(zn)
			}
		case strings.HasSuffix(name, "."+zn):
			if len(zn) > bestLen {
				bestZone = zone
				bestSub = strings.TrimSuffix(name, "."+zn)
				bestLen = len(zn)
			}
		}
	}
	return bestZone, bestSub
}

// lookupTunnel checks the Redis tunnel registry for the given subdomain and
// returns the IP that should answer for it. The tunnel registry stores the
// hosting node's HTTPAddr in ServerID (e.g. "159.194.203.103:10080"); we
// strip the port to get the bare IP. If a NodeLookup is configured we prefer
// its PublicAddr instead, which is correct when ServerID is a node ID rather
// than a host:port pair.
func (s *Server) lookupTunnel(subdomain string) string {
	if s.tunnels == nil {
		return ""
	}
	entry, err := s.tunnels.LookupBySubdomain(subdomain)
	if err != nil {
		s.log.Debug().Err(err).Str("subdomain", subdomain).Msg("tunnel lookup error")
		return ""
	}
	if entry == nil || entry.ServerID == "" {
		return ""
	}

	// First try parsing ServerID as host:port (the common case in this codebase).
	if host, _, err := net.SplitHostPort(entry.ServerID); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
		// Hostname rather than IP — fall through to NodeLookup or DNS.
	} else if ip := net.ParseIP(entry.ServerID); ip != nil {
		return ip.String()
	}

	// Fall back: treat ServerID as a node ID and look it up.
	if s.nodes != nil {
		if node, err := s.nodes.GetNode(entry.ServerID); err == nil && node != nil {
			if host, _, err := net.SplitHostPort(node.PublicAddr); err == nil {
				if ip := net.ParseIP(host); ip != nil {
					return ip.String()
				}
			}
			if ip := net.ParseIP(node.PublicAddr); ip != nil {
				return ip.String()
			}
		}
	}
	return ""
}

// matchType returns true if a record of recType should answer a query of qType.
func matchType(recType string, qType uint16) bool {
	switch strings.ToUpper(recType) {
	case "A":
		return qType == dns.TypeA || qType == dns.TypeANY
	case "AAAA":
		return qType == dns.TypeAAAA || qType == dns.TypeANY
	case "MX":
		return qType == dns.TypeMX || qType == dns.TypeANY
	case "TXT":
		return qType == dns.TypeTXT || qType == dns.TypeANY
	case "NS":
		return qType == dns.TypeNS || qType == dns.TypeANY
	case "CNAME":
		return qType == dns.TypeCNAME || qType == dns.TypeANY
	case "CAA":
		return qType == dns.TypeCAA || qType == dns.TypeANY
	case "SRV":
		return qType == dns.TypeSRV || qType == dns.TypeANY
	case "SOA":
		return qType == dns.TypeSOA || qType == dns.TypeANY
	}
	return false
}

// buildRR converts a Record into a miekg/dns RR.
func buildRR(rec Record, qname string, ttl uint32) dns.RR {
	hdr := dns.RR_Header{Name: qname, Class: dns.ClassINET, Ttl: ttl}
	switch strings.ToUpper(rec.Type) {
	case "A":
		ip := net.ParseIP(rec.Value)
		if ip == nil || ip.To4() == nil {
			return nil
		}
		hdr.Rrtype = dns.TypeA
		return &dns.A{Hdr: hdr, A: ip.To4()}
	case "AAAA":
		ip := net.ParseIP(rec.Value)
		if ip == nil || ip.To16() == nil {
			return nil
		}
		hdr.Rrtype = dns.TypeAAAA
		return &dns.AAAA{Hdr: hdr, AAAA: ip.To16()}
	case "MX":
		if !fitsUint16(rec.Priority) || !validName(rec.Value) {
			return nil
		}
		hdr.Rrtype = dns.TypeMX
		return &dns.MX{Hdr: hdr, Preference: uint16(rec.Priority), Mx: dns.Fqdn(rec.Value)} //nolint:gosec // range-checked above
	case "TXT":
		hdr.Rrtype = dns.TypeTXT
		return &dns.TXT{Hdr: hdr, Txt: splitTXT(rec.Value)}
	case "NS":
		if !validName(rec.Value) {
			return nil
		}
		hdr.Rrtype = dns.TypeNS
		return &dns.NS{Hdr: hdr, Ns: dns.Fqdn(rec.Value)}
	case "CNAME":
		if !validName(rec.Value) {
			return nil
		}
		hdr.Rrtype = dns.TypeCNAME
		return &dns.CNAME{Hdr: hdr, Target: dns.Fqdn(rec.Value)}
	case "CAA":
		// Expected format: `<flags> <tag> "<value>"` e.g. `0 issue "letsencrypt.org"`.
		parts := strings.SplitN(rec.Value, " ", 3)
		if len(parts) != 3 {
			return nil
		}
		tag := parts[1]
		value := strings.Trim(parts[2], "\"")
		// Both are packed as character-strings, capped at 255 bytes each.
		if len(tag) > 255 || len(value) > 255 {
			return nil
		}
		hdr.Rrtype = dns.TypeCAA
		return &dns.CAA{
			Hdr:   hdr,
			Flag:  0,
			Tag:   tag,
			Value: value,
		}
	case "SRV":
		if !fitsUint16(rec.Priority) || !fitsUint16(rec.Weight) || !fitsUint16(rec.Port) || !validName(rec.Value) {
			return nil
		}
		hdr.Rrtype = dns.TypeSRV
		return &dns.SRV{
			Hdr:      hdr,
			Priority: uint16(rec.Priority), //nolint:gosec // range-checked above
			Weight:   uint16(rec.Weight),   //nolint:gosec // range-checked above
			Port:     uint16(rec.Port),     //nolint:gosec // range-checked above
			Target:   dns.Fqdn(rec.Value),
		}
	}
	return nil
}

// fitsUint16 reports whether a zone-file integer fits the 16-bit wire field it
// is destined for. Out-of-range values would silently wrap into a different
// number (port 70000 becomes 4464), so the record is dropped instead.
func fitsUint16(v int) bool { return v >= 0 && v <= 65535 }

// validName reports whether a zone-file value can be packed as a domain name.
// An unpackable name does not just break its own record: the whole reply fails
// to pack and the query is answered with nothing at all.
func validName(v string) bool {
	name := dns.Fqdn(v)
	// A value ending in a backslash escapes the dot Fqdn appends, so the name
	// stays relative and fails to pack even though IsDomainName accepts it.
	if !dns.IsFqdn(name) {
		return false
	}
	_, ok := dns.IsDomainName(name)
	return ok
}

// splitTXT chunks a TXT value into 255-byte segments as required by the DNS spec.
func splitTXT(value string) []string {
	const maxLen = 255
	if len(value) <= maxLen {
		return []string{value}
	}
	var out []string
	for len(value) > maxLen {
		out = append(out, value[:maxLen])
		value = value[maxLen:]
	}
	if len(value) > 0 {
		out = append(out, value)
	}
	return out
}

// buildSOA synthesizes an SOA record for a zone. The serial is derived from
// the current time so that secondary servers (if any) can detect changes.
func buildSOA(zone *Zone) dns.RR {
	zoneName := dns.Fqdn(zone.Name)
	return &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   zoneName,
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    3600,
		},
		Ns:      "ns1." + zoneName,
		Mbox:    "admin." + zoneName,
		Serial:  uint32(time.Now().Unix()), //nolint:gosec // unix seconds fit uint32 until year 2106
		Refresh: 3600,
		Retry:   900,
		Expire:  604800,
		Minttl:  300,
	}
}
