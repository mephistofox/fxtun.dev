package e2e

import (
	"io"
	"math"
	"net"
	"testing"
	"time"
)

// linkProfile emulates a link: the same bandwidth each way and a round-trip delay.
type linkProfile struct {
	name string
	mbps float64
	rtt  time.Duration
}

var benchProfiles = []linkProfile{
	{"100M/30ms", 100, 30 * time.Millisecond},
	{"50M/60ms", 50, 60 * time.Millisecond},
	{"10M/100ms", 10, 100 * time.Millisecond},
}

// startLinkProxy listens on a free local port and forwards every connection to
// target, shaping each direction to the profile. No tc/root needed.
func startLinkProxy(t *testing.T, target string, p linkProfile) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			in, err := ln.Accept()
			if err != nil {
				return
			}
			out, err := net.Dial("tcp", target)
			if err != nil {
				in.Close()
				continue
			}
			go shape(in, out, p)
			go shape(out, in, p)
		}
	}()
	return ln.Addr().String()
}

type chunk struct {
	data []byte
	at   time.Time
}

// shape copies src to dst: serialisation at the profile's rate, then half the
// RTT of propagation delay. Closes dst's write side when src ends.
//
// The queue holds about one bandwidth-delay product's worth of chunks, not an
// arbitrary large buffer: a bigger queue would let the reader race ahead of
// the simulated link and absorb bursts that a real link's TCP window would
// instead push back on the sender as backpressure.
func shape(src, dst net.Conn, p linkProfile) {
	bytesPerSec := p.mbps * 1e6 / 8
	bdp := bytesPerSec * p.rtt.Seconds()
	qcap := int(math.Ceil(bdp / (16 * 1024)))
	if qcap < 4 {
		qcap = 4
	}
	q := make(chan chunk, qcap)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if tc, ok := dst.(*net.TCPConn); ok {
				_ = tc.CloseWrite()
			}
		}()
		for c := range q {
			if d := time.Until(c.at); d > 0 {
				time.Sleep(d)
			}
			if _, err := dst.Write(c.data); err != nil {
				return
			}
		}
	}()
	next := time.Now()
	buf := make([]byte, 16*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			now := time.Now()
			if next.Before(now) {
				next = now
			}
			next = next.Add(time.Duration(float64(n) / bytesPerSec * float64(time.Second)))
			data := make([]byte, n)
			copy(data, buf[:n])
			// If the writer already gave up (dst.Write failed), nobody drains
			// q any more; without this select the send below would block
			// forever instead of letting the reader unwind too.
			select {
			case q <- chunk{data, next.Add(p.rtt / 2)}:
			case <-done:
				src.Close()
				return
			}
		}
		if err != nil {
			close(q)
			return
		}
	}
}

func TestLinkProxyShapesBandwidthAndDelay(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_, _ = c.Write(make([]byte, 1<<20)) // 1 MB
	}()
	addr := startLinkProxy(t, ln.Addr().String(), linkProfile{"t", 10, 100 * time.Millisecond})
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	start := time.Now()
	n, _ := io.Copy(io.Discard, c)
	d := time.Since(start)
	// 1 MB at 10 Mbit/s ≈ 0.84 s plus 50 ms one-way delay.
	if n != 1<<20 || d < 800*time.Millisecond || d > 2*time.Second {
		t.Fatalf("got %d bytes in %v, want 1 MiB in ~0.9s", n, d)
	}
}
