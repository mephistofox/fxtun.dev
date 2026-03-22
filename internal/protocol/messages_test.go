package protocol

import (
	"bytes"
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

func TestTunnelRequestSecurityFieldsRoundTrip(t *testing.T) {
	orig := TunnelRequestMessage{
		Message:       NewMessage(MsgTunnelRequest),
		TunnelType:    TunnelHTTP,
		Name:          "my-tunnel",
		Subdomain:     "test",
		LocalPort:     8080,
		BasicAuthHash: "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012",
		AllowIPs:      []string{"10.0.0.0/8", "192.168.1.1", "2001:db8::/32"},
		AutoClose:     "30m",
		MaxLifetime:   "8h",
	}

	// Test JSON round-trip
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded TunnelRequestMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.BasicAuthHash != orig.BasicAuthHash {
		t.Errorf("BasicAuthHash = %q, want %q", decoded.BasicAuthHash, orig.BasicAuthHash)
	}
	if len(decoded.AllowIPs) != len(orig.AllowIPs) {
		t.Fatalf("AllowIPs length = %d, want %d", len(decoded.AllowIPs), len(orig.AllowIPs))
	}
	for i, ip := range decoded.AllowIPs {
		if ip != orig.AllowIPs[i] {
			t.Errorf("AllowIPs[%d] = %q, want %q", i, ip, orig.AllowIPs[i])
		}
	}
	if decoded.AutoClose != orig.AutoClose {
		t.Errorf("AutoClose = %q, want %q", decoded.AutoClose, orig.AutoClose)
	}
	if decoded.MaxLifetime != orig.MaxLifetime {
		t.Errorf("MaxLifetime = %q, want %q", decoded.MaxLifetime, orig.MaxLifetime)
	}

	// Test Codec round-trip
	var buf bytes.Buffer
	codec := NewCodec(&buf, &buf)

	if err := codec.Encode(&orig); err != nil {
		t.Fatalf("codec encode: %v", err)
	}

	var codecDecoded TunnelRequestMessage
	if err := codec.Decode(&codecDecoded); err != nil {
		t.Fatalf("codec decode: %v", err)
	}

	if codecDecoded.BasicAuthHash != orig.BasicAuthHash {
		t.Errorf("codec BasicAuthHash = %q, want %q", codecDecoded.BasicAuthHash, orig.BasicAuthHash)
	}
	if len(codecDecoded.AllowIPs) != len(orig.AllowIPs) {
		t.Fatalf("codec AllowIPs length = %d, want %d", len(codecDecoded.AllowIPs), len(orig.AllowIPs))
	}
	for i, ip := range codecDecoded.AllowIPs {
		if ip != orig.AllowIPs[i] {
			t.Errorf("codec AllowIPs[%d] = %q, want %q", i, ip, orig.AllowIPs[i])
		}
	}
	if codecDecoded.AutoClose != orig.AutoClose {
		t.Errorf("codec AutoClose = %q, want %q", codecDecoded.AutoClose, orig.AutoClose)
	}
	if codecDecoded.MaxLifetime != orig.MaxLifetime {
		t.Errorf("codec MaxLifetime = %q, want %q", codecDecoded.MaxLifetime, orig.MaxLifetime)
	}
}

func TestTunnelRequestSecurityFieldsOmitempty(t *testing.T) {
	orig := TunnelRequestMessage{
		Message:    NewMessage(MsgTunnelRequest),
		TunnelType: TunnelHTTP,
		Subdomain:  "test",
		LocalPort:  8080,
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	for _, key := range []string{"basic_auth_hash", "allow_ips", "auto_close", "max_lifetime"} {
		if _, found := raw[key]; found {
			t.Errorf("expected %q key to be absent when empty (omitempty)", key)
		}
	}
}

func TestTunnelCreatedSecurityFieldsRoundTrip(t *testing.T) {
	orig := TunnelCreatedMessage{
		Message:          NewMessage(MsgTunnelCreated),
		TunnelID:         "t1",
		TunnelType:       TunnelHTTP,
		Name:             "my-tunnel",
		URL:              "http://test.example.com",
		Subdomain:        "test",
		BasicAuthEnabled: true,
		AllowIPsCount:    3,
		AutoClose:        "30m",
		MaxLifetime:      "8h",
	}

	// Test JSON round-trip
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded TunnelCreatedMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.BasicAuthEnabled != orig.BasicAuthEnabled {
		t.Errorf("BasicAuthEnabled = %v, want %v", decoded.BasicAuthEnabled, orig.BasicAuthEnabled)
	}
	if decoded.AllowIPsCount != orig.AllowIPsCount {
		t.Errorf("AllowIPsCount = %d, want %d", decoded.AllowIPsCount, orig.AllowIPsCount)
	}
	if decoded.AutoClose != orig.AutoClose {
		t.Errorf("AutoClose = %q, want %q", decoded.AutoClose, orig.AutoClose)
	}
	if decoded.MaxLifetime != orig.MaxLifetime {
		t.Errorf("MaxLifetime = %q, want %q", decoded.MaxLifetime, orig.MaxLifetime)
	}

	// Test Codec round-trip
	var buf bytes.Buffer
	codec := NewCodec(&buf, &buf)

	if err := codec.Encode(&orig); err != nil {
		t.Fatalf("codec encode: %v", err)
	}

	var codecDecoded TunnelCreatedMessage
	if err := codec.Decode(&codecDecoded); err != nil {
		t.Fatalf("codec decode: %v", err)
	}

	if codecDecoded.BasicAuthEnabled != orig.BasicAuthEnabled {
		t.Errorf("codec BasicAuthEnabled = %v, want %v", codecDecoded.BasicAuthEnabled, orig.BasicAuthEnabled)
	}
	if codecDecoded.AllowIPsCount != orig.AllowIPsCount {
		t.Errorf("codec AllowIPsCount = %d, want %d", codecDecoded.AllowIPsCount, orig.AllowIPsCount)
	}
	if codecDecoded.AutoClose != orig.AutoClose {
		t.Errorf("codec AutoClose = %q, want %q", codecDecoded.AutoClose, orig.AutoClose)
	}
	if codecDecoded.MaxLifetime != orig.MaxLifetime {
		t.Errorf("codec MaxLifetime = %q, want %q", codecDecoded.MaxLifetime, orig.MaxLifetime)
	}
}

func TestTunnelCreatedSecurityFieldsOmitempty(t *testing.T) {
	orig := TunnelCreatedMessage{
		Message:    NewMessage(MsgTunnelCreated),
		TunnelID:   "t1",
		TunnelType: TunnelHTTP,
		Subdomain:  "test",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	for _, key := range []string{"basic_auth_enabled", "allow_ips_count", "auto_close", "max_lifetime"} {
		if _, found := raw[key]; found {
			t.Errorf("expected %q key to be absent when zero/empty (omitempty)", key)
		}
	}
}
