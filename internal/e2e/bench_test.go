package e2e

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	clientcore "github.com/mephistofox/fxtun.dev/internal/client/core"
	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/protocol"
)

var benchLevels = []struct {
	name     string
	compress bool
	level    zstd.EncoderLevel
}{
	{"off", false, zstd.SpeedDefault},
	{"fastest", true, zstd.SpeedFastest},
	{"default", true, zstd.SpeedDefault},
}

// benchWords seeds the varied strings in benchJSON — enough variety that
// records don't compress away to almost nothing, unlike a fixed template.
var benchWords = []string{
	"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
	"india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
	"quebec", "romeo", "sierra", "tango", "uniform", "victor", "whiskey", "yankee",
}

// benchJSON builds ~20 MB of realistic, only moderately compressible JSON
// records: nested objects, varied string lengths, random ids and numbers,
// and an optional field — unlike a fixed template repeated with a counter.
func benchJSON() []byte {
	rnd := rand.New(rand.NewSource(2))
	word := func() string { return benchWords[rnd.Intn(len(benchWords))] }
	hexID := func(n int) string {
		b := make([]byte, n)
		rnd.Read(b)
		return hex.EncodeToString(b)
	}

	var js bytes.Buffer
	js.WriteByte('[')
	for js.Len() < 20<<20 {
		if js.Len() > 1 {
			js.WriteByte(',')
		}
		rec := map[string]any{
			"id":     rnd.Intn(1_000_000),
			"uuid":   hexID(16),
			"name":   fmt.Sprintf("%s-%s-%d", word(), word(), rnd.Intn(10000)),
			"email":  fmt.Sprintf("%s.%s@example.com", word(), hexID(4)),
			"active": rnd.Intn(2) == 0,
			"score":  rnd.Float64() * 100,
			"address": map[string]any{
				"city": word(),
				"zip":  fmt.Sprintf("%05d", rnd.Intn(100000)),
			},
		}
		if rnd.Intn(3) == 0 {
			rec["note"] = word() + " " + word() + " " + word()
		}
		b, _ := json.Marshal(rec)
		js.Write(b)
	}
	js.WriteByte(']')
	return js.Bytes()
}

// benchApp is the local service behind the tunnel.
func benchApp(t *testing.T) int {
	t.Helper()
	rnd := make([]byte, 50<<20)
	rand.New(rand.NewSource(1)).Read(rnd)
	js := benchJSON()
	mux := http.NewServeMux()
	mux.HandleFunc("/rand", func(w http.ResponseWriter, r *http.Request) { w.Write(rnd) })
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) { w.Write(js) })
	mux.HandleFunc("/small", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"ok":true}`)) })
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		n, _ := io.Copy(io.Discard, r.Body)
		fmt.Fprint(w, n)
	})
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		f := w.(http.Flusher)
		for i := 0; i < 150; i++ {
			fmt.Fprintf(w, "data: %d\n\n", i)
			f.Flush()
			time.Sleep(time.Second)
		}
	})
	port := getFreePort(t)
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return port
}

func benchClient(t *testing.T, h *E2EHarness, addr string, port int, compress bool) *clientcore.Client {
	t.Helper()
	c := clientcore.New(&config.ClientConfig{
		Server:  config.ClientServerSettings{Address: addr, Token: h.Token, Insecure: true, Compression: compress},
		Tunnels: []config.TunnelConfig{{Name: "b", Type: "http", LocalPort: port, Subdomain: "bench"}},
		Logging: config.LoggingSettings{Level: "error", Format: "console"},
	}, h.log)
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	return c
}

// closeSoon stops f without waiting on it forever: a client with compression
// can hang in Close until the zstd close-order fix lands, and the server's
// Stop can hang the same way closing that client's session. It reports
// whether f finished — a timeout leaves f's goroutine running into whatever
// the caller does next (e.g. the next benchmark iteration), which the caller
// should flag rather than silently trust the row it just measured.
func closeSoon(f func()) bool {
	done := make(chan struct{})
	go func() { f(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(5 * time.Second):
		return false
	}
}

func benchGet(h *E2EHarness, path string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", "http://"+h.HTTPAddr+path, nil)
	req.Host = "bench." + testDomain
	return http.DefaultClient.Do(req)
}

func mbps(n int64, d time.Duration) string {
	return fmt.Sprintf("%.1f MB/s", float64(n)/(1<<20)/d.Seconds())
}

// TestBench measures the tunnel over emulated links. Run: FXTUN_BENCH=1 go test ./internal/e2e -run TestBench -v -timeout 60m
func TestBench(t *testing.T) {
	if os.Getenv("FXTUN_BENCH") != "1" {
		t.Skip("set FXTUN_BENCH=1 to run the benchmark")
	}
	port := benchApp(t)
	var rows []string
	defaultLevel := protocol.ZstdLevel
	for _, p := range benchProfiles {
		for _, lv := range benchLevels {
			protocol.ZstdLevel = lv.level
			h := NewHarness(t)
			h.ServerCfg.Server.CompressionEnabled = lv.compress
			h.Start()
			addr := startLinkProxy(t, h.ServerAddr, p)
			c := benchClient(t, h, addr, port, lv.compress)
			row := []string{p.name, lv.name}
			for _, path := range []string{"/rand", "/json"} {
				start := time.Now()
				resp, err := benchGet(h, path)
				var n int64
				if err == nil {
					n, err = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
				row = append(row, fmt.Sprintf("%s %s err=%v", path, mbps(n, time.Since(start)), err))
			}
			var ttfb []time.Duration
			for i := 0; i < 200; i++ {
				start := time.Now()
				resp, err := benchGet(h, "/small")
				if err != nil {
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				ttfb = append(ttfb, time.Since(start))
			}
			sort.Slice(ttfb, func(i, j int) bool { return ttfb[i] < ttfb[j] })
			if len(ttfb) > 0 {
				row = append(row, fmt.Sprintf("small n=%d p50=%v p95=%v", len(ttfb),
					ttfb[len(ttfb)/2].Round(time.Millisecond), ttfb[len(ttfb)*95/100].Round(time.Millisecond)))
			}
			closedClient := closeSoon(c.Close)
			closedHarness := closeSoon(h.Stop)
			if !closedClient || !closedHarness {
				row = append(row, "[teardown timeout]")
			}
			rows = append(rows, strings.Join(row, " | "))
		}
	}
	protocol.ZstdLevel = defaultLevel
	rows = append(rows, benchLongScenarios(t, port)...)
	fmt.Fprintln(os.Stderr, "\nBENCH RESULTS")
	for _, r := range rows {
		fmt.Fprintln(os.Stderr, r)
	}
}

// profileByName looks up a benchProfiles entry by name, so callers don't rely
// on slice position staying in sync with the declaration order above.
func profileByName(name string) linkProfile {
	for _, p := range benchProfiles {
		if p.name == name {
			return p
		}
	}
	panic("unknown profile: " + name)
}

// benchLongScenarios runs the slow scenarios once, on the 10M/100ms link with
// the current default compression: SSE for 150 s, an upload longer than 30 s,
// and small requests during a large download (ping starvation).
func benchLongScenarios(t *testing.T, port int) []string {
	p := profileByName("10M/100ms")
	h := NewHarness(t)
	h.ServerCfg.Server.CompressionEnabled = true
	h.Start()
	addr := startLinkProxy(t, h.ServerAddr, p)
	c := benchClient(t, h, addr, port, true)
	var rows []string

	start := time.Now()
	events := 0
	resp, err := benchGet(h, "/sse")
	if err == nil {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "data:") {
				events++
			}
		}
		err = sc.Err()
		resp.Body.Close()
	}
	rows = append(rows, fmt.Sprintf("%s | SSE 150s | %d/150 events in %v err=%v", p.name, events, time.Since(start).Round(time.Second), err))

	uploadBody := make([]byte, 50<<20) // ~42 s at 10 Mbit/s if it can't be compressed away
	rand.New(rand.NewSource(3)).Read(uploadBody)
	body := bytes.NewReader(uploadBody)
	req, _ := http.NewRequest("POST", "http://"+h.HTTPAddr+"/upload", body)
	req.Host = "bench." + testDomain
	start = time.Now()
	resp, err = http.DefaultClient.Do(req)
	got := ""
	if err == nil {
		b, _ := io.ReadAll(resp.Body)
		got = string(b)
		resp.Body.Close()
	}
	rows = append(rows, fmt.Sprintf("%s | upload 50MB | %v got=%s err=%v", p.name, time.Since(start).Round(time.Second), got, err))

	var wg sync.WaitGroup
	wg.Add(1)
	var dlErr error
	go func() {
		defer wg.Done()
		resp, err := benchGet(h, "/rand")
		if err != nil {
			dlErr = err
			return
		}
		_, dlErr = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	time.Sleep(time.Second)
	var worst time.Duration
	fails := 0
	for i := 0; i < 20; i++ {
		s := time.Now()
		resp, err := benchGet(h, "/small")
		if err != nil || resp.StatusCode != http.StatusOK {
			fails++
		} else {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		if d := time.Since(s); d > worst {
			worst = d
		}
		time.Sleep(time.Second)
	}
	wg.Wait()
	rows = append(rows, fmt.Sprintf("%s | small during 50MB download | worst=%v fails=%d download err=%v", p.name, worst.Round(time.Millisecond), fails, dlErr))

	closedClient := closeSoon(c.Close)
	closedHarness := closeSoon(h.Stop)
	if !closedClient || !closedHarness {
		rows = append(rows, fmt.Sprintf("%s | teardown | [teardown timeout]", p.name))
	}
	return rows
}
