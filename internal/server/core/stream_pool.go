package core

import (
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

// streamPoolSize is how many yamux streams are kept open per client ahead of
// demand.
//
// It was 256. Measured against production over a real 88 ms link, that bought
// nothing: warmed-up p50 was 359-409 ms at 256 against 324-335 ms at 32, and a
// cold tunnel answered its first burst in 562 ms against 533 ms — the
// difference is within run-to-run spread, but it never favoured the larger
// pool, including the cold start the pool exists for. The cost was real:
// 256 goroutines and about 1.85 MB per connected client.
const streamPoolSize = 32

// OpenStream returns a pre-opened yamux stream from the pool,
// falling back to opening a new one via round-robin if the pool is empty.
func (c *Client) OpenStream() (net.Conn, error) {
	// Try pool first (non-blocking)
	select {
	case stream := <-c.streamPool:
		return stream, nil
	default:
		return c.openStreamRoundRobin()
	}
}

// openStreamRoundRobin opens a stream from one of the available sessions using round-robin.
func (c *Client) openStreamRoundRobin() (net.Conn, error) {
	sessions := c.allSessions()
	n := uint32(len(sessions)) //nolint:gosec // length is bounded by pool size
	idx := c.sessionIdx.Add(1)
	// Try starting from idx, fall through to others on error
	for i := uint32(0); i < n; i++ {
		s := sessions[(idx+i)%n]
		if s.IsClosed() {
			continue
		}
		stream, err := s.Open()
		if err == nil {
			return stream, nil
		}
	}
	// Last resort: primary session
	return c.Session.Open()
}

// allSessions returns the primary session plus all data sessions.
func (c *Client) allSessions() []*yamux.Session {
	c.DataMu.RLock()
	sessions := make([]*yamux.Session, 0, 1+len(c.DataSessions))
	sessions = append(sessions, c.Session)
	sessions = append(sessions, c.DataSessions...)
	c.DataMu.RUnlock()
	return sessions
}

// startStreamPool launches a background goroutine that keeps the stream pool full.
func (c *Client) startStreamPool() {
	c.streamPool = make(chan net.Conn, streamPoolSize)
	go c.refillStreamPool()
}

func (c *Client) refillStreamPool() {
	for {
		select {
		case <-c.ctx.Done():
			// Drain and close pooled streams
			for {
				select {
				case s := <-c.streamPool:
					s.Close()
				default:
					return
				}
			}
		default:
		}

		// Only refill if pool has room
		if len(c.streamPool) >= streamPoolSize {
			// Pool full, wait a bit
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(5 * time.Millisecond):
			}
			continue
		}

		stream, err := c.openStreamRoundRobin()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(10 * time.Millisecond):
			}
			continue
		}

		select {
		case c.streamPool <- stream:
		case <-c.ctx.Done():
			stream.Close()
			return
		}
	}
}
