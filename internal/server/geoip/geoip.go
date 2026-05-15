package geoip

import (
	"net"
	"strings"

	"github.com/oschwald/maxminddb-golang"
)

// Lookup provides GeoIP country lookups using an MMDB database.
type Lookup struct {
	db *maxminddb.Reader
}

// result is the MMDB query structure for country-level lookups.
type result struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

// regionCountries maps region prefixes to country codes.
// When a client's country matches a region, the node in that region is preferred.
var regionCountries = map[string][]string{
	"ru":   {"RU", "BY", "KZ", "UZ", "KG", "TJ", "AM", "AZ", "GE", "MD", "UA", "LV", "LT", "EE", "FI"},
	"eu":   {"DE", "FR", "GB", "NL", "IT", "ES", "PL", "CZ", "AT", "CH", "BE", "SE", "NO", "DK", "IE", "PT", "RO", "BG", "HR", "SK", "HU", "SI", "LU"},
	"us":   {"US", "CA", "MX"},
	"asia": {"JP", "KR", "SG", "HK", "TW", "IN", "TH", "VN", "MY", "ID", "PH", "AU", "NZ"},
}

// New opens an MMDB database file and returns a Lookup.
func New(dbPath string) (*Lookup, error) {
	db, err := maxminddb.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &Lookup{db: db}, nil
}

// Country returns the ISO country code for the given IP address.
// Returns empty string if lookup fails or the Lookup is nil.
func (l *Lookup) Country(ipStr string) string {
	if l == nil || l.db == nil {
		return ""
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}
	var r result
	if err := l.db.Lookup(ip, &r); err != nil {
		return ""
	}
	return r.Country.ISOCode
}

// Close releases the MMDB database resources.
func (l *Lookup) Close() error {
	if l != nil && l.db != nil {
		return l.db.Close()
	}
	return nil
}

// RegionMatchesCountry checks whether a node region matches a client's country code.
// The region string is split on "-" and the first segment is used as the prefix
// (e.g., "ru-msk" -> "ru", "eu-fra" -> "eu").
func RegionMatchesCountry(region, country string) bool {
	if region == "" || country == "" {
		return false
	}
	prefix := region
	if idx := strings.Index(region, "-"); idx > 0 {
		prefix = region[:idx]
	}
	countries, ok := regionCountries[prefix]
	if !ok {
		return false
	}
	for _, c := range countries {
		if c == country {
			return true
		}
	}
	return false
}
