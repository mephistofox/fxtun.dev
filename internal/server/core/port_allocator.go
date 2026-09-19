package core

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

// maxBindRetries bounds how many ports AllocateAndBind walks past before it
// gives up when the OS refuses to bind them.
const maxBindRetries = 8

// AllocateAndBind reserves a port and hands it to bind. A port free in the
// allocator's bookkeeping can still be refused by the OS — an unrelated process
// or an outbound connection may hold it when the configured range overlaps the
// ephemeral port range. When no specific port was requested, walk on to the next
// candidate instead of failing the whole tunnel request; ports whose bind failed
// stay reserved until the call returns so Allocate does not hand them back.
func (a *PortAllocator) AllocateAndBind(requested int, bind func(port int) error) (int, error) {
	var failed []int
	defer func() {
		for _, p := range failed {
			a.Release(p)
		}
	}()

	for {
		port, err := a.Allocate(requested)
		if err != nil {
			return 0, err
		}
		bindErr := bind(port)
		if bindErr == nil {
			return port, nil
		}
		if requested != 0 || len(failed) >= maxBindRetries {
			a.Release(port)
			return 0, fmt.Errorf("failed to bind port %d: %w", port, bindErr)
		}
		failed = append(failed, port)
	}
}

// Release frees a previously allocated port.
func (a *PortAllocator) Release(port int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.usedPorts, port)
}
