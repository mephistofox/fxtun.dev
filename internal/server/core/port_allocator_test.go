package core

import (
	"sync"
	"testing"

	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAllocator() *PortAllocator {
	return NewPortAllocator(config.PortRange{Min: 10000, Max: 10005})
}

func TestPortAllocator_AllocateSpecific(t *testing.T) {
	a := newTestAllocator()

	port, err := a.Allocate(10002)
	require.NoError(t, err)
	assert.Equal(t, 10002, port)
}

func TestPortAllocator_AllocateAuto(t *testing.T) {
	a := newTestAllocator()

	port, err := a.Allocate(0)
	require.NoError(t, err)
	assert.Equal(t, 10000, port)

	port, err = a.Allocate(0)
	require.NoError(t, err)
	assert.Equal(t, 10001, port)
}

func TestPortAllocator_RangeValidation(t *testing.T) {
	a := newTestAllocator()

	_, err := a.Allocate(9999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside allowed range")

	_, err = a.Allocate(10006)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside allowed range")
}

func TestPortAllocator_ConflictDetection(t *testing.T) {
	a := newTestAllocator()

	_, err := a.Allocate(10000)
	require.NoError(t, err)

	_, err = a.Allocate(10000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in use")
}

func TestPortAllocator_Release(t *testing.T) {
	a := newTestAllocator()

	port, err := a.Allocate(10000)
	require.NoError(t, err)
	assert.Equal(t, 10000, port)

	a.Release(10000)

	port, err = a.Allocate(10000)
	require.NoError(t, err)
	assert.Equal(t, 10000, port)
}

func TestPortAllocator_Exhaustion(t *testing.T) {
	a := newTestAllocator()

	// Allocate all 6 ports (10000-10005)
	for i := 0; i < 6; i++ {
		_, err := a.Allocate(0)
		require.NoError(t, err)
	}

	_, err := a.Allocate(0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no available ports")
}

func TestPortAllocator_ConcurrentAccess(t *testing.T) {
	a := NewPortAllocator(config.PortRange{Min: 10000, Max: 10999})

	var wg sync.WaitGroup
	allocated := make(chan int, 1000)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			port, err := a.Allocate(0)
			if err == nil {
				allocated <- port
			}
		}()
	}

	wg.Wait()
	close(allocated)

	seen := make(map[int]bool)
	for port := range allocated {
		assert.False(t, seen[port], "duplicate port %d", port)
		seen[port] = true
	}
	assert.Len(t, seen, 1000)
}

func TestPortAllocator_AllocateAndBindSkipsRefusedPorts(t *testing.T) {
	a := newTestAllocator()

	var tried []int
	port, err := a.AllocateAndBind(0, func(p int) error {
		tried = append(tried, p)
		if p < 10002 {
			return assert.AnError // OS refuses the port (taken by something else)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 10002, port)
	assert.Equal(t, []int{10000, 10001, 10002}, tried)

	// The ports walked past must be free again once the call returns.
	p, err := a.Allocate(10000)
	require.NoError(t, err)
	assert.Equal(t, 10000, p)
}

func TestPortAllocator_AllocateAndBindKeepsRequestedPortStrict(t *testing.T) {
	a := newTestAllocator()

	calls := 0
	_, err := a.AllocateAndBind(10003, func(int) error {
		calls++
		return assert.AnError
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls, "an explicitly requested port must not silently fall back to another")

	// The failed port is released, not leaked.
	p, err := a.Allocate(10003)
	require.NoError(t, err)
	assert.Equal(t, 10003, p)
}
