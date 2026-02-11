package client

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventEmitter_SubscribeAndEmit(t *testing.T) {
	emitter := NewEventEmitter()
	var received Event
	var wg sync.WaitGroup
	wg.Add(1)
	emitter.Subscribe(func(e Event) {
		received = e
		wg.Done()
	})
	evt := Event{Type: EventConnected, Payload: map[string]interface{}{"key": "val"}}
	emitter.Emit(evt)
	wg.Wait()
	assert.Equal(t, EventConnected, received.Type)
	assert.Equal(t, "val", received.Payload["key"])
}

func TestEventEmitter_MultipleSubscribers(t *testing.T) {
	emitter := NewEventEmitter()
	var wg sync.WaitGroup
	var mu sync.Mutex
	count := 0
	for i := 0; i < 3; i++ {
		wg.Add(1)
		emitter.Subscribe(func(e Event) {
			mu.Lock()
			count++
			mu.Unlock()
			wg.Done()
		})
	}
	emitter.EmitType(EventConnecting)
	wg.Wait()
	assert.Equal(t, 3, count)
}

func TestEventEmitter_EmitType(t *testing.T) {
	emitter := NewEventEmitter()
	var received Event
	var wg sync.WaitGroup
	wg.Add(1)
	emitter.Subscribe(func(e Event) {
		received = e
		wg.Done()
	})
	emitter.EmitType(EventDisconnected)
	wg.Wait()
	assert.Equal(t, EventDisconnected, received.Type)
	assert.Nil(t, received.Payload)
}

func TestEventEmitter_EmitWithPayload(t *testing.T) {
	emitter := NewEventEmitter()
	var received Event
	var wg sync.WaitGroup
	wg.Add(1)
	emitter.Subscribe(func(e Event) {
		received = e
		wg.Done()
	})
	payload := map[string]interface{}{"foo": "bar", "num": 42}
	emitter.EmitWithPayload(EventLog, payload)
	wg.Wait()
	assert.Equal(t, EventLog, received.Type)
	assert.Equal(t, "bar", received.Payload["foo"])
	assert.Equal(t, 42, received.Payload["num"])
}

func TestEventEmitter_EmitError(t *testing.T) {
	emitter := NewEventEmitter()
	var received Event
	var wg sync.WaitGroup
	wg.Add(1)
	emitter.Subscribe(func(e Event) {
		received = e
		wg.Done()
	})
	emitter.EmitError(errors.New("something failed"))
	wg.Wait()
	assert.Equal(t, EventError, received.Type)
	assert.Equal(t, "something failed", received.Payload["error"])
}

func TestEventEmitter_EmitTunnelCreated(t *testing.T) {
	emitter := NewEventEmitter()
	var received Event
	var wg sync.WaitGroup
	wg.Add(1)
	emitter.Subscribe(func(e Event) {
		received = e
		wg.Done()
	})
	now := time.Now()
	tunnel := &ActiveTunnel{
		ID: "t-123",
		Config: config.TunnelConfig{
			Name:      "web",
			Type:      "http",
			LocalPort: 3000,
		},
		URL:        "https://web.example.com",
		RemoteAddr: "1.2.3.4:4443",
		Connected:  now,
	}
	emitter.EmitTunnelCreated(tunnel)
	wg.Wait()
	require.Equal(t, EventTunnelCreated, received.Type)
	assert.Equal(t, "t-123", received.Payload["id"])
	assert.Equal(t, "web", received.Payload["name"])
	assert.Equal(t, "http", received.Payload["type"])
	assert.Equal(t, 3000, received.Payload["local_port"])
	assert.Equal(t, "https://web.example.com", received.Payload["url"])
	assert.Equal(t, "1.2.3.4:4443", received.Payload["remote_addr"])
	assert.Equal(t, now.Format("2006-01-02T15:04:05Z07:00"), received.Payload["connected"])
}

func TestEventEmitter_EmitTunnelClosed(t *testing.T) {
	emitter := NewEventEmitter()
	var received Event
	var wg sync.WaitGroup
	wg.Add(1)
	emitter.Subscribe(func(e Event) {
		received = e
		wg.Done()
	})
	emitter.EmitTunnelClosed("t-456")
	wg.Wait()
	assert.Equal(t, EventTunnelClosed, received.Type)
	assert.Equal(t, "t-456", received.Payload["tunnel_id"])
}

func TestEventEmitter_EmitLog(t *testing.T) {
	emitter := NewEventEmitter()
	var received Event
	var wg sync.WaitGroup
	wg.Add(1)
	emitter.Subscribe(func(e Event) {
		received = e
		wg.Done()
	})
	emitter.EmitLog("info", "hello world")
	wg.Wait()
	assert.Equal(t, EventLog, received.Type)
	assert.Equal(t, "info", received.Payload["level"])
	assert.Equal(t, "hello world", received.Payload["message"])
}

func TestEventEmitter_Clear(t *testing.T) {
	emitter := NewEventEmitter()
	called := false
	emitter.Subscribe(func(e Event) {
		called = true
	})
	emitter.Clear()
	emitter.EmitType(EventConnected)
	time.Sleep(50 * time.Millisecond)
	assert.False(t, called)
}

func TestEventEmitter_ConcurrentSafety(t *testing.T) {
	emitter := NewEventEmitter()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			emitter.Subscribe(func(e Event) {})
		}()
		go func() {
			defer wg.Done()
			emitter.EmitType(EventConnecting)
		}()
	}
	wg.Wait()
}
