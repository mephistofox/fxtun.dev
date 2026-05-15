package monitor

import (
	"sync"
	"time"
)

// SlidingWindow implements a sliding-window rate limiter.
// It counts events within a rolling time window and rejects
// new events once the configured limit is reached.
// A limit of 0 means unlimited (Allow always returns true).
type SlidingWindow struct {
	mu      sync.Mutex
	limit   int64
	window  time.Duration
	events  []time.Time
	denied  int64
}

// NewSlidingWindow creates a new sliding-window rate limiter.
// limit is the maximum number of events allowed within window.
// If limit <= 0, rate limiting is disabled (unlimited).
func NewSlidingWindow(limit int64, window time.Duration) *SlidingWindow {
	return &SlidingWindow{
		limit:  limit,
		window: window,
		events: make([]time.Time, 0, 64),
	}
}

// Allow checks whether a new event is permitted under the rate limit.
// Returns true if the event is allowed, false if it exceeds the limit.
func (sw *SlidingWindow) Allow() bool {
	if sw.limit <= 0 {
		return true
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	sw.evict(now)

	if int64(len(sw.events)) >= sw.limit {
		sw.denied++
		return false
	}

	sw.events = append(sw.events, now)
	return true
}

// Count returns the number of events currently within the window.
func (sw *SlidingWindow) Count() int64 {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.evict(time.Now())
	return int64(len(sw.events))
}

// Denied returns the total number of denied events.
func (sw *SlidingWindow) Denied() int64 {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.denied
}

// IsIdle returns true if no events have been recorded within the given duration.
func (sw *SlidingWindow) IsIdle(d time.Duration) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if len(sw.events) == 0 {
		return true
	}
	return time.Since(sw.events[len(sw.events)-1]) > d
}

// evict removes events that have fallen outside the window.
// Must be called with sw.mu held.
func (sw *SlidingWindow) evict(now time.Time) {
	cutoff := now.Add(-sw.window)
	i := 0
	for i < len(sw.events) && sw.events[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		copy(sw.events, sw.events[i:])
		sw.events = sw.events[:len(sw.events)-i]
	}
}
