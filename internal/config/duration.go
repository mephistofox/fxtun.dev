package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration parses a duration string with support for "d" suffix (days).
// This is the client-side equivalent of the server's parseTunnelDuration.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	if strings.HasSuffix(s, "d") {
		trimmed := strings.TrimSuffix(s, "d")
		days, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q: %w", s, err)
		}
		if days <= 0 {
			return 0, fmt.Errorf("invalid duration %q: must be positive", s)
		}
		return time.Duration(days * float64(24*time.Hour)), nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("invalid duration %q: must be positive", s)
	}
	return d, nil
}

// ValidateAutoClose validates the auto-close duration string.
// Minimum: 1m, Maximum: 24h.
func ValidateAutoClose(s string) error {
	if s == "" {
		return nil
	}
	d, err := ParseDuration(s)
	if err != nil {
		return err
	}
	if d < 1*time.Minute {
		return fmt.Errorf("auto-close minimum is 1m, got %s", s)
	}
	if d > 24*time.Hour {
		return fmt.Errorf("auto-close maximum is 24h, got %s", s)
	}
	return nil
}

// ValidateMaxLifetime validates the max-lifetime duration string.
// Minimum: 1m, Maximum: 7d (168h).
func ValidateMaxLifetime(s string) error {
	if s == "" {
		return nil
	}
	d, err := ParseDuration(s)
	if err != nil {
		return err
	}
	if d < 1*time.Minute {
		return fmt.Errorf("max-lifetime minimum is 1m, got %s", s)
	}
	if d > 7*24*time.Hour {
		return fmt.Errorf("max-lifetime maximum is 7d, got %s", s)
	}
	return nil
}
