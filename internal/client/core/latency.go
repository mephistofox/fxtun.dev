package core

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/protocol"
)

const latencyProbeTimeout = 800 * time.Millisecond

// probeLatency dials each candidate in parallel and returns the one with
// lowest successful TCP connect time.
// Returns fastest candidate + its RTT. If all probes fail, returns empty
// candidate and the last error.
func probeLatency(ctx context.Context, candidates []protocol.NodeRedirectCandidate) (protocol.NodeRedirectCandidate, time.Duration, error) {
	if len(candidates) == 0 {
		return protocol.NodeRedirectCandidate{}, 0, fmt.Errorf("no candidates")
	}
	if len(candidates) == 1 {
		return candidates[0], 0, nil
	}

	type result struct {
		idx int
		rtt time.Duration
		err error
	}

	results := make(chan result, len(candidates))
	var wg sync.WaitGroup

	dialer := &net.Dialer{Timeout: latencyProbeTimeout}

	for i, c := range candidates {
		wg.Add(1)
		go func(idx int, addr string) {
			defer wg.Done()
			probeCtx, cancel := context.WithTimeout(ctx, latencyProbeTimeout)
			defer cancel()
			start := time.Now()
			conn, err := dialer.DialContext(probeCtx, "tcp", addr)
			rtt := time.Since(start)
			if err != nil {
				results <- result{idx: idx, err: err}
				return
			}
			conn.Close()
			results <- result{idx: idx, rtt: rtt}
		}(i, c.Addr)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	bestIdx := -1
	bestRTT := time.Duration(0)
	var lastErr error
	for r := range results {
		if r.err != nil {
			lastErr = r.err
			continue
		}
		if bestIdx == -1 || r.rtt < bestRTT {
			bestIdx = r.idx
			bestRTT = r.rtt
		}
	}

	if bestIdx == -1 {
		return protocol.NodeRedirectCandidate{}, 0, fmt.Errorf("all probes failed: %w", lastErr)
	}
	return candidates[bestIdx], bestRTT, nil
}
