package monitor

import "time"

// Config holds global monitor settings (detection only, rate limits come from plans).
type Config struct {
	Enabled           bool
	DetectionInterval time.Duration
	Detection         DetectionConfig
}

// DefaultConfig returns default monitor configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		DetectionInterval: 30 * time.Second,
		Detection:         DefaultDetectionConfig(),
	}
}
