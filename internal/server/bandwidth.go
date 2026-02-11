package server

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

// BandwidthLimiter limits the bandwidth for a client using a token bucket.
type BandwidthLimiter struct {
	limiter *rate.Limiter
}

// NewBandwidthLimiter creates a new bandwidth limiter with the given rate in bytes/sec.
// burst is set to match a reasonable buffer size (256KB) to allow smooth streaming.
func NewBandwidthLimiter(bytesPerSec int) *BandwidthLimiter {
	burst := 256 * 1024 // 256KB burst
	if bytesPerSec < burst {
		burst = bytesPerSec
	}
	return &BandwidthLimiter{
		limiter: rate.NewLimiter(rate.Limit(bytesPerSec), burst),
	}
}

// Reader wraps an io.Reader with bandwidth throttling.
func (b *BandwidthLimiter) Reader(r io.Reader) io.Reader {
	return &throttledReader{r: r, limiter: b.limiter}
}

// Writer wraps an io.Writer with bandwidth throttling.
func (b *BandwidthLimiter) Writer(w io.Writer) io.Writer {
	return &throttledWriter{w: w, limiter: b.limiter}
}

type throttledReader struct {
	r       io.Reader
	limiter *rate.Limiter
}

func (tr *throttledReader) Read(p []byte) (int, error) {
	n, err := tr.r.Read(p)
	if n > 0 {
		_ = tr.limiter.WaitN(context.Background(), n)
	}
	return n, err
}

type throttledWriter struct {
	w       io.Writer
	limiter *rate.Limiter
}

func (tw *throttledWriter) Write(p []byte) (int, error) {
	_ = tw.limiter.WaitN(context.Background(), len(p))
	return tw.w.Write(p)
}
