package auth

import "testing"

// A password leak plus an unthrottled second factor is a 10^6 guessing game
// that a botnet wins: the per-IP limit does nothing when the attempts come
// from a thousand hosts. The budget must be per account.
func TestTOTPThrottleBlocksAfterBudget(t *testing.T) {
	th := newTOTPThrottle()
	const victim = int64(42)

	if th.blocked(victim) {
		t.Fatal("fresh account is already blocked")
	}
	for i := 0; i < totpMaxFailures-1; i++ {
		th.recordFailure(victim)
	}
	if th.blocked(victim) {
		t.Errorf("blocked after %d failures, budget is %d", totpMaxFailures-1, totpMaxFailures)
	}

	th.recordFailure(victim)
	if !th.blocked(victim) {
		t.Errorf("still accepting guesses after %d failures", totpMaxFailures)
	}

	// Another account must not be affected.
	if th.blocked(victim + 1) {
		t.Error("throttle leaked onto an unrelated account")
	}

	// A correct code clears the budget.
	th.reset(victim)
	if th.blocked(victim) {
		t.Error("budget survived a successful authentication")
	}
}
