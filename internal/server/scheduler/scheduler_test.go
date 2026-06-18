package scheduler

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/server/database"
	"github.com/mephistofox/fxtunnel/internal/server/payment"
)

// setupTestDB connects to a Postgres instance given by FXTUNNEL_TEST_DSN,
// runs migrations (via database.New) and truncates mutable tables so each
// test starts from a clean slate while keeping the seeded plans. The test is
// skipped when no DSN is configured, so the suite stays green in environments
// without a database.
func setupTestDB(t *testing.T) *database.Database {
	t.Helper()

	dsn := os.Getenv("FXTUNNEL_TEST_DSN")
	if dsn == "" {
		t.Skip("FXTUNNEL_TEST_DSN not set; skipping Postgres-backed scheduler test")
	}

	log := zerolog.New(zerolog.NewTestWriter(t))
	db, err := database.New(dsn, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if _, err := db.Pool().Exec(context.Background(),
		"TRUNCATE users, subscriptions, payments, audit_logs RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("Failed to reset test tables: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestScheduler_New(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{
		YooKassa: config.YooKassaSettings{
			Enabled:   true,
			ShopID:    "test",
			SecretKey: "test_secret",
			TestMode:  true,
		},
	}
	log := zerolog.New(zerolog.NewTestWriter(t))

	providers := payment.NewRegistry()
	providers.Register(payment.NewYooKassa(payment.YooKassaConfig{
		ShopID:    cfg.YooKassa.ShopID,
		SecretKey: cfg.YooKassa.SecretKey,
		TestMode:  cfg.YooKassa.TestMode,
	}))

	s := New(db, cfg, providers, log)
	if s == nil {
		t.Fatal("Expected scheduler to be created")
	}
	if s.providers == nil || !s.providers.Has("yookassa") {
		t.Fatal("Expected yookassa provider to be registered")
	}
}

func TestScheduler_NewWithoutYooKassa(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{
		YooKassa: config.YooKassaSettings{
			Enabled: false,
		},
	}
	log := zerolog.New(zerolog.NewTestWriter(t))

	providers := payment.NewRegistry()

	s := New(db, cfg, providers, log)
	if s == nil {
		t.Fatal("Expected scheduler to be created")
	}
	if s.providers.Has("yookassa") {
		t.Fatal("Expected yookassa provider to not be registered when disabled")
	}
}

func TestScheduler_OnEvent(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{}
	log := zerolog.New(zerolog.NewTestWriter(t))

	s := New(db, cfg, nil, log)

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

	s := New(db, cfg, nil, log)

	// Should not panic with empty database
	s.RunOnce()
}

func TestScheduler_StartAndStop(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{}
	log := zerolog.New(zerolog.NewTestWriter(t))

	s := New(db, cfg, nil, log)
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

// TestScheduler_ExpiredSubscriptionDoesNotApplyScheduledUpgrade reproduces the
// plan-upgrade bypass: a subscription that expires at period end must drop the
// user to the free plan, NOT have its pending next_plan_id applied. Otherwise
// any subscriber could schedule an upgrade to a pricier plan, let the cheap
// subscription lapse, and keep the pricier plan permanently for free.
func TestScheduler_ExpiredSubscriptionDoesNotApplyScheduledUpgrade(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{}
	log := zerolog.New(zerolog.NewTestWriter(t))

	freePlan, err := db.Plans.GetBySlug("free")
	if err != nil {
		t.Fatalf("Failed to get free plan: %v", err)
	}
	basePlan, err := db.Plans.GetBySlug("base")
	if err != nil {
		t.Fatalf("Failed to get base plan: %v", err)
	}
	businessPlan, err := db.Plans.GetBySlug("business")
	if err != nil {
		t.Fatalf("Failed to get business plan: %v", err)
	}

	// User on the cheapest paid plan.
	user := &database.User{
		Phone:        "+79990001122",
		PasswordHash: "hash",
		PlanID:       basePlan.ID,
		IsActive:     true,
	}
	if err := db.Users.Create(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Active, non-recurring subscription whose period has ended, with a pending
	// upgrade to the priciest plan staged in next_plan_id.
	expiredTime := time.Now().Add(-1 * time.Hour)
	startTime := expiredTime.Add(-30 * 24 * time.Hour)
	sub := &database.Subscription{
		UserID:             user.ID,
		PlanID:             basePlan.ID,
		NextPlanID:         &businessPlan.ID,
		Status:             database.SubscriptionStatusActive,
		Recurring:          false,
		CurrentPeriodStart: &startTime,
		CurrentPeriodEnd:   &expiredTime,
	}
	if err := db.Subscriptions.Create(sub); err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	s := New(db, cfg, nil, log)
	s.RunOnce()

	updatedUser, err := db.Users.GetByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if updatedUser.PlanID != freePlan.ID {
		t.Errorf("expected user downgraded to free plan (%d), got %d — scheduled upgrade was applied to an expired subscription (free paid plan)",
			freePlan.ID, updatedUser.PlanID)
	}

	updatedSub, err := db.Subscriptions.GetByID(sub.ID)
	if err != nil {
		t.Fatalf("Failed to get subscription: %v", err)
	}
	if updatedSub.Status != database.SubscriptionStatusExpired {
		t.Errorf("expected subscription status expired, got %s", updatedSub.Status)
	}
	if updatedSub.NextPlanID != nil {
		t.Errorf("expected next_plan_id cleared on expiry, still set to %d", *updatedSub.NextPlanID)
	}
}

// TestScheduler_RunChecksSkipsWhenLockHeld verifies the cluster advisory lock:
// while another holder owns the lock, runChecks must skip (no node double-runs).
func TestScheduler_RunChecksSkipsWhenLockHeld(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.ServerConfig{}
	log := zerolog.New(zerolog.NewTestWriter(t))

	paidPlan, err := db.Plans.GetBySlug("pro")
	if err != nil {
		t.Fatalf("pro plan: %v", err)
	}
	user := &database.User{Phone: "+79992223344", PasswordHash: "hash", PlanID: paidPlan.ID, IsActive: true}
	if err := db.Users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	expired := time.Now().Add(-1 * time.Hour)
	start := expired.Add(-30 * 24 * time.Hour)
	sub := &database.Subscription{
		UserID: user.ID, PlanID: paidPlan.ID, Status: database.SubscriptionStatusActive,
		Recurring: false, CurrentPeriodStart: &start, CurrentPeriodEnd: &expired,
	}
	if err := db.Subscriptions.Create(sub); err != nil {
		t.Fatalf("create sub: %v", err)
	}

	// Hold the advisory lock on a separate connection (simulating another node).
	ctx := context.Background()
	conn, err := db.Pool().Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer conn.Release()
	var locked bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", schedulerAdvisoryLockKey).Scan(&locked); err != nil || !locked {
		t.Fatalf("hold lock: locked=%v err=%v", locked, err)
	}

	s := New(db, cfg, nil, log)
	s.RunOnce() // must skip — lock is held elsewhere

	got, err := db.Subscriptions.GetByID(sub.ID)
	if err != nil {
		t.Fatalf("get sub: %v", err)
	}
	if got.Status != database.SubscriptionStatusActive {
		t.Fatalf("expected subscription untouched while lock held, got status %s", got.Status)
	}

	// Release the lock; now runChecks should process the expired subscription.
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", schedulerAdvisoryLockKey); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	s.RunOnce()

	got, err = db.Subscriptions.GetByID(sub.ID)
	if err != nil {
		t.Fatalf("get sub: %v", err)
	}
	if got.Status != database.SubscriptionStatusExpired {
		t.Fatalf("expected subscription processed after lock release, got status %s", got.Status)
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

	s := New(db, cfg, nil, log)

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
