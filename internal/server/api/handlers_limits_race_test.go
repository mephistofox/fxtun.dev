package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// postConcurrently fires n identical POSTs at once and returns how many were created.
func postConcurrently(t *testing.T, n int, token string, url string, body func(i int) string) int {
	t.Helper()

	var wg, ready sync.WaitGroup
	start := make(chan struct{})
	codes := make([]int, n)

	for i := range n {
		wg.Add(1)
		ready.Add(1)
		// Each goroutine gets its own client so the TCP connect happens during
		// warm-up, not inside the window we are trying to overlap.
		client := &http.Client{Transport: &http.Transport{}}
		go func() {
			defer wg.Done()
			warm, err := http.NewRequest(http.MethodGet, url, nil)
			if err == nil {
				warm.Header.Set("Authorization", "Bearer "+token)
				if resp, err := client.Do(warm); err == nil {
					resp.Body.Close()
				}
			}
			req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body(i)))
			if err != nil {
				ready.Done()
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			ready.Done()
			<-start
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			codes[i] = resp.StatusCode
		}()
	}
	ready.Wait()
	close(start)
	wg.Wait()

	created := 0
	for _, c := range codes {
		if c == http.StatusCreated {
			created++
		}
	}
	return created
}

// TestReserveDomain_ConcurrentRespectsPlanLimit: parallel reservations must not
// slip past max_domains (free plan = 1).
func TestReserveDomain_ConcurrentRespectsPlanLimit(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+20000000901", "password123", "Race Domain User")

	created := postConcurrently(t, 10, user.AccessToken, env.Server.URL+"/api/domains",
		func(i int) string { return fmt.Sprintf(`{"subdomain":"race%d"}`, i) })

	var count int
	require.NoError(t, env.DB.Pool().QueryRow(context.Background(),
		`SELECT COUNT(*) FROM reserved_domains WHERE user_id = $1`, user.User.ID).Scan(&count))

	require.Equal(t, 1, count, "free plan allows exactly one reserved domain")
	require.Equal(t, 1, created)
}

// TestCreateToken_ConcurrentRespectsPlanLimit: same for api tokens (free = 1).
func TestCreateToken_ConcurrentRespectsPlanLimit(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+20000000902", "password123", "Race Token User")

	created := postConcurrently(t, 10, user.AccessToken, env.Server.URL+"/api/tokens",
		func(i int) string { return fmt.Sprintf(`{"name":"race%d"}`, i) })

	var count int
	require.NoError(t, env.DB.Pool().QueryRow(context.Background(),
		`SELECT COUNT(*) FROM api_tokens WHERE user_id = $1`, user.User.ID).Scan(&count))

	require.Equal(t, 1, count, "free plan allows exactly one api token")
	require.Equal(t, 1, created)
}

// TestCreateToken_NoPlanUsesDefaultLimit: a user without a plan must not get an
// unlimited token quota.
func TestCreateToken_NoPlanUsesDefaultLimit(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createTestUser(t, "+20000000903", "password123", "Planless User")

	_, err := env.DB.Pool().Exec(context.Background(),
		`UPDATE users SET plan_id = NULL WHERE id = $1`, user.User.ID)
	require.NoError(t, err)

	// Pre-fill the default quota (10) directly.
	for i := range 10 {
		_, err := env.DB.Pool().Exec(context.Background(),
			`INSERT INTO api_tokens (user_id, token_hash, name, allowed_subdomains, max_tunnels, allowed_ips, created_at)
			 VALUES ($1, $2, $3, '["*"]', 10, '[]', NOW())`,
			user.User.ID, fmt.Sprintf("hash-%d", i), fmt.Sprintf("seed%d", i))
		require.NoError(t, err)
	}

	req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/tokens",
		strings.NewReader(`{"name":"overflow"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusForbidden, resp.StatusCode,
		"a nil plan must fall back to the default limit, not unlimited")
}
