package core

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// autoCloseTimer tracks idle time and calls onClose when the tunnel has been
// idle (no activity) for the configured duration.
type autoCloseTimer struct {
	mu           sync.Mutex
	duration     time.Duration
	lastActivity time.Time
	timer        *time.Timer
	onClose      func()
	stopped      bool
}

// newAutoCloseTimer creates and starts an auto-close timer.
// The onClose callback is invoked once when the idle timeout expires.
func newAutoCloseTimer(duration time.Duration, onClose func()) *autoCloseTimer {
	t := &autoCloseTimer{
		duration:     duration,
		lastActivity: time.Now(),
		onClose:      onClose,
	}
	t.timer = time.AfterFunc(duration, t.check)
	return t
}

// recordActivity resets the idle timer. Should be called on every tunnel activity.
func (t *autoCloseTimer) recordActivity() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return
	}
	t.lastActivity = time.Now()
	t.timer.Reset(t.duration)
}

// check is called when the timer fires. It checks whether the tunnel has actually
// been idle long enough, and either fires the callback or reschedules.
func (t *autoCloseTimer) check() {
	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return
	}

	idle := time.Since(t.lastActivity)
	if idle >= t.duration {
		t.stopped = true
		t.mu.Unlock() // release BEFORE callback to avoid deadlock
		t.onClose()
		return
	}
	// Activity happened since the timer was set; reschedule for the remaining time.
	remaining := t.duration - idle
	t.timer.Reset(remaining)
	t.mu.Unlock()
}

// stop cancels the auto-close timer. Safe to call multiple times.
func (t *autoCloseTimer) stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopped = true
	t.timer.Stop()
}

// maxLifetimeTimer fires onClose exactly once after the specified duration,
// regardless of activity.
type maxLifetimeTimer struct {
	mu      sync.Mutex
	timer   *time.Timer
	stopped bool
}

// newMaxLifetimeTimer creates and starts a max-lifetime timer.
func newMaxLifetimeTimer(duration time.Duration, onClose func()) *maxLifetimeTimer {
	t := &maxLifetimeTimer{}
	t.timer = time.AfterFunc(duration, func() {
		t.mu.Lock()
		if t.stopped {
			t.mu.Unlock()
			return
		}
		t.stopped = true
		t.mu.Unlock() // release BEFORE callback to avoid deadlock
		onClose()
	})
	return t
}

// stop cancels the max-lifetime timer. Safe to call multiple times.
func (t *maxLifetimeTimer) stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopped = true
	t.timer.Stop()
}

// parseDuration parses a duration string with support for "d" suffix (days).
// This is the client-side equivalent of the server's parseTunnelDuration.
func parseDuration(s string) (time.Duration, error) {
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
	d, err := parseDuration(s)
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
	d, err := parseDuration(s)
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
