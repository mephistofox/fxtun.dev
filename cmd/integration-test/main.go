package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"

	clientcore "github.com/mephistofox/fxtunnel/internal/client/core"
	"github.com/mephistofox/fxtunnel/internal/config"
)

// CLI flags
var (
	serverAddr = flag.String("server", "mfdev.ru:4443", "Tunnel server address (host:port)")
	apiURL     = flag.String("api-url", "https://mfdev.ru", "API base URL")
	adminToken = flag.String("admin-token", "", "Admin JWT access token for managing users")
	cleanup    = flag.Bool("cleanup", true, "Delete test user after run")
	verbose    = flag.Bool("verbose", false, "Enable verbose client logging")
	httpOnly   = flag.Bool("http-only", false, "Only test HTTP tunnels")
	planFilter = flag.String("plan", "", "Test only this plan slug (empty = all)")
)

// --- API types ---

type planDTO struct {
	ID                 int64   `json:"id"`
	Slug               string  `json:"slug"`
	Name               string  `json:"name"`
	Price              float64 `json:"price"`
	MaxTunnels         int     `json:"max_tunnels"`
	MaxDomains         int     `json:"max_domains"`
	MaxTokens          int     `json:"max_tokens"`
	MaxTunnelsPerToken int     `json:"max_tunnels_per_token"`
	InspectorEnabled   bool    `json:"inspector_enabled"`
	MaxDataSessions    int     `json:"max_data_sessions"`
}

type userDTO struct {
	ID          int64  `json:"id"`
	Phone       string `json:"phone"`
	DisplayName string `json:"display_name"`
	IsAdmin     bool   `json:"is_admin"`
	IsActive    bool   `json:"is_active"`
	PlanID      int64  `json:"plan_id"`
}

type authResponse struct {
	User         *userDTO `json:"user"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"`
}

type createTokenResponse struct {
	Token string   `json:"token"`
	Info  tokenDTO `json:"info"`
}

type tokenDTO struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	MaxTunnels int   `json:"max_tunnels"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// --- API client ---

type apiClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func newAPIClient(baseURL, token string) *apiClient {
	return &apiClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (a *apiClient) withToken(token string) *apiClient {
	return &apiClient{
		baseURL: a.baseURL,
		token:   token,
		http:    a.http,
	}
}

func (a *apiClient) doRequest(method, path string, body interface{}) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := a.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("do request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return data, resp.StatusCode, nil
}

func (a *apiClient) get(path string) ([]byte, int, error) {
	return a.doRequest("GET", path, nil)
}

func (a *apiClient) post(path string, body interface{}) ([]byte, int, error) {
	return a.doRequest("POST", path, body)
}

func (a *apiClient) put(path string, body interface{}) ([]byte, int, error) {
	return a.doRequest("PUT", path, body)
}

func (a *apiClient) delete(path string) ([]byte, int, error) {
	return a.doRequest("DELETE", path, nil)
}

// --- Local echo servers ---

func startHTTPEcho(response string) (int, func()) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("start HTTP echo: %v", err))
	}
	port := ln.Addr().(*net.TCPAddr).Port

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, response)
		}),
	}
	go srv.Serve(ln)

	return port, func() {
		srv.Close()
	}
}

func startTCPEcho() (int, func()) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("start TCP echo: %v", err))
	}
	port := ln.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()

	return port, func() {
		ln.Close()
	}
}

func startUDPEcho() (int, func()) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("resolve UDP addr: %v", err))
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(fmt.Sprintf("start UDP echo: %v", err))
	}
	port := conn.LocalAddr().(*net.UDPAddr).Port

	go func() {
		buf := make([]byte, 65536)
		for {
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			conn.WriteToUDP(buf[:n], remoteAddr)
		}
	}()

	return port, func() {
		conn.Close()
	}
}

// --- Test results ---

type resultEntry struct {
	OK      bool
	Latency time.Duration
	Err     string
}

type testResult struct {
	Plan string
	HTTP resultEntry
	TCP  resultEntry
	UDP  resultEntry
}

func (r resultEntry) statusStr() string {
	if r.Err == "SKIP" {
		return "SKIP"
	}
	if r.OK {
		return fmt.Sprintf("OK %dms", r.Latency.Milliseconds())
	}
	errStr := r.Err
	if len(errStr) > 30 {
		errStr = errStr[:30] + "..."
	}
	return fmt.Sprintf("FAIL: %s", errStr)
}

func (r testResult) allOK() bool {
	return r.HTTP.OK && (r.TCP.OK || r.TCP.Err == "SKIP") && (r.UDP.OK || r.UDP.Err == "SKIP")
}

func printResultsTable(results []testResult) {
	// Calculate column widths
	planW := 12
	for _, r := range results {
		if len(r.Plan) > planW {
			planW = len(r.Plan)
		}
	}
	httpW, tcpW, udpW := 20, 20, 20
	for _, r := range results {
		if l := len(r.HTTP.statusStr()); l > httpW {
			httpW = l
		}
		if l := len(r.TCP.statusStr()); l > tcpW {
			tcpW = l
		}
		if l := len(r.UDP.statusStr()); l > udpW {
			udpW = l
		}
	}
	resultW := 6

	hLine := func(l, m, r string, fill string) string {
		return l +
			strings.Repeat(fill, planW+2) + m +
			strings.Repeat(fill, httpW+2) + m +
			strings.Repeat(fill, tcpW+2) + m +
			strings.Repeat(fill, udpW+2) + m +
			strings.Repeat(fill, resultW+2) + r
	}

	pad := func(s string, w int) string {
		if len(s) >= w {
			return s[:w]
		}
		return s + strings.Repeat(" ", w-len(s))
	}

	center := func(s string, w int) string {
		if len(s) >= w {
			return s[:w]
		}
		left := (w - len(s)) / 2
		right := w - len(s) - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	}

	fmt.Println()
	fmt.Println(hLine("╔", "╦", "╗", "═"))
	fmt.Printf("║ %s ║ %s ║ %s ║ %s ║ %s ║\n",
		center("Plan", planW),
		center("HTTP", httpW),
		center("TCP", tcpW),
		center("UDP", udpW),
		center("Result", resultW),
	)
	fmt.Println(hLine("╠", "╬", "╣", "═"))

	for _, r := range results {
		resStr := "PASS"
		if !r.allOK() {
			resStr = "FAIL"
		}
		fmt.Printf("║ %s ║ %s ║ %s ║ %s ║ %s ║\n",
			pad(r.Plan, planW),
			pad(r.HTTP.statusStr(), httpW),
			pad(r.TCP.statusStr(), tcpW),
			pad(r.UDP.statusStr(), udpW),
			center(resStr, resultW),
		)
	}
	fmt.Println(hLine("╚", "╩", "╝", "═"))
	fmt.Println()
}

// --- Helpers ---

// baseDomain extracts the domain from the server address (host:port -> host)
func baseDomain() string {
	host, _, err := net.SplitHostPort(*serverAddr)
	if err != nil {
		return *serverAddr
	}
	return host
}

func logInfo(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func logError(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

func logStep(step int, total int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%d/%d] %s\n", step, total, msg)
}

// --- Main flow ---

func main() {
	flag.Parse()

	if *adminToken == "" {
		fmt.Fprintln(os.Stderr, "Error: --admin-token is required")
		flag.Usage()
		os.Exit(1)
	}

	domain := baseDomain()
	ts := time.Now().Unix()

	logInfo("Integration test starting")
	logInfo("  Server:   %s", *serverAddr)
	logInfo("  API URL:  %s", *apiURL)
	logInfo("  Domain:   %s", domain)
	logInfo("  Cleanup:  %v", *cleanup)
	fmt.Println()

	adminAPI := newAPIClient(*apiURL, *adminToken)

	// Step 1: Fetch plans
	logStep(1, 5, "Fetching plans from %s", *apiURL)
	plans, err := fetchPlans(adminAPI)
	if err != nil {
		logError("Failed to fetch plans: %v", err)
		os.Exit(1)
	}
	logInfo("Found %d plans", len(plans))
	for _, p := range plans {
		logInfo("  - %s (id=%d, max_tunnels=%d, max_tokens=%d)", p.Slug, p.ID, p.MaxTunnels, p.MaxTokens)
	}

	if *planFilter != "" {
		filtered := make([]planDTO, 0)
		for _, p := range plans {
			if p.Slug == *planFilter {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			logError("Plan %q not found", *planFilter)
			os.Exit(1)
		}
		plans = filtered
		logInfo("Filtered to plan: %s", *planFilter)
	}

	// Step 2: Create test user via register
	logStep(2, 5, "Creating test user")
	testPhone := fmt.Sprintf("+1test%d", ts)
	testPassword := "IntegrationTest123!"

	user, userAccessToken, err := createTestUser(adminAPI, testPhone, testPassword)
	if err != nil {
		logError("Failed to create test user: %v", err)
		os.Exit(1)
	}
	logInfo("Test user created: id=%d, phone=%s", user.ID, user.Phone)

	// Ensure cleanup
	if *cleanup {
		defer func() {
			logStep(5, 5, "Cleaning up: deleting test user %d", user.ID)
			_, status, err := adminAPI.delete(fmt.Sprintf("/api/admin/users/%d", user.ID))
			if err != nil {
				logError("Failed to delete test user: %v", err)
			} else if status >= 400 {
				logError("Failed to delete test user: HTTP %d", status)
			} else {
				logInfo("Test user deleted")
			}
		}()
	}

	userAPI := adminAPI.withToken(userAccessToken)

	// Step 3: Test each plan
	logStep(3, 5, "Running tunnel tests for %d plan(s)", len(plans))
	var results []testResult

	for i, plan := range plans {
		fmt.Println()
		logInfo("=== Plan %d/%d: %s (id=%d) ===", i+1, len(plans), plan.Slug, plan.ID)

		result := testPlan(adminAPI, userAPI, user, plan, domain, ts)
		results = append(results, result)
	}

	// Step 4: Print results
	fmt.Println()
	logStep(4, 5, "Results summary")
	printResultsTable(results)

	// Check overall status
	allPassed := true
	for _, r := range results {
		if !r.allOK() {
			allPassed = false
		}
	}

	if allPassed {
		logInfo("All tests PASSED")
	} else {
		logError("Some tests FAILED")
		os.Exit(1)
	}
}

func fetchPlans(api *apiClient) ([]planDTO, error) {
	data, status, err := api.get("/api/admin/plans")
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(data))
	}

	var plans []planDTO
	if err := json.Unmarshal(data, &plans); err != nil {
		return nil, fmt.Errorf("unmarshal plans: %w", err)
	}
	return plans, nil
}

func createTestUser(api *apiClient, phone, password string) (*userDTO, string, error) {
	// Try to register
	regReq := map[string]string{
		"phone":        phone,
		"password":     password,
		"display_name": "Integration Test",
	}
	data, status, err := api.post("/api/auth/register", regReq)
	if err != nil {
		return nil, "", fmt.Errorf("register request: %w", err)
	}

	if status == http.StatusCreated || status == http.StatusOK {
		var resp authResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, "", fmt.Errorf("unmarshal register response: %w", err)
		}
		return resp.User, resp.AccessToken, nil
	}

	// If phone exists, try login
	if status == http.StatusConflict {
		loginReq := map[string]string{
			"phone":    phone,
			"password": password,
		}
		data, status, err = api.post("/api/auth/login", loginReq)
		if err != nil {
			return nil, "", fmt.Errorf("login request: %w", err)
		}
		if status != http.StatusOK {
			return nil, "", fmt.Errorf("login failed: HTTP %d: %s", status, string(data))
		}
		var resp authResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, "", fmt.Errorf("unmarshal login response: %w", err)
		}
		return resp.User, resp.AccessToken, nil
	}

	return nil, "", fmt.Errorf("register failed: HTTP %d: %s", status, string(data))
}

func testPlan(adminAPI, userAPI *apiClient, user *userDTO, plan planDTO, domain string, ts int64) testResult {
	result := testResult{Plan: plan.Slug}

	// Assign plan to user via admin grant-subscription
	logInfo("Granting plan %q to user %d", plan.Slug, user.ID)
	grantReq := map[string]interface{}{
		"plan_id": plan.ID,
		"months":  1,
	}
	data, status, err := adminAPI.post(fmt.Sprintf("/api/admin/users/%d/grant-subscription", user.ID), grantReq)
	if err != nil {
		logError("Failed to grant subscription: %v", err)
		result.HTTP = resultEntry{Err: "grant plan failed"}
		result.TCP = resultEntry{Err: "grant plan failed"}
		result.UDP = resultEntry{Err: "grant plan failed"}
		return result
	}
	if status >= 400 {
		logError("Failed to grant subscription: HTTP %d: %s", status, string(data))
		result.HTTP = resultEntry{Err: fmt.Sprintf("grant plan HTTP %d", status)}
		result.TCP = resultEntry{Err: fmt.Sprintf("grant plan HTTP %d", status)}
		result.UDP = resultEntry{Err: fmt.Sprintf("grant plan HTTP %d", status)}
		return result
	}
	logInfo("Plan granted successfully")

	// Create API token for this plan test
	logInfo("Creating API token for plan %q", plan.Slug)
	tokenReq := map[string]interface{}{
		"name":       fmt.Sprintf("itest-%s-%d", plan.Slug, ts),
		"max_tunnels": 10,
	}
	data, status, err = userAPI.post("/api/tokens", tokenReq)
	if err != nil {
		logError("Failed to create token: %v", err)
		result.HTTP = resultEntry{Err: "create token failed"}
		result.TCP = resultEntry{Err: "create token failed"}
		result.UDP = resultEntry{Err: "create token failed"}
		return result
	}
	if status >= 400 {
		logError("Failed to create token: HTTP %d: %s", status, string(data))
		result.HTTP = resultEntry{Err: fmt.Sprintf("create token HTTP %d", status)}
		result.TCP = resultEntry{Err: fmt.Sprintf("create token HTTP %d", status)}
		result.UDP = resultEntry{Err: fmt.Sprintf("create token HTTP %d", status)}
		return result
	}

	var tokenResp createTokenResponse
	if err := json.Unmarshal(data, &tokenResp); err != nil {
		logError("Failed to parse token response: %v", err)
		result.HTTP = resultEntry{Err: "parse token failed"}
		result.TCP = resultEntry{Err: "parse token failed"}
		result.UDP = resultEntry{Err: "parse token failed"}
		return result
	}
	apiToken := tokenResp.Token
	tokenID := tokenResp.Info.ID
	logInfo("Token created: id=%d", tokenID)

	// Ensure token cleanup
	defer func() {
		logInfo("Deleting API token %d", tokenID)
		_, _, _ = userAPI.delete(fmt.Sprintf("/api/tokens/%d", tokenID))
	}()

	// Start local echo servers
	httpResponse := fmt.Sprintf("HTTP_OK:%s", plan.Slug)
	httpPort, httpStop := startHTTPEcho(httpResponse)
	defer httpStop()

	tcpPort, tcpStop := startTCPEcho()
	defer tcpStop()

	udpPort, udpStop := startUDPEcho()
	defer udpStop()

	logInfo("Local echo servers started: HTTP=%d, TCP=%d, UDP=%d", httpPort, tcpPort, udpPort)

	// Build tunnel configs
	subdomain := fmt.Sprintf("itest-%s-%d", plan.Slug, ts)

	tunnelConfigs := []config.TunnelConfig{
		{
			Name:      "itest-http",
			Type:      "http",
			LocalPort: httpPort,
			Subdomain: subdomain,
		},
	}

	if !*httpOnly {
		tunnelConfigs = append(tunnelConfigs,
			config.TunnelConfig{
				Name:      "itest-tcp",
				Type:      "tcp",
				LocalPort: tcpPort,
			},
			config.TunnelConfig{
				Name:      "itest-udp",
				Type:      "udp",
				LocalPort: udpPort,
			},
		)
	}

	// Connect fxtunnel client
	logInfo("Connecting tunnel client to %s", *serverAddr)
	logLevel := zerolog.WarnLevel
	if *verbose {
		logLevel = zerolog.DebugLevel
	}
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"}).
		With().Timestamp().Logger().Level(logLevel)

	clientCfg := &config.ClientConfig{
		Server: config.ClientServerSettings{
			Address:   *serverAddr,
			Token:     apiToken,
			Insecure:  false,
			TLSVerify: false,
		},
		Tunnels:   tunnelConfigs,
		Reconnect: config.ReconnectSettings{Enabled: false},
		Inspect:   config.InspectSettings{Enabled: false},
	}

	client := clientcore.New(clientCfg, logger)
	if err := client.Connect(); err != nil {
		logError("Failed to connect tunnel client: %v", err)
		result.HTTP = resultEntry{Err: fmt.Sprintf("connect: %v", err)}
		result.TCP = resultEntry{Err: fmt.Sprintf("connect: %v", err)}
		result.UDP = resultEntry{Err: fmt.Sprintf("connect: %v", err)}
		return result
	}
	defer client.Close()

	logInfo("Client connected, waiting for tunnel establishment...")

	// Wait for tunnels to be established
	var tunnels []*clientcore.ActiveTunnel
	deadline := time.Now().Add(15 * time.Second)
	expectedCount := len(tunnelConfigs)
	for time.Now().Before(deadline) {
		tunnels = client.GetTunnels()
		if len(tunnels) >= expectedCount {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if len(tunnels) < expectedCount {
		logError("Tunnel establishment timeout: got %d/%d tunnels", len(tunnels), expectedCount)
		result.HTTP = resultEntry{Err: fmt.Sprintf("got %d/%d tunnels", len(tunnels), expectedCount)}
		if !*httpOnly {
			result.TCP = resultEntry{Err: fmt.Sprintf("got %d/%d tunnels", len(tunnels), expectedCount)}
			result.UDP = resultEntry{Err: fmt.Sprintf("got %d/%d tunnels", len(tunnels), expectedCount)}
		}
		return result
	}

	logInfo("Tunnels established: %d", len(tunnels))
	for _, t := range tunnels {
		if t.URL != "" {
			logInfo("  HTTP: %s (HTTPS: %s)", t.URL, t.HTTPSURL)
		} else {
			logInfo("  %s: %s", strings.ToUpper(t.Config.Type), t.RemoteAddr)
		}
	}

	// Find tunnel info by type
	var httpTunnel, tcpTunnel, udpTunnel *clientcore.ActiveTunnel
	for _, t := range tunnels {
		switch t.Config.Type {
		case "http":
			httpTunnel = t
		case "tcp":
			tcpTunnel = t
		case "udp":
			udpTunnel = t
		}
	}

	// Test HTTP
	if httpTunnel != nil {
		result.HTTP = testHTTPTunnel(httpTunnel, domain, subdomain, httpResponse)
	} else {
		result.HTTP = resultEntry{Err: "no HTTP tunnel"}
	}

	// Test TCP
	if *httpOnly {
		result.TCP = resultEntry{Err: "SKIP"}
		result.UDP = resultEntry{Err: "SKIP"}
	} else {
		if tcpTunnel != nil {
			result.TCP = testTCPTunnel(tcpTunnel, plan.Slug)
		} else {
			result.TCP = resultEntry{Err: "no TCP tunnel"}
		}

		// Test UDP
		if udpTunnel != nil {
			result.UDP = testUDPTunnel(udpTunnel, plan.Slug)
		} else {
			result.UDP = resultEntry{Err: "no UDP tunnel"}
		}
	}

	return result
}

func testHTTPTunnel(tunnel *clientcore.ActiveTunnel, domain, subdomain, expectedBody string) resultEntry {
	logInfo("Testing HTTP tunnel...")

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Try HTTPS first via the tunnel URL, fall back to HTTP with Host header
	var testURL string
	if tunnel.HTTPSURL != "" {
		testURL = tunnel.HTTPSURL + "/integration-test"
	} else if tunnel.URL != "" {
		testURL = tunnel.URL + "/integration-test"
	} else {
		// Construct URL manually
		testURL = fmt.Sprintf("http://%s/integration-test", domain)
	}

	start := time.Now()

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return resultEntry{Err: fmt.Sprintf("create request: %v", err)}
	}

	// If using base domain URL (port 80), set Host header
	if !strings.Contains(testURL, subdomain) {
		req.Host = fmt.Sprintf("%s.%s", subdomain, domain)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		// Try HTTP with Host header as fallback
		logInfo("  HTTPS failed, trying HTTP with Host header...")
		fallbackURL := fmt.Sprintf("http://%s:80/integration-test", domain)
		req2, _ := http.NewRequest("GET", fallbackURL, nil)
		req2.Host = fmt.Sprintf("%s.%s", subdomain, domain)

		start = time.Now()
		resp, err = httpClient.Do(req2)
		if err != nil {
			return resultEntry{Err: fmt.Sprintf("request: %v", err)}
		}
	}
	latency := time.Since(start)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resultEntry{Err: fmt.Sprintf("read body: %v", err)}
	}

	if resp.StatusCode != http.StatusOK {
		return resultEntry{Err: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))}
	}

	if !strings.Contains(string(body), expectedBody) {
		return resultEntry{Err: fmt.Sprintf("body mismatch: got %q", string(body))}
	}

	logInfo("  HTTP OK: %dms", latency.Milliseconds())
	return resultEntry{OK: true, Latency: latency}
}

func testTCPTunnel(tunnel *clientcore.ActiveTunnel, planSlug string) resultEntry {
	logInfo("Testing TCP tunnel at %s...", tunnel.RemoteAddr)

	if tunnel.RemoteAddr == "" {
		return resultEntry{Err: "no remote address"}
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", tunnel.RemoteAddr, 10*time.Second)
	if err != nil {
		return resultEntry{Err: fmt.Sprintf("dial: %v", err)}
	}
	defer conn.Close()

	testData := fmt.Sprintf("TCP_PING:%s", planSlug)
	_, err = conn.Write([]byte(testData))
	if err != nil {
		return resultEntry{Err: fmt.Sprintf("write: %v", err)}
	}

	buf := make([]byte, len(testData))
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return resultEntry{Err: fmt.Sprintf("read: %v", err)}
	}
	latency := time.Since(start)

	if string(buf) != testData {
		return resultEntry{Err: fmt.Sprintf("echo mismatch: got %q", string(buf))}
	}

	logInfo("  TCP OK: %dms", latency.Milliseconds())
	return resultEntry{OK: true, Latency: latency}
}

func testUDPTunnel(tunnel *clientcore.ActiveTunnel, planSlug string) resultEntry {
	logInfo("Testing UDP tunnel at %s...", tunnel.RemoteAddr)

	if tunnel.RemoteAddr == "" {
		return resultEntry{Err: "no remote address"}
	}

	remoteAddr, err := net.ResolveUDPAddr("udp", tunnel.RemoteAddr)
	if err != nil {
		return resultEntry{Err: fmt.Sprintf("resolve: %v", err)}
	}

	start := time.Now()
	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		return resultEntry{Err: fmt.Sprintf("dial: %v", err)}
	}
	defer conn.Close()

	testData := fmt.Sprintf("UDP_PING:%s", planSlug)
	_, err = conn.Write([]byte(testData))
	if err != nil {
		return resultEntry{Err: fmt.Sprintf("write: %v", err)}
	}

	buf := make([]byte, len(testData))
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		return resultEntry{Err: fmt.Sprintf("read: %v", err)}
	}
	latency := time.Since(start)

	if string(buf[:n]) != testData {
		return resultEntry{Err: fmt.Sprintf("echo mismatch: got %q", string(buf[:n]))}
	}

	logInfo("  UDP OK: %dms", latency.Milliseconds())
	return resultEntry{OK: true, Latency: latency}
}
