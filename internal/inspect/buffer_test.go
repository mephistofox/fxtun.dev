package inspect

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeExchange(id string) *CapturedExchange {
	return &CapturedExchange{ID: id, Timestamp: time.Now()}
}

func TestRingBuffer_AddAndList(t *testing.T) {
	rb := NewRingBuffer(3)
	rb.Add(makeExchange("a"))
	rb.Add(makeExchange("b"))
	rb.Add(makeExchange("c"))

	list := rb.List(0, 10)
	require.Len(t, list, 3)
	assert.Equal(t, "c", list[0].ID)
	assert.Equal(t, "b", list[1].ID)
	assert.Equal(t, "a", list[2].ID)
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer(2)
	rb.Add(makeExchange("a"))
	rb.Add(makeExchange("b"))
	rb.Add(makeExchange("c"))

	assert.Equal(t, 2, rb.Len())
	list := rb.List(0, 10)
	require.Len(t, list, 2)
	assert.Equal(t, "c", list[0].ID)
	assert.Equal(t, "b", list[1].ID)
}

func TestRingBuffer_Pagination(t *testing.T) {
	rb := NewRingBuffer(10)
	for i := 0; i < 5; i++ {
		rb.Add(makeExchange(fmt.Sprintf("e%d", i)))
	}

	// newest first: e4, e3, e2, e1, e0
	// offset=1, limit=2 → e3, e2
	list := rb.List(1, 2)
	require.Len(t, list, 2)
	assert.Equal(t, "e3", list[0].ID)
	assert.Equal(t, "e2", list[1].ID)
}

func TestRingBuffer_GetByID(t *testing.T) {
	rb := NewRingBuffer(5)
	rb.Add(makeExchange("x"))
	rb.Add(makeExchange("y"))

	assert.NotNil(t, rb.Get("x"))
	assert.Equal(t, "x", rb.Get("x").ID)
	assert.Nil(t, rb.Get("z"))
}

func TestRingBuffer_Subscribe(t *testing.T) {
	rb := NewRingBuffer(10)
	ch := rb.Subscribe()
	defer rb.Unsubscribe(ch)

	go func() {
		rb.Add(makeExchange("s1"))
	}()

	select {
	case ex := <-ch:
		assert.Equal(t, "s1", ex.ID)
	case <-time.After(time.Second):
		t.Fatal("did not receive exchange within 1 second")
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer(5)
	rb.Add(makeExchange("a"))
	rb.Add(makeExchange("b"))
	assert.Equal(t, 2, rb.Len())

	rb.Clear()
	assert.Equal(t, 0, rb.Len())
}

func TestRingBuffer_Close(t *testing.T) {
	rb := NewRingBuffer(5)
	ch := rb.Subscribe()
	rb.Close()

	_, ok := <-ch
	assert.False(t, ok, "channel should be closed")
}
