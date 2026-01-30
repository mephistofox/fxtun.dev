package inspect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_CreateAndGet(t *testing.T) {
	m := NewManager(64, 4096)
	require.True(t, m.Enabled())

	buf1 := m.GetOrCreate("tunnel-1")
	require.NotNil(t, buf1)

	buf2 := m.GetOrCreate("tunnel-1")
	require.NotNil(t, buf2)

	assert.Same(t, buf1, buf2, "GetOrCreate should return the same pointer")

	buf3 := m.Get("tunnel-1")
	assert.Same(t, buf1, buf3, "Get should return the same pointer")

	assert.Nil(t, m.Get("nonexistent"))
}

func TestManager_Remove(t *testing.T) {
	m := NewManager(64, 4096)

	buf1 := m.GetOrCreate("tunnel-1")
	require.NotNil(t, buf1)

	m.Remove("tunnel-1")
	assert.Nil(t, m.Get("tunnel-1"))

	buf2 := m.GetOrCreate("tunnel-1")
	require.NotNil(t, buf2)
	assert.NotSame(t, buf1, buf2, "after Remove, GetOrCreate should return a fresh buffer")
	assert.Equal(t, 0, buf2.Len())
}

func TestManager_Disabled(t *testing.T) {
	m := NewManager(0, 4096)
	assert.False(t, m.Enabled())
	assert.Nil(t, m.GetOrCreate("tunnel-1"))
}
