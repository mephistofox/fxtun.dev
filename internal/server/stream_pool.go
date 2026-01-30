package server

import (
	"net"
	"time"
)

const streamPoolSize = 24

// OpenStream returns a pre-opened yamux stream from the pool,
// falling back to opening a new one if the pool is empty.
func (c *Client) OpenStream() (net.Conn, error) {
	// Try pool first (non-blocking)
	select {
	case stream := <-c.streamPool:
		return stream, nil
	default:
		return c.Session.Open()
	}
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
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		stream, err := c.Session.Open()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
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
