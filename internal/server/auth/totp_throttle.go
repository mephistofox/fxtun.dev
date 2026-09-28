package auth

import (
	"sync"
	"time"
)

// totpFailureWindow is how long failed second-factor attempts are remembered.
const totpFailureWindow = 15 * time.Minute

// totpMaxFailures is how many wrong codes an account tolerates inside the
// window before every further attempt is refused.
const totpMaxFailures = 10

// totpThrottle counts failed second-factor attempts per account.
//
// Without it the only brake is the per-IP request limit, so an attacker who
// already has the password can spread the guessing across a botnet: at eight
// attempts per minute per IP, a thousand hosts cover a meaningful share of the
// 10^6 code space within hours. Counting per account instead of per IP removes
// that leverage.
//
// ponytail: process-local. On a single node this is the whole defence; with
// several API nodes move the counter to the shared Redis store.
type totpThrottle struct {
	mu       sync.Mutex
	failures map[int64]*totpFailureEntry
}

type totpFailureEntry struct {
	count     int
	expiresAt time.Time
}

func newTOTPThrottle() *totpThrottle {
	return &totpThrottle{failures: make(map[int64]*totpFailureEntry)}
}

// blocked reports whether the account has burned its attempt budget.
func (t *totpThrottle) blocked(userID int64) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	e, ok := t.failures[userID]
	if !ok {
		return false
	}
	if time.Now().After(e.expiresAt) {
		delete(t.failures, userID)
		return false
	}
	return e.count >= totpMaxFailures
}

// recordFailure counts one wrong code and starts the window if needed.
func (t *totpThrottle) recordFailure(userID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	e, ok := t.failures[userID]
	if !ok || now.After(e.expiresAt) {
		t.failures[userID] = &totpFailureEntry{count: 1, expiresAt: now.Add(totpFailureWindow)}
		return
	}
	e.count++
}

// reset clears the counter after a successful second factor.
func (t *totpThrottle) reset(userID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.failures, userID)
}
