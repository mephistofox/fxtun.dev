package server

import (
	"fmt"
	"sync"

	"github.com/mephistofox/fxtunnel/internal/config"
)

// PortAllocator manages port allocation within a configured range.
type PortAllocator struct {
	portRange config.PortRange
	usedPorts map[int]bool
	mu        sync.Mutex
}

// NewPortAllocator creates a new PortAllocator for the given range.
func NewPortAllocator(portRange config.PortRange) *PortAllocator {
	return &PortAllocator{
		portRange: portRange,
		usedPorts: make(map[int]bool),
	}
}

// Allocate reserves a port. If requested is 0, the first available port in the
// range is returned. Returns the allocated port number or an error.
func (a *PortAllocator) Allocate(requested int) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if requested != 0 {
		if requested < a.portRange.Min || requested > a.portRange.Max {
			return 0, fmt.Errorf("port %d is outside allowed range (%d-%d)",
				requested, a.portRange.Min, a.portRange.Max)
		}
		if a.usedPorts[requested] {
			return 0, fmt.Errorf("port %d is already in use", requested)
		}
		a.usedPorts[requested] = true
		return requested, nil
	}

	// Auto-assign
	for port := a.portRange.Min; port <= a.portRange.Max; port++ {
		if a.usedPorts[port] {
			continue
		}
		a.usedPorts[port] = true
		return port, nil
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", a.portRange.Min, a.portRange.Max)
}

// Release frees a previously allocated port.
func (a *PortAllocator) Release(port int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.usedPorts, port)
}
