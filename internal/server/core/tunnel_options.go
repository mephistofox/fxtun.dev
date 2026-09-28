package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseTunnelDuration parses a human-friendly duration string.
// It supports standard Go durations ("30s", "5m", "1h") plus a "d" suffix for days (e.g. "1d" = 24h).
// Returns 0 for an empty string.
func parseTunnelDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// Handle "d" suffix for days
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
