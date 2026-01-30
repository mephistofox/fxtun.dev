package inspect

import "sync"

// Manager manages per-tunnel RingBuffers.
type Manager struct {
	mu          sync.RWMutex
	buffers     map[string]*RingBuffer
	capacity    int
	maxBodySize int
}

// NewManager creates a new Manager. If capacity is 0, inspection is disabled.
func NewManager(capacity, maxBodySize int) *Manager {
	return &Manager{
		buffers:     make(map[string]*RingBuffer),
		capacity:    capacity,
		maxBodySize: maxBodySize,
	}
}

// Enabled returns true if inspection is enabled (capacity > 0).
func (m *Manager) Enabled() bool {
	return m.capacity > 0
}

// MaxBodySize returns the maximum body size to capture.
func (m *Manager) MaxBodySize() int {
	return m.maxBodySize
}

// GetOrCreate returns the RingBuffer for the given tunnel ID, creating one if needed.
// Returns nil if the manager is disabled.
func (m *Manager) GetOrCreate(tunnelID string) *RingBuffer {
	if !m.Enabled() {
		return nil
	}

	// Fast path: read lock.
	m.mu.RLock()
	buf, ok := m.buffers[tunnelID]
	m.mu.RUnlock()
	if ok {
		return buf
	}

	// Slow path: write lock with double-check.
	m.mu.Lock()
	defer m.mu.Unlock()
	if buf, ok = m.buffers[tunnelID]; ok {
		return buf
	}
	buf = NewRingBuffer(m.capacity)
	m.buffers[tunnelID] = buf
	return buf
}

// Get returns the RingBuffer for the given tunnel ID, or nil if not found.
func (m *Manager) Get(tunnelID string) *RingBuffer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.buffers[tunnelID]
}

// Remove closes and removes the buffer for the given tunnel ID.
func (m *Manager) Remove(tunnelID string) {
	m.mu.Lock()
	buf, ok := m.buffers[tunnelID]
	if ok {
		delete(m.buffers, tunnelID)
	}
	m.mu.Unlock()
	if ok {
		buf.Close()
	}
}

// Close closes all buffers and clears the map.
func (m *Manager) Close() {
	m.mu.Lock()
	buffers := m.buffers
	m.buffers = make(map[string]*RingBuffer)
	m.mu.Unlock()
	for _, buf := range buffers {
		buf.Close()
	}
}
