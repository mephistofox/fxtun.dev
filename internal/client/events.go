package client

import (
	"sync"
)

// EventType defines the types of events the client can emit
type EventType string

const (
	EventConnecting    EventType = "connecting"
	EventConnected     EventType = "connected"
	EventDisconnected  EventType = "disconnected"
	EventReconnecting  EventType = "reconnecting"
	EventTunnelCreated EventType = "tunnel_created"
	EventTunnelClosed  EventType = "tunnel_closed"
	EventTunnelError   EventType = "tunnel_error"
	EventTrafficUpdate EventType = "traffic_update"
	EventError         EventType = "error"
	EventLog           EventType = "log"
)

// Event represents a client event with optional payload
type Event struct {
	Type    EventType              `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// EventHandler is a callback function for handling events
type EventHandler func(Event)

// EventEmitter manages event subscriptions and emissions
type EventEmitter struct {
	handlers []EventHandler
	mu       sync.RWMutex
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		handlers: make([]EventHandler, 0),
	}
}

// Subscribe adds an event handler
func (e *EventEmitter) Subscribe(handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, handler)
}

// Emit sends an event to all subscribers
func (e *EventEmitter) Emit(event Event) {
	e.mu.RLock()
	handlers := make([]EventHandler, len(e.handlers))
	copy(handlers, e.handlers)
	e.mu.RUnlock()

	for _, h := range handlers {
		go h(event)
	}
}

// EmitType emits an event with just a type
func (e *EventEmitter) EmitType(eventType EventType) {
	e.Emit(Event{Type: eventType})
}

// EmitWithPayload emits an event with payload
func (e *EventEmitter) EmitWithPayload(eventType EventType, payload map[string]interface{}) {
	e.Emit(Event{Type: eventType, Payload: payload})
}

// EmitError emits an error event
func (e *EventEmitter) EmitError(err error) {
	e.EmitWithPayload(EventError, map[string]interface{}{
		"error": err.Error(),
	})
}

// EmitTunnelCreated emits a tunnel created event
func (e *EventEmitter) EmitTunnelCreated(tunnel *ActiveTunnel) {
	payload := map[string]interface{}{
		"id":         tunnel.ID,
		"name":       tunnel.Config.Name,
		"type":       tunnel.Config.Type,
		"local_port": tunnel.Config.LocalPort,
		"connected":  tunnel.Connected.Format("2006-01-02T15:04:05Z07:00"),
	}
	if tunnel.URL != "" {
		payload["url"] = tunnel.URL
	}
	if tunnel.RemoteAddr != "" {
		payload["remote_addr"] = tunnel.RemoteAddr
	}
	e.EmitWithPayload(EventTunnelCreated, payload)
}

// EmitTunnelClosed emits a tunnel closed event
func (e *EventEmitter) EmitTunnelClosed(tunnelID string) {
	e.EmitWithPayload(EventTunnelClosed, map[string]interface{}{
		"tunnel_id": tunnelID,
	})
}

// EmitLog emits a log event
func (e *EventEmitter) EmitLog(level, message string) {
	e.EmitWithPayload(EventLog, map[string]interface{}{
		"level":   level,
		"message": message,
	})
}

// Clear removes all handlers
func (e *EventEmitter) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = make([]EventHandler, 0)
}
