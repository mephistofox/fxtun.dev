package e2e

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clientcore "github.com/mephistofox/fxtun.dev/internal/client/core"
	"github.com/mephistofox/fxtun.dev/internal/config"
)

// The CLI refuses auto-close below 1m; the core takes the config as is, so a
// short timeout keeps these tests fast.
const testAutoClose = "3s"

// pingFor sends one ping every 500ms for d; every ping must be echoed.
func pingFor(t *testing.T, conn net.Conn, d time.Duration) {
	t.Helper()
	buf := make([]byte, 1)
	for start := time.Now(); time.Since(start) < d; time.Sleep(500 * time.Millisecond) {
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		_, err := conn.Write([]byte("x"))
		require.NoError(t, err, "write after %v", time.Since(start))
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err, "echo after %v", time.Since(start))
	}
}

// requireIdleClose checks the tunnel closes about one auto-close period after
// the traffic stopped: not sooner, not much later.
func requireIdleClose(t *testing.T, c *clientcore.Client) {
	t.Helper()
	stopped := time.Now()
	for len(c.GetTunnels()) > 0 && time.Since(stopped) < 6*time.Second {
		time.Sleep(50 * time.Millisecond)
	}
	idle := time.Since(stopped)
	t.Logf("closed %v after the traffic stopped", idle.Round(10*time.Millisecond))
	require.Empty(t, c.GetTunnels(), "tunnel still open %v after traffic stopped", idle)
	require.Greater(t, idle, 2*time.Second, "closed too early")
	require.Less(t, idle, 5*time.Second, "closed too late")
}

func TestAutoCloseKeepsTCPTunnelWithTraffic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { defer conn.Close(); _, _ = io.Copy(conn, conn) }()
		}
	}()

	h := NewHarness(t)
	h.Start()
	t.Cleanup(func() { h.Stop() })
	c := h.ConnectClient([]config.TunnelConfig{{
		Name: "t", Type: "tcp", LocalPort: ln.Addr().(*net.TCPAddr).Port, AutoClose: testAutoClose,
	}})

	// One long-lived connection: only its bytes can keep the tunnel alive.
	conn, err := net.DialTimeout("tcp", tcpTunnelAddr(t, c), 5*time.Second)
	require.NoError(t, err)
	defer conn.Close()
	pingFor(t, conn, 6*time.Second)
	require.Len(t, c.GetTunnels(), 1, "tunnel auto-closed while carrying traffic")

	requireIdleClose(t, c)
}

func TestAutoCloseKeepsUDPTunnelWithTraffic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	ul, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	t.Cleanup(func() { ul.Close() })
	go func() {
		b := make([]byte, 1500)
		for {
			n, a, err := ul.ReadFromUDP(b)
			if err != nil {
				return
			}
			_, _ = ul.WriteToUDP(b[:n], a)
		}
	}()

	h := NewHarness(t)
	h.Start()
	t.Cleanup(func() { h.Stop() })
	rp := h.ServerCfg.Server.UDPPortRange.Min + 1
	c := h.ConnectClient([]config.TunnelConfig{{
		Name: "u", Type: "udp", LocalPort: ul.LocalAddr().(*net.UDPAddr).Port, RemotePort: rp, AutoClose: testAutoClose,
	}})

	conn, err := net.Dial("udp", fmt.Sprintf("127.0.0.1:%d", rp))
	require.NoError(t, err)
	defer conn.Close()
	pingFor(t, conn, 6*time.Second)
	require.Len(t, c.GetTunnels(), 1, "tunnel auto-closed while carrying traffic")

	requireIdleClose(t, c)
}
