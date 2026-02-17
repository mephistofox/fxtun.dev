package client

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestAutoCloseTimer_TriggersOnIdle(t *testing.T) {
	var fired atomic.Bool
	timer := newAutoCloseTimer(100*time.Millisecond, func() {
		fired.Store(true)
	})
	defer timer.stop()

	// Wait for slightly more than the duration
	time.Sleep(200 * time.Millisecond)

	if !fired.Load() {
		t.Fatal("expected auto-close timer to fire after idle timeout")
	}
}

func TestAutoCloseTimer_ResetOnActivity(t *testing.T) {
	var fired atomic.Bool
	timer := newAutoCloseTimer(150*time.Millisecond, func() {
		fired.Store(true)
	})
	defer timer.stop()

	// Record activity every 50ms for 300ms total — should keep resetting the timer
	for i := 0; i < 6; i++ {
		time.Sleep(50 * time.Millisecond)
		timer.recordActivity()
	}

	if fired.Load() {
		t.Fatal("auto-close timer should not fire while activity is being recorded")
	}

	// Now wait for the idle timeout to expire without activity
	time.Sleep(250 * time.Millisecond)

	if !fired.Load() {
		t.Fatal("expected auto-close timer to fire after activity stops")
	}
}

func TestAutoCloseTimer_Stop(t *testing.T) {
	var fired atomic.Bool
	timer := newAutoCloseTimer(100*time.Millisecond, func() {
		fired.Store(true)
	})

	// Stop immediately
	timer.stop()

	time.Sleep(200 * time.Millisecond)

	if fired.Load() {
		t.Fatal("auto-close timer should not fire after being stopped")
	}
}

func TestMaxLifetimeTimer_Triggers(t *testing.T) {
	var fired atomic.Bool
	timer := newMaxLifetimeTimer(100*time.Millisecond, func() {
		fired.Store(true)
	})
	defer timer.stop()

	time.Sleep(200 * time.Millisecond)

	if !fired.Load() {
		t.Fatal("expected max-lifetime timer to fire after duration")
	}
}

func TestMaxLifetimeTimer_Stop(t *testing.T) {
	var fired atomic.Bool
	timer := newMaxLifetimeTimer(100*time.Millisecond, func() {
		fired.Store(true)
	})

	// Stop immediately
	timer.stop()

	time.Sleep(200 * time.Millisecond)

	if fired.Load() {
		t.Fatal("max-lifetime timer should not fire after being stopped")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"", 0, false},
		{"30s", 30 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"2h", 2 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"0.5d", 12 * time.Hour, false},
		{"-1m", 0, true},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		d, err := parseDuration(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseDuration(%q): expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDuration(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if d != tt.expected {
			t.Errorf("parseDuration(%q) = %v, want %v", tt.input, d, tt.expected)
		}
	}
}

func TestValidateAutoClose(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"", false},
		{"1m", false},
		{"30m", false},
		{"24h", false},
		{"1d", false},  // 1d = 24h exactly, allowed
		{"30s", true},  // below minimum
		{"25h", true},  // above maximum
		{"2d", true},   // 2d = 48h, above maximum
	}

	for _, tt := range tests {
		err := ValidateAutoClose(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateAutoClose(%q): expected error, got nil", tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateAutoClose(%q): unexpected error: %v", tt.input, err)
		}
	}
}

func TestValidateMaxLifetime(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"", false},
		{"1m", false},
		{"8h", false},
		{"7d", false},
		{"30s", true},  // below minimum
		{"8d", true},   // above 7d maximum
	}

	for _, tt := range tests {
		err := ValidateMaxLifetime(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateMaxLifetime(%q): expected error, got nil", tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateMaxLifetime(%q): unexpected error: %v", tt.input, err)
		}
	}
}
