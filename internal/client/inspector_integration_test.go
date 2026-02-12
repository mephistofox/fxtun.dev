package client

import (
	"testing"

	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/protocol"
	"github.com/rs/zerolog"
)

func TestClientInspectorLifecycle(t *testing.T) {
	cfg := &config.ClientConfig{
		Server: config.ClientServerSettings{Address: "localhost:4443", Token: "test"},
		Inspect: config.InspectSettings{
			Enabled:     true,
			Addr:        "127.0.0.1:0",
			MaxBodySize: 262144,
			MaxEntries:  100,
		},
	}

	// Test: capabilities enable inspector
	c := New(cfg, zerolog.Nop())
	caps := &protocol.ClientCapabilities{InspectorEnabled: true}
	c.applyCapabilities(caps)
	if c.inspector == nil {
		t.Fatal("inspector should be created when capabilities allow")
	}
	if c.inspectMgr == nil {
		t.Fatal("inspect manager should be created")
	}

	// Test: capabilities deny inspector
	c2 := New(cfg, zerolog.Nop())
	c2.applyCapabilities(&protocol.ClientCapabilities{InspectorEnabled: false})
	if c2.inspector != nil {
		t.Error("inspector should be nil when plan denies")
	}

	// Test: nil capabilities (old server)
	c3 := New(cfg, zerolog.Nop())
	c3.applyCapabilities(nil)
	if c3.inspector != nil {
		t.Error("inspector should be nil for old servers")
	}

	// Test: config disabled
	cfgDisabled := &config.ClientConfig{
		Server: config.ClientServerSettings{Address: "localhost:4443", Token: "test"},
		Inspect: config.InspectSettings{
			Enabled: false,
			Addr:    "127.0.0.1:0",
		},
	}
	c4 := New(cfgDisabled, zerolog.Nop())
	c4.applyCapabilities(&protocol.ClientCapabilities{InspectorEnabled: true})
	if c4.inspector != nil {
		t.Error("inspector should be nil when config disables it")
	}
}

func TestClientInspectorAddr(t *testing.T) {
	cfg := &config.ClientConfig{
		Server: config.ClientServerSettings{Address: "localhost:4443", Token: "test"},
		Inspect: config.InspectSettings{
			Enabled:     true,
			Addr:        "127.0.0.1:0",
			MaxBodySize: 262144,
			MaxEntries:  100,
		},
	}

	// No inspector — empty addr
	c := New(cfg, zerolog.Nop())
	if addr := c.InspectorAddr(); addr != "" {
		t.Errorf("expected empty addr, got %q", addr)
	}

	// With inspector — still empty until started
	c.applyCapabilities(&protocol.ClientCapabilities{InspectorEnabled: true})
	if addr := c.InspectorAddr(); addr != "" {
		t.Errorf("expected empty addr before Start, got %q", addr)
	}
}

func TestCapabilitiesOverrideConfig(t *testing.T) {
	cfg := &config.ClientConfig{
		Server: config.ClientServerSettings{Address: "localhost:4443", Token: "test"},
		Inspect: config.InspectSettings{
			Enabled:     true,
			Addr:        "127.0.0.1:0",
			MaxBodySize: 262144,
			MaxEntries:  1000,
		},
	}

	// Server can override max body size and entries
	c := New(cfg, zerolog.Nop())
	c.applyCapabilities(&protocol.ClientCapabilities{
		InspectorEnabled: true,
		MaxBodySize:      512000,
		MaxBufferEntries: 500,
	})

	if c.inspectMgr == nil {
		t.Fatal("inspect manager should exist")
	}
	if c.inspectMgr.MaxBodySize() != 512000 {
		t.Errorf("maxBodySize = %d, want 512000", c.inspectMgr.MaxBodySize())
	}
}
