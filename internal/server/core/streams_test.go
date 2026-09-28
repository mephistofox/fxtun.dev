package core

import (
	"net"
	"testing"

	"github.com/hashicorp/yamux"
)

// testSession returns the server side of a yamux session over net.Pipe; the
// peer only accepts, so streams stay open.
func testSession(t *testing.T) *yamux.Session {
	t.Helper()
	a, b := net.Pipe()
	s, err := yamux.Server(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	peer, err := yamux.Client(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			if _, err := peer.Accept(); err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() { s.Close(); peer.Close() })
	return s
}

func TestOpenStreamPicksLeastLoadedSession(t *testing.T) {
	busy, light, idle := testSession(t), testSession(t), testSession(t)
	for range 5 {
		if _, err := busy.Open(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := light.Open(); err != nil {
		t.Fatal(err)
	}
	c := &Client{Session: busy, DataSessions: []*yamux.Session{light, idle}}

	st, err := c.OpenStream()
	if err != nil {
		t.Fatal(err)
	}
	if st.(*yamux.Stream).Session() != idle {
		t.Fatalf("stream opened on session with %d streams, want the idle one", st.(*yamux.Stream).Session().NumStreams()-1)
	}

	idle.Close()
	st, err = c.OpenStream()
	if err != nil {
		t.Fatal(err)
	}
	if st.(*yamux.Stream).Session() != light {
		t.Fatalf("after closing the idle session, stream opened on session with %d streams, want the one with 1", st.(*yamux.Stream).Session().NumStreams()-1)
	}
}

// Server-Timing reads the RTT of the socket that actually carries a request,
// not the control connection's.
func TestSocketOfFindsTheStreamsSession(t *testing.T) {
	primary, data := testSession(t), testSession(t)
	ctrl, dataSock := &net.TCPConn{}, &net.TCPConn{}
	c := &Client{Session: primary, conn: ctrl,
		DataSessions: []*yamux.Session{data}, DataConns: []net.Conn{dataSock}}

	onData, err := data.Open()
	if err != nil {
		t.Fatal(err)
	}
	if c.socketOf(onData) != net.Conn(dataSock) {
		t.Error("stream on a data session: want that session's socket")
	}
	onPrimary, err := primary.Open()
	if err != nil {
		t.Fatal(err)
	}
	if c.socketOf(onPrimary) != net.Conn(ctrl) {
		t.Error("stream on the primary session: want the control socket")
	}
}
