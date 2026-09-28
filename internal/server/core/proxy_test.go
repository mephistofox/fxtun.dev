package core

import "testing"

// One copy write becomes one yamux frame; a frame this size keeps other
// streams' frames from queuing behind it for long.
func TestProxyBufPoolChunkSize(t *testing.T) {
	buf := proxyBufPool.Get().(*[]byte)
	defer proxyBufPool.Put(buf)
	if len(*buf) != 64<<10 {
		t.Fatalf("expected buffer size %d, got %d", 64<<10, len(*buf))
	}
}
