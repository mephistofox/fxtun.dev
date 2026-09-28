package core

import (
	"net"
	"runtime"
	"testing"
	"time"
)

func TestServerTiming(t *testing.T) {
	ms := time.Millisecond
	for _, tc := range []struct {
		name                  string
		edge, send, wait, rtt time.Duration
		rttKnown              bool
		want                  string
	}{
		{"split", 400 * time.Microsecond, 0, 221 * ms, 41 * ms, true,
			`fxt-edge;dur=0.4;desc="fxTunnel edge", fxt-tunnel;dur=41.0;desc="tunnel to your machine", fxt-app;dur=180.0;desc="your app"`},
		// RTT jitter can exceed a near-instant app's wait; the app share is never negative.
		{"clamp", ms, 0, 30 * ms, 40 * ms, true,
			`fxt-edge;dur=1.0;desc="fxTunnel edge", fxt-tunnel;dur=40.0;desc="tunnel to your machine", fxt-app;dur=0.0;desc="your app"`},
		// Sending a request body is tunnel work, not the app's.
		{"upload", ms, 500 * ms, 221 * ms, 41 * ms, true,
			`fxt-edge;dur=1.0;desc="fxTunnel edge", fxt-tunnel;dur=541.0;desc="tunnel to your machine", fxt-app;dur=180.0;desc="your app"`},
		// Without an RTT the wait cannot be split.
		{"no rtt", ms, 5 * ms, 30 * ms, 0, false,
			`fxt-edge;dur=1.0;desc="fxTunnel edge", fxt-upstream;dur=35.0;desc="tunnel + your app"`},
	} {
		if got := serverTiming(tc.edge, tc.send, tc.wait, tc.rtt, tc.rttKnown); got != tc.want {
			t.Errorf("%s:\ngot  %s\nwant %s", tc.name, got, tc.want)
		}
	}
}

func TestSessionRTTReadsTCPSocket(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("TCP_INFO is read on linux only")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		if c, err := ln.Accept(); err == nil {
			defer c.Close()
			_, _ = c.Write([]byte("x"))
		}
	}()
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Read(make([]byte, 1)); err != nil {
		t.Fatal(err)
	}

	if _, ok := sessionRTT(c); !ok {
		t.Fatal("sessionRTT: no RTT for a TCP socket")
	}
	if _, ok := sessionRTT(&net.UnixConn{}); ok {
		t.Fatal("sessionRTT: RTT for a non-TCP socket")
	}
}
