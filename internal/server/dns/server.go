package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/server/store"
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
}

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
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.RecursionAvailable = false

	if len(r.Question) == 0 {
		m.SetRcode(r, dns.RcodeFormatError)
		_ = w.WriteMsg(m)
		return
	}

	q := r.Question[0]
	qName := strings.ToLower(q.Name)

	zone, subdomain := s.findZone(qName)
	if zone == nil {
		m.SetRcode(r, dns.RcodeRefused)
		_ = w.WriteMsg(m)
		return
	}

	// SOA query at apex — synthesize from zone.
	if subdomain == "" && (q.Qtype == dns.TypeSOA || q.Qtype == dns.TypeANY) {
		m.Answer = append(m.Answer, buildSOA(zone))
	}

	// Dynamic tunnel lookup. Only consult the registry for actual subdomains
	// (not the apex) and only when the zone enables tunnels.
	if zone.TunnelsEnabled && subdomain != "" {
		if q.Qtype == dns.TypeA || q.Qtype == dns.TypeANY {
			if ip := s.lookupTunnel(subdomain); ip != "" {
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
					_ = w.WriteMsg(m)
					return
				}
			}
		}
	}

	// Static records.
	for _, rec := range zone.Records {
		fullName := strings.ToLower(rec.FullName(zone.Name))
		if fullName != qName {
			continue
		}
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

	// No answers — return NXDOMAIN with SOA in authority section
	// (RFC 2308 negative caching) so resolvers cache the negative result briefly.
	if len(m.Answer) == 0 {
		m.SetRcode(r, dns.RcodeNameError)
		m.Ns = append(m.Ns, buildSOA(zone))
	}

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
		hdr.Rrtype = dns.TypeMX
		return &dns.MX{Hdr: hdr, Preference: uint16(rec.Priority), Mx: dns.Fqdn(rec.Value)}
	case "TXT":
		hdr.Rrtype = dns.TypeTXT
		return &dns.TXT{Hdr: hdr, Txt: splitTXT(rec.Value)}
	case "NS":
		hdr.Rrtype = dns.TypeNS
		return &dns.NS{Hdr: hdr, Ns: dns.Fqdn(rec.Value)}
	case "CNAME":
		hdr.Rrtype = dns.TypeCNAME
		return &dns.CNAME{Hdr: hdr, Target: dns.Fqdn(rec.Value)}
	case "CAA":
		// Expected format: `<flags> <tag> "<value>"` e.g. `0 issue "letsencrypt.org"`.
		parts := strings.SplitN(rec.Value, " ", 3)
		if len(parts) != 3 {
			return nil
		}
		hdr.Rrtype = dns.TypeCAA
		return &dns.CAA{
			Hdr:   hdr,
			Flag:  0,
			Tag:   parts[1],
			Value: strings.Trim(parts[2], "\""),
		}
	case "SRV":
		hdr.Rrtype = dns.TypeSRV
		return &dns.SRV{
			Hdr:      hdr,
			Priority: uint16(rec.Priority),
			Weight:   uint16(rec.Weight),
			Port:     uint16(rec.Port),
			Target:   dns.Fqdn(rec.Value),
		}
	}
	return nil
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
		Serial:  uint32(time.Now().Unix()),
		Refresh: 3600,
		Retry:   900,
		Expire:  604800,
		Minttl:  300,
	}
}
