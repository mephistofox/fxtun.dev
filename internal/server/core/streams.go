package core

import (
	"net"
	"sort"

	"github.com/hashicorp/yamux"
)

// OpenStream opens a yamux stream on the open session carrying the fewest
// streams, falling back to the next one on error.
//
// Streams used to come from a pre-opened pool filled round-robin, blind to
// load, so an SSE stream could land on the session busy with a download. The
// pool bought nothing measurable (see 67fc1f9), and Session.Open does not wait
// for the peer.
func (c *Client) OpenStream() (net.Conn, error) {
	sessions := c.allSessions()
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].NumStreams() < sessions[j].NumStreams()
	})
	for _, s := range sessions {
		if s.IsClosed() {
			continue
		}
		if stream, err := s.Open(); err == nil {
			return stream, nil
		}
	}
	// Last resort: primary session
	return c.Session.Open()
}

// socketOf returns the TCP connection under the session carrying stream, so
// Server-Timing reads the RTT of the socket the request actually took.
func (c *Client) socketOf(stream net.Conn) net.Conn {
	if ys, ok := stream.(*yamux.Stream); ok {
		c.DataMu.RLock()
		defer c.DataMu.RUnlock()
		for i, s := range c.DataSessions {
			if s == ys.Session() && i < len(c.DataConns) {
				return c.DataConns[i]
			}
		}
	}
	return c.conn
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
