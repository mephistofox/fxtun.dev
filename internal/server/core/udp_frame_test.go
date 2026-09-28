package core

import "testing"

// A client controls the 16-bit length field of every UDP frame, but the pooled
// buffer it is read into is only udpHeaderSize+maxUDPPacketSize bytes. Without
// a bound check, frame[:length] panics — in a bare goroutine, which kills the
// whole server and every tenant's tunnels.
func TestUDPFrameLenValid(t *testing.T) {
	fp := udpFramePool.Get().(*[]byte)
	capacity := len(*fp)
	udpFramePool.Put(fp)

	if capacity >= 65535 {
		t.Fatalf("pooled frame is %d bytes; a uint16 length can no longer overflow it, "+
			"so this guard and test need revisiting", capacity)
	}

	if !udpFrameLenValid(maxUDPPacketSize) {
		t.Errorf("udpFrameLenValid(%d) = false, want true: a full-size datagram must pass", maxUDPPacketSize)
	}
	if udpFrameLenValid(65535) {
		t.Errorf("udpFrameLenValid(65535) = true, want false: it exceeds the %d-byte buffer", capacity)
	}
}
