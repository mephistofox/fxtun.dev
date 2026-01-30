package inspect

import "sync"

// RingBuffer is a fixed-size circular buffer of captured exchanges
// with fan-out subscription support for real-time streaming.
type RingBuffer struct {
	mu          sync.RWMutex
	entries     []*CapturedExchange
	capacity    int
	writeIdx    int
	count       int
	subscribers map[chan *CapturedExchange]struct{}
	closed      bool
}

// NewRingBuffer creates a new ring buffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		entries:     make([]*CapturedExchange, capacity),
		capacity:    capacity,
		subscribers: make(map[chan *CapturedExchange]struct{}),
	}
}

// Add inserts an exchange into the buffer and notifies all subscribers.
func (rb *RingBuffer) Add(ex *CapturedExchange) {
	rb.mu.Lock()
	rb.entries[rb.writeIdx] = ex
	rb.writeIdx = (rb.writeIdx + 1) % rb.capacity
	if rb.count < rb.capacity {
		rb.count++
	}
	// snapshot subscribers
	subs := make([]chan *CapturedExchange, 0, len(rb.subscribers))
	for ch := range rb.subscribers {
		subs = append(subs, ch)
	}
	rb.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- ex:
		default:
		}
	}
}

// List returns exchanges newest-first with pagination (offset, limit).
func (rb *RingBuffer) List(offset, limit int) []*CapturedExchange {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if offset >= rb.count {
		return nil
	}

	available := rb.count - offset
	if limit > available {
		limit = available
	}

	result := make([]*CapturedExchange, limit)
	// newest is at writeIdx-1, second newest at writeIdx-2, etc.
	for i := 0; i < limit; i++ {
		idx := (rb.writeIdx - 1 - offset - i + rb.capacity*2) % rb.capacity
		result[i] = rb.entries[idx]
	}
	return result
}

// Get finds an exchange by ID. Returns nil if not found.
func (rb *RingBuffer) Get(id string) *CapturedExchange {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	for i := 0; i < rb.count; i++ {
		idx := (rb.writeIdx - 1 - i + rb.capacity) % rb.capacity
		if rb.entries[idx].ID == id {
			return rb.entries[idx]
		}
	}
	return nil
}

// Len returns the number of entries currently in the buffer.
func (rb *RingBuffer) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Clear resets the buffer.
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	for i := range rb.entries {
		rb.entries[i] = nil
	}
	rb.writeIdx = 0
	rb.count = 0
}

// Subscribe returns a buffered channel that receives new exchanges.
func (rb *RingBuffer) Subscribe() chan *CapturedExchange {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.closed {
		ch := make(chan *CapturedExchange)
		close(ch)
		return ch
	}

	ch := make(chan *CapturedExchange, 64)
	rb.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber channel.
func (rb *RingBuffer) Unsubscribe(ch chan *CapturedExchange) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	delete(rb.subscribers, ch)
}

// Close closes all subscriber channels and marks the buffer as closed.
func (rb *RingBuffer) Close() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.closed = true
	for ch := range rb.subscribers {
		close(ch)
	}
	rb.subscribers = make(map[chan *CapturedExchange]struct{})
}
