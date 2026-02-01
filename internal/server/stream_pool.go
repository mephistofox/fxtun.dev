package server

import (
	"time"

	"github.com/mephistofox/fxtunnel/internal/transport"
)

const streamPoolSize = 24

// OpenStream returns a pre-opened stream from the pool,
// falling back to opening a new one if the pool is empty.
func (c *Client) OpenStream() (transport.Stream, error) {
	select {
	case stream := <-c.streamPool:
		return stream, nil
	default:
		return c.Session.OpenStream(c.ctx)
	}
}

// startStreamPool launches a background goroutine that keeps the stream pool full.
func (c *Client) startStreamPool() {
	c.streamPool = make(chan transport.Stream, streamPoolSize)
	go c.refillStreamPool()
}

func (c *Client) refillStreamPool() {
	for {
		select {
		case <-c.ctx.Done():
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

		if len(c.streamPool) >= streamPoolSize {
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		stream, err := c.Session.OpenStream(c.ctx)
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
