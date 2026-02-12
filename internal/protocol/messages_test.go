package protocol

import (
	"encoding/json"
	"testing"
)

func TestAuthResultCapabilities(t *testing.T) {
	orig := AuthResultMessage{
		Message:  NewMessage(MsgAuthResult),
		Success:  true,
		ClientID: "client-123",
		Capabilities: &ClientCapabilities{
			InspectorEnabled: true,
			MaxBodySize:      1024 * 1024,
			MaxBufferEntries: 500,
		},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded AuthResultMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Capabilities == nil {
		t.Fatal("expected capabilities to be present after round-trip")
	}
	if !decoded.Capabilities.InspectorEnabled {
		t.Error("expected InspectorEnabled to be true")
	}
	if decoded.Capabilities.MaxBodySize != 1024*1024 {
		t.Errorf("MaxBodySize = %d, want %d", decoded.Capabilities.MaxBodySize, 1024*1024)
	}
	if decoded.Capabilities.MaxBufferEntries != 500 {
		t.Errorf("MaxBufferEntries = %d, want 500", decoded.Capabilities.MaxBufferEntries)
	}
	if decoded.Success != orig.Success {
		t.Errorf("Success = %v, want %v", decoded.Success, orig.Success)
	}
	if decoded.ClientID != orig.ClientID {
		t.Errorf("ClientID = %q, want %q", decoded.ClientID, orig.ClientID)
	}
}

func TestAuthResultCapabilitiesNil(t *testing.T) {
	orig := AuthResultMessage{
		Message:  NewMessage(MsgAuthResult),
		Success:  true,
		ClientID: "client-456",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify the "capabilities" key is absent from the JSON payload (omitempty).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, found := raw["capabilities"]; found {
		t.Error("expected capabilities key to be absent when nil (omitempty)")
	}

	var decoded AuthResultMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Capabilities != nil {
		t.Errorf("expected nil capabilities, got %+v", decoded.Capabilities)
	}
	if decoded.Success != orig.Success {
		t.Errorf("Success = %v, want %v", decoded.Success, orig.Success)
	}
	if decoded.ClientID != orig.ClientID {
		t.Errorf("ClientID = %q, want %q", decoded.ClientID, orig.ClientID)
	}
}
