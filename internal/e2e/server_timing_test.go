package e2e

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/server/database"
)

// A slow app must show up as app time, not tunnel time, in DevTools.
func TestServerTimingSplitsAppFromTunnel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	localPort := getFreePort(t)
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	require.NoError(t, err)
	app := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Server-Timing", "db;dur=150")
		w.Write([]byte("ok"))
	})}
	go app.Serve(ln)
	t.Cleanup(func() { app.Close() })

	h := NewHarness(t)
	h.Start()
	t.Cleanup(h.Stop)
	h.ConnectClient([]config.TunnelConfig{{Name: "web", Type: "http", LocalPort: localPort, Subdomain: "timing"}})

	req, _ := http.NewRequest("GET", "http://"+h.HTTPAddr+"/", nil)
	req.Host = "timing." + testDomain
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	timings := resp.Header.Values("Server-Timing")
	require.Contains(t, timings, "db;dur=150", "the app's own Server-Timing must survive")
	dur := func(name string) float64 {
		for _, v := range timings {
			if m := regexp.MustCompile(name + `;dur=([0-9.]+)`).FindStringSubmatch(v); m != nil {
				d, _ := strconv.ParseFloat(m[1], 64)
				return d
			}
		}
		t.Fatalf("no %s in %q", name, timings)
		return 0
	}
	require.InDelta(t, 200, dur("fxt-app"), 50, "app share")
	require.Less(t, dur("fxt-tunnel"), 20.0, "loopback tunnel share")
	require.Less(t, dur("fxt-edge"), 50.0, "edge share")

	// Custom domains often front production: no timings of ours there.
	h.Server.AddCustomDomain(&database.CustomDomain{Domain: "app.example.test", TargetSubdomain: "timing", Verified: true})
	req, _ = http.NewRequest("GET", "http://"+h.HTTPAddr+"/", nil)
	req.Host = "app.example.test"
	resp, err = (&http.Client{Timeout: 10 * time.Second}).Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, []string{"db;dur=150"}, resp.Header.Values("Server-Timing"))
}
