package core

import (
	"sync"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/config"
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

	// traffic returns the bytes the tunnel has moved so far (nil: none).
	// The timer polls it once a second instead of the copy loops pushing
	// activity, so a busy tunnel pays nothing per chunk; the close lands
	// at most a second after the idle period.
	traffic   func() int64
	lastBytes int64
}

// autoClosePoll is how often the timer looks at the traffic counters.
const autoClosePoll = time.Second

// newAutoCloseTimer creates and starts an auto-close timer.
// The onClose callback is invoked once when the idle timeout expires.
func newAutoCloseTimer(duration time.Duration, traffic func() int64, onClose func()) *autoCloseTimer {
	t := &autoCloseTimer{
		duration:     duration,
		lastActivity: time.Now(),
		onClose:      onClose,
		traffic:      traffic,
	}
	if traffic != nil {
		t.lastBytes = traffic()
	}
	t.mu.Lock() // check reads t.timer
	t.timer = time.AfterFunc(min(duration, autoClosePoll), t.check)
	t.mu.Unlock()
	return t
}

// recordActivity marks the tunnel active now (a new stream).
func (t *autoCloseTimer) recordActivity() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastActivity = time.Now()
}

// check is called when the timer fires. It checks whether the tunnel has actually
// been idle long enough, and either fires the callback or reschedules.
func (t *autoCloseTimer) check() {
	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return
	}
	if t.traffic != nil {
		if b := t.traffic(); b != t.lastBytes {
			t.lastBytes = b
			t.lastActivity = time.Now()
		}
	}

	idle := time.Since(t.lastActivity)
	if idle >= t.duration {
		t.stopped = true
		t.mu.Unlock() // release BEFORE callback to avoid deadlock
		t.onClose()
		return
	}
	// Not idle long enough yet; look again when it could be.
	t.timer.Reset(min(t.duration-idle, autoClosePoll))
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

// The duration rules live in config so config.Validate applies them to
// tunnels from a file too; these aliases keep the core API unchanged.
var (
	parseDuration       = config.ParseDuration
	ValidateAutoClose   = config.ValidateAutoClose
	ValidateMaxLifetime = config.ValidateMaxLifetime
)
