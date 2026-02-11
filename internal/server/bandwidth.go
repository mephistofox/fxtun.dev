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
// The context allows cancellation when the request is done.
func (b *BandwidthLimiter) Reader(ctx context.Context, r io.Reader) io.Reader {
	return &throttledReader{r: r, limiter: b.limiter, ctx: ctx}
}

// Writer wraps an io.Writer with bandwidth throttling.
// The context allows cancellation when the request is done.
func (b *BandwidthLimiter) Writer(ctx context.Context, w io.Writer) io.Writer {
	return &throttledWriter{w: w, limiter: b.limiter, ctx: ctx}
}

type throttledReader struct {
	r       io.Reader
	limiter *rate.Limiter
	ctx     context.Context
}

func (tr *throttledReader) Read(p []byte) (int, error) {
	n, err := tr.r.Read(p)
	if n > 0 {
		_ = tr.limiter.WaitN(tr.ctx, n)
	}
	return n, err
}

type throttledWriter struct {
	w       io.Writer
	limiter *rate.Limiter
	ctx     context.Context
}

func (tw *throttledWriter) Write(p []byte) (int, error) {
	if err := tw.limiter.WaitN(tw.ctx, len(p)); err != nil {
		return 0, err
	}
	return tw.w.Write(p)
}
