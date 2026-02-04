package scheduler

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/database"
)

func setupTestDB(t *testing.T) *database.Database {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "scheduler_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	log := zerolog.New(zerolog.NewTestWriter(t))
	db, err := database.New(tmpFile.Name(), log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestScheduler_New(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{
		Robokassa: config.RobokassaSettings{
			Enabled:       true,
			MerchantLogin: "test",
			Password1:     "pass1",
			Password2:     "pass2",
			TestMode:      true,
		},
	}
	log := zerolog.New(zerolog.NewTestWriter(t))

	s := New(db, cfg, log)
	if s == nil {
		t.Fatal("Expected scheduler to be created")
	}
	if s.robokassa == nil {
		t.Fatal("Expected robokassa client to be created")
	}
}

func TestScheduler_NewWithoutRobokassa(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{
		Robokassa: config.RobokassaSettings{
			Enabled: false,
		},
	}
	log := zerolog.New(zerolog.NewTestWriter(t))

	s := New(db, cfg, log)
	if s == nil {
		t.Fatal("Expected scheduler to be created")
	}
	if s.robokassa != nil {
		t.Fatal("Expected robokassa client to be nil when disabled")
	}
}

func TestScheduler_OnEvent(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{}
	log := zerolog.New(zerolog.NewTestWriter(t))

	s := New(db, cfg, log)

	var receivedEvent *Event
	s.OnEvent(func(e Event) {
		receivedEvent = &e
	})

	testEvent := Event{
		Type:     EventSubscriptionExpired,
		UserID:   123,
		DaysLeft: 3,
	}
	s.emit(testEvent)

	if receivedEvent == nil {
		t.Fatal("Expected event to be received")
	}
	if receivedEvent.Type != EventSubscriptionExpired {
		t.Errorf("Expected event type %s, got %s", EventSubscriptionExpired, receivedEvent.Type)
	}
	if receivedEvent.UserID != 123 {
		t.Errorf("Expected user ID 123, got %d", receivedEvent.UserID)
	}
}

func TestScheduler_RunOnce(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{}
	log := zerolog.New(zerolog.NewTestWriter(t))

	s := New(db, cfg, log)

	// Should not panic with empty database
	s.RunOnce()
}

func TestScheduler_StartAndStop(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{}
	log := zerolog.New(zerolog.NewTestWriter(t))

	s := New(db, cfg, log)
	s.checkInterval = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()

	// Let it run a few cycles
	time.Sleep(250 * time.Millisecond)

	cancel()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Scheduler did not stop in time")
	}
}

func TestScheduler_ProcessExpiredSubscriptions(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{}
	log := zerolog.New(zerolog.NewTestWriter(t))

	// Get free plan created by migration
	freePlan, err := db.Plans.GetBySlug("free")
	if err != nil {
		t.Fatalf("Failed to get free plan: %v", err)
	}

	// Get paid plan created by migration
	paidPlan, err := db.Plans.GetBySlug("pro")
	if err != nil {
		t.Fatalf("Failed to get paid plan: %v", err)
	}

	// Create a user with paid plan
	user := &database.User{
		Phone:        "+79001234567",
		PasswordHash: "hash",
		PlanID:       paidPlan.ID,
		IsActive:     true,
	}
	if err := db.Users.Create(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create expired subscription (non-recurring)
	expiredTime := time.Now().Add(-1 * time.Hour)
	startTime := expiredTime.Add(-30 * 24 * time.Hour)
	sub := &database.Subscription{
		UserID:             user.ID,
		PlanID:             paidPlan.ID,
		Status:             database.SubscriptionStatusActive,
		Recurring:          false,
		CurrentPeriodStart: &startTime,
		CurrentPeriodEnd:   &expiredTime,
	}
	if err := db.Subscriptions.Create(sub); err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	s := New(db, cfg, log)

	var expiredEvents []Event
	s.OnEvent(func(e Event) {
		if e.Type == EventSubscriptionExpired {
			expiredEvents = append(expiredEvents, e)
		}
	})

	s.RunOnce()

	// Check subscription was expired
	updatedSub, err := db.Subscriptions.GetByID(sub.ID)
	if err != nil {
		t.Fatalf("Failed to get subscription: %v", err)
	}
	if updatedSub.Status != database.SubscriptionStatusExpired {
		t.Errorf("Expected subscription status to be expired, got %s", updatedSub.Status)
	}

	// Check user was downgraded to free plan
	updatedUser, err := db.Users.GetByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if updatedUser.PlanID != freePlan.ID {
		t.Errorf("Expected user plan to be free (%d), got %d", freePlan.ID, updatedUser.PlanID)
	}

	// Check event was emitted
	if len(expiredEvents) != 1 {
		t.Errorf("Expected 1 expired event, got %d", len(expiredEvents))
	}
}
