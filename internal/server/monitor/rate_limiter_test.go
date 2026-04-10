package monitor

import (
	"testing"
	"time"
)

func TestSlidingWindow_Basic(t *testing.T) {
	sw := NewSlidingWindow(5, time.Minute)
	for i := 0; i < 5; i++ {
		if !sw.Allow() {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if sw.Allow() {
		t.Fatal("6th request should be denied")
	}
	if c := sw.Count(); c != 5 {
		t.Fatalf("expected count 5, got %d", c)
	}
}

func TestSlidingWindow_Unlimited(t *testing.T) {
	sw := NewSlidingWindow(0, time.Minute)
	for i := 0; i < 100000; i++ {
		if !sw.Allow() {
			t.Fatalf("unlimited window denied request %d", i)
		}
	}
	if c := sw.Count(); c != 0 {
		t.Fatalf("unlimited window should report count 0, got %d", c)
	}
}

func TestSlidingWindow_IsIdle_Empty(t *testing.T) {
	sw := NewSlidingWindow(5, time.Minute)
	if !sw.IsIdle(time.Second) {
		t.Fatal("empty window should be idle")
	}
}

func TestSlidingWindow_IsIdle_Recent(t *testing.T) {
	sw := NewSlidingWindow(5, time.Minute)
	sw.Allow()
	if sw.IsIdle(time.Second) {
		t.Fatal("window with recent event should not be idle")
	}
}

func TestSlidingWindow_IsIdle_AfterTimeout(t *testing.T) {
	sw := NewSlidingWindow(5, time.Minute)
	sw.Allow()
	time.Sleep(120 * time.Millisecond)
	if !sw.IsIdle(100 * time.Millisecond) {
		t.Fatal("window should be idle after timeout exceeds last event age")
	}
}

func TestSlidingWindow_Expiry(t *testing.T) {
	sw := NewSlidingWindow(5, 100*time.Millisecond)
	for i := 0; i < 5; i++ {
		if !sw.Allow() {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if sw.Allow() {
		t.Fatal("should be denied at limit")
	}
	time.Sleep(150 * time.Millisecond)
	if !sw.Allow() {
		t.Fatal("should be allowed after window expiry")
	}
}
