package inspect

import "sync"

// Store is an interface for persistent exchange storage.
type Store interface {
	Save(ex *CapturedExchange, userID int64) error
	ListByTunnelID(tunnelID string, offset, limit int) ([]*CapturedExchange, int, error)
	ListByHostAndUser(host string, userID int64, offset, limit int) ([]*CapturedExchange, int, error)
	GetByID(id string) (*CapturedExchange, error)
	DeleteByTunnelID(tunnelID string) (int64, error)
}

type persistJob struct {
	exchange *CapturedExchange
	userID   int64
}

// Manager manages per-tunnel RingBuffers.
type Manager struct {
	mu          sync.RWMutex
	buffers     map[string]*RingBuffer
	userIDs     map[string]int64
	capacity    int
	maxBodySize int
	store       Store
	persistCh   chan persistJob
}

// NewManager creates a new Manager. If capacity is 0, inspection is disabled.
func NewManager(capacity, maxBodySize int) *Manager {
	return &Manager{
		buffers:     make(map[string]*RingBuffer),
		userIDs:     make(map[string]int64),
		capacity:    capacity,
		maxBodySize: maxBodySize,
	}
}

// SetStore sets the persistent store and starts the background persist goroutine.
func (m *Manager) SetStore(store Store) {
	m.store = store
	m.persistCh = make(chan persistJob, 256)
	go m.persistLoop()
}

func (m *Manager) persistLoop() {
	for job := range m.persistCh {
		_ = m.store.Save(job.exchange, job.userID)
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

// GetOrCreateWithUser returns the RingBuffer for the given tunnel ID and tracks the user ID.
func (m *Manager) GetOrCreateWithUser(tunnelID string, userID int64) *RingBuffer {
	buf := m.GetOrCreate(tunnelID)
	if buf != nil {
		m.mu.Lock()
		m.userIDs[tunnelID] = userID
		m.mu.Unlock()
	}
	return buf
}

// Get returns the RingBuffer for the given tunnel ID, or nil if not found.
func (m *Manager) Get(tunnelID string) *RingBuffer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.buffers[tunnelID]
}

// AddAndPersist adds the exchange to the in-memory buffer and enqueues async DB write.
func (m *Manager) AddAndPersist(tunnelID string, ex *CapturedExchange) {
	buf := m.Get(tunnelID)
	if buf != nil {
		buf.Add(ex)
	}

	if m.store == nil || m.persistCh == nil {
		return
	}
	m.mu.RLock()
	userID := m.userIDs[tunnelID]
	m.mu.RUnlock()
	if userID == 0 {
		return
	}

	select {
	case m.persistCh <- persistJob{exchange: ex, userID: userID}:
	default:
		// Channel full, drop persistence for this exchange
	}
}

// ListPersisted delegates to the store for DB-backed listing.
func (m *Manager) ListPersisted(tunnelID string, offset, limit int) ([]*CapturedExchange, int, error) {
	if m.store == nil {
		return nil, 0, nil
	}
	return m.store.ListByTunnelID(tunnelID, offset, limit)
}

// ListPersistedByHostAndUser delegates to the store for host+user-based DB listing.
func (m *Manager) ListPersistedByHostAndUser(host string, userID int64, offset, limit int) ([]*CapturedExchange, int, error) {
	if m.store == nil {
		return nil, 0, nil
	}
	return m.store.ListByHostAndUser(host, userID, offset, limit)
}

// GetPersisted delegates to the store for DB-backed retrieval.
func (m *Manager) GetPersisted(id string) (*CapturedExchange, error) {
	if m.store == nil {
		return nil, nil
	}
	return m.store.GetByID(id)
}

// Remove closes and removes the buffer for the given tunnel ID.
func (m *Manager) Remove(tunnelID string) {
	m.mu.Lock()
	buf, ok := m.buffers[tunnelID]
	if ok {
		delete(m.buffers, tunnelID)
	}
	delete(m.userIDs, tunnelID)
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
	m.userIDs = make(map[string]int64)
	m.mu.Unlock()
	for _, buf := range buffers {
		buf.Close()
	}
	if m.persistCh != nil {
		close(m.persistCh)
	}
}
