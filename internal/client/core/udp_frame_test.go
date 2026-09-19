package core

import "testing"

// Mirror of the server-side guard: a hostile server can declare a frame length
// larger than the client's payload buffer and panic the client process.
func TestUDPFrameLenValid(t *testing.T) {
	if !udpFrameLenValid(maxUDPPacketSize) {
		t.Errorf("udpFrameLenValid(%d) = false, want true", maxUDPPacketSize)
	}
	if udpFrameLenValid(65535) {
		t.Errorf("udpFrameLenValid(65535) = true, want false: it exceeds the %d-byte buffer", maxUDPPacketSize)
	}
}
