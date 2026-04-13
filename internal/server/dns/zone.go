package dns

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Zone represents a single DNS zone loaded from a YAML zone file.
type Zone struct {
	Name           string   `yaml:"name"`
	TunnelsEnabled bool     `yaml:"tunnels_enabled"`
	TTL            uint32   `yaml:"ttl"`
	Records        []Record `yaml:"records"`
}

// Record describes a single DNS record entry from the zone file.
type Record struct {
	Name     string `yaml:"name"` // "@" for apex, or a subdomain label
	Type     string `yaml:"type"` // A, AAAA, MX, TXT, NS, CNAME, CAA, SRV, SOA
	Value    string `yaml:"value"`
	Priority int    `yaml:"priority,omitempty"`
	Weight   int    `yaml:"weight,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	TTL      uint32 `yaml:"ttl,omitempty"` // overrides zone TTL when non-zero
}

// ZoneFile is the top-level YAML structure with one or more zones.
type ZoneFile struct {
	Zones []Zone `yaml:"zones"`
}

// LoadZoneFile reads and parses a YAML zone file from disk.
func LoadZoneFile(path string) (*ZoneFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var zf ZoneFile
	if err := yaml.Unmarshal(data, &zf); err != nil {
		return nil, err
	}
	return &zf, nil
}

// FullName returns the fully qualified name (with trailing dot) for a record
// relative to the given zone name.
//
// "@" or empty → "<zoneName>."
// otherwise   → "<name>.<zoneName>."
func (r Record) FullName(zoneName string) string {
	zoneName = strings.TrimSuffix(zoneName, ".")
	if r.Name == "@" || r.Name == "" {
		return zoneName + "."
	}
	return strings.TrimSuffix(r.Name, ".") + "." + zoneName + "."
}
