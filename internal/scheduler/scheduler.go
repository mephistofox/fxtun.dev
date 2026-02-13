package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/database"
	"github.com/mephistofox/fxtun.dev/internal/exchange"
	"github.com/mephistofox/fxtun.dev/internal/payment"
)

// EventType represents the type of scheduler event
type EventType string

const (
	EventSubscriptionExpiring    EventType = "subscription_expiring"
	EventSubscriptionExpired     EventType = "subscription_expired"
	EventSubscriptionRenewed     EventType = "subscription_renewed"
	EventSubscriptionRenewFailed EventType = "subscription_renew_failed"
	EventPlanChanged             EventType = "plan_changed"
)

// Event represents a scheduler event for notifications
type Event struct {
	Type         EventType
	UserID       int64
	Subscription *database.Subscription
	Plan         *database.Plan
	DaysLeft     int
	Error        error
}

// EventHandler is called when a scheduler event occurs
type EventHandler func(event Event)

// Scheduler handles subscription lifecycle tasks
type Scheduler struct {
	db       *database.Database
	cfg      *config.ServerConfig
	log      zerolog.Logger
	yookassa *payment.YooKassa
	handlers []EventHandler

	// Check intervals
	checkInterval time.Duration
}

// New creates a new scheduler
func New(db *database.Database, cfg *config.ServerConfig, log zerolog.Logger) *Scheduler {
	var yookassa *payment.YooKassa
	if cfg.YooKassa.Enabled {
		yookassa = payment.NewYooKassa(payment.YooKassaConfig{
			ShopID:    cfg.YooKassa.ShopID,
			SecretKey: cfg.YooKassa.SecretKey,
			TestMode:  cfg.YooKassa.TestMode,
			ReturnURL: cfg.YooKassa.ReturnURL,
		})
	}

	return &Scheduler{
		db:            db,
		cfg:           cfg,
		log:           log.With().Str("component", "scheduler").Logger(),
		yookassa:      yookassa,
		checkInterval: 1 * time.Hour,
	}
}

// OnEvent registers an event handler
func (s *Scheduler) OnEvent(handler EventHandler) {
	s.handlers = append(s.handlers, handler)
}

// emit sends event to all handlers
func (s *Scheduler) emit(event Event) {
	for _, h := range s.handlers {
		h(event)
	}
}

// Start begins the scheduler loop
func (s *Scheduler) Start(ctx context.Context) {
	s.log.Info().
		Dur("interval", s.checkInterval).
		Msg("Subscription scheduler started")

	// Run immediately on start
	s.runChecks()

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info().Msg("Subscription scheduler stopped")
			return
		case <-ticker.C:
			s.runChecks()
		}
	}
}

// runChecks performs all scheduled checks
func (s *Scheduler) runChecks() {
	s.log.Debug().Msg("Running subscription checks")

	// 1. Process expired subscriptions (non-recurring or cancelled)
	s.processExpiredSubscriptions()

	// 2. Process recurring renewals
	s.processRecurringRenewals()

	// 3. Apply pending plan changes
	s.applyPlanChanges()

	// 4. Send expiration reminders
	s.sendExpirationReminders()

	// 5. Cleanup stale pending payments
	s.cleanupStalePendingPayments()
}

// processExpiredSubscriptions deactivates expired non-recurring subscriptions
func (s *Scheduler) processExpiredSubscriptions() {
	// Get subscriptions that have expired and are not set for recurring
	subs, err := s.db.Subscriptions.GetExpired()
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get expired subscriptions")
		return
	}

	for _, sub := range subs {
		// Skip recurring subscriptions - they will be handled by renewal process
		if sub.Recurring && sub.Status == database.SubscriptionStatusActive {
			continue
		}

		s.log.Info().
			Int64("subscription_id", sub.ID).
			Int64("user_id", sub.UserID).
			Msg("Deactivating expired subscription")

		// Mark as expired
		sub.Status = database.SubscriptionStatusExpired
		if err := s.db.Subscriptions.Update(sub); err != nil {
			s.log.Error().Err(err).Int64("id", sub.ID).Msg("Failed to update subscription")
			continue
		}

		// Downgrade user to free plan
		if err := s.downgradeToFreePlan(sub.UserID); err != nil {
			s.log.Error().Err(err).Int64("user_id", sub.UserID).Msg("Failed to downgrade user")
			continue
		}

		// Log audit
		_ = s.db.Audit.Log(&sub.UserID, database.ActionSubscriptionExpired, map[string]interface{}{
			"subscription_id": sub.ID,
			"plan_id":         sub.PlanID,
		}, "scheduler")

		// Emit event
		s.emit(Event{
			Type:         EventSubscriptionExpired,
			UserID:       sub.UserID,
			Subscription: sub,
		})
	}
}

// processRecurringRenewals handles automatic renewal of recurring subscriptions
func (s *Scheduler) processRecurringRenewals() {
	if s.yookassa == nil {
		return
	}

	// Get subscriptions expiring within 1 hour that are recurring
	subs, err := s.db.Subscriptions.GetExpiring(1 * time.Hour)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get expiring subscriptions")
		return
	}

	for _, sub := range subs {
		if !sub.Recurring {
			continue
		}

		// Check if subscription has saved payment method for autopayment
		if sub.YooKassaPaymentMethodID == nil || *sub.YooKassaPaymentMethodID == "" {
			s.log.Warn().Int64("subscription_id", sub.ID).Msg("Recurring subscription without payment method")
			continue
		}

		// Check if there's already a pending payment for this subscription
		pendingPayments, err := s.db.Payments.GetPendingBySubscriptionID(sub.ID)
		if err != nil {
			s.log.Error().Err(err).Int64("subscription_id", sub.ID).Msg("Failed to check pending payments")
			continue
		}
		if len(pendingPayments) > 0 {
			s.log.Debug().Int64("subscription_id", sub.ID).Msg("Subscription already has pending payment")
			continue
		}

		// Get plan for pricing
		plan, err := s.db.Plans.GetByID(sub.PlanID)
		if err != nil || plan == nil {
			s.log.Error().Err(err).Int64("plan_id", sub.PlanID).Msg("Failed to get plan")
			continue
		}

		// Free plans don't need renewal
		if plan.Price <= 0 {
			continue
		}

		s.log.Info().
			Int64("subscription_id", sub.ID).
			Int64("user_id", sub.UserID).
			Float64("amount", plan.Price).
			Msg("Processing recurring renewal")

		// Generate new invoice ID
		invoiceID, err := s.db.Payments.GetNextInvoiceID()
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to generate invoice ID")
			continue
		}

		// Convert USD to RUB
		priceRUB := exchange.ConvertUSDToRUB(plan.Price)

		// Create payment record
		pmt := &database.Payment{
			UserID:         sub.UserID,
			SubscriptionID: &sub.ID,
			InvoiceID:      invoiceID,
			Amount:         priceRUB,
			Status:         database.PaymentStatusPending,
			IsRecurring:    true,
		}
		if err := s.db.Payments.Create(pmt); err != nil {
			s.log.Error().Err(err).Msg("Failed to create payment record")
			continue
		}

		// Call YooKassa autopayment API
		yooPayment, err := s.createAutopayment(sub, plan, invoiceID, priceRUB)
		if err != nil {
			s.log.Error().Err(err).
				Int64("invoice_id", invoiceID).
				Str("payment_method_id", *sub.YooKassaPaymentMethodID).
				Msg("Autopayment creation failed")

			pmt.Status = database.PaymentStatusFailed
			_ = s.db.Payments.Update(pmt)

			// Emit failure event
			s.emit(Event{
				Type:         EventSubscriptionRenewFailed,
				UserID:       sub.UserID,
				Subscription: sub,
				Plan:         plan,
				Error:        err,
			})
			continue
		}

		// Save YooKassa payment ID
		yookassaData, _ := json.Marshal(map[string]interface{}{
			"yookassa_payment_id": yooPayment.ID,
			"autopayment":         true,
		})
		pmt.YooKassaData = string(yookassaData)
		_ = s.db.Payments.Update(pmt)

		// Check if payment succeeded immediately (autopayments may succeed without user confirmation)
		if yooPayment.Status == "succeeded" {
			s.handleAutopaymentSuccess(sub, pmt, yooPayment, plan)
		} else {
			s.log.Info().
				Int64("subscription_id", sub.ID).
				Int64("invoice_id", invoiceID).
				Str("yookassa_payment_id", yooPayment.ID).
				Str("status", yooPayment.Status).
				Msg("Autopayment created, waiting for confirmation")
		}
	}
}

// createAutopayment creates an autopayment using saved payment method
func (s *Scheduler) createAutopayment(sub *database.Subscription, plan *database.Plan, invoiceID int64, amount float64) (*payment.Payment, error) {
	idempotencyKey := uuid.New().String()

	req := payment.CreatePaymentRequest{
		Amount: payment.Amount{
			Value:    payment.FormatAmount(amount),
			Currency: "RUB",
		},
		Description:     fmt.Sprintf("fxTunnel %s subscription renewal", plan.Name),
		Capture:         true,
		PaymentMethodID: *sub.YooKassaPaymentMethodID,
		Metadata: map[string]string{
			"invoice_id":      fmt.Sprintf("%d", invoiceID),
			"user_id":         fmt.Sprintf("%d", sub.UserID),
			"subscription_id": fmt.Sprintf("%d", sub.ID),
			"plan_id":         fmt.Sprintf("%d", plan.ID),
			"autopayment":     "true",
		},
	}

	return s.yookassa.CreatePayment(req, idempotencyKey)
}

// handleAutopaymentSuccess processes successful autopayment
func (s *Scheduler) handleAutopaymentSuccess(sub *database.Subscription, pmt *database.Payment, yooPayment *payment.Payment, plan *database.Plan) {
	// Update payment status
	pmt.Status = database.PaymentStatusSuccess
	yookassaData, _ := json.Marshal(map[string]interface{}{
		"yookassa_payment_id": yooPayment.ID,
		"autopayment":         true,
		"paid":                yooPayment.Paid,
	})
	pmt.YooKassaData = string(yookassaData)
	_ = s.db.Payments.Update(pmt)

	// Extend subscription period
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0) // +1 month
	sub.CurrentPeriodStart = &now
	sub.CurrentPeriodEnd = &periodEnd
	sub.Status = database.SubscriptionStatusActive

	// Update payment method if new one was saved
	if yooPayment.PaymentMethod != nil && yooPayment.PaymentMethod.Saved {
		sub.YooKassaPaymentMethodID = &yooPayment.PaymentMethod.ID
	}

	if err := s.db.Subscriptions.Update(sub); err != nil {
		s.log.Error().Err(err).Int64("id", sub.ID).Msg("Failed to extend subscription")
		return
	}

	s.log.Info().
		Int64("subscription_id", sub.ID).
		Int64("user_id", sub.UserID).
		Str("yookassa_payment_id", yooPayment.ID).
		Msg("Subscription renewed via autopayment")

	// Log audit
	_ = s.db.Audit.Log(&sub.UserID, "subscription_renewed", map[string]interface{}{
		"subscription_id":     sub.ID,
		"yookassa_payment_id": yooPayment.ID,
		"plan_id":             sub.PlanID,
		"amount":              pmt.Amount,
	}, "scheduler")

	// Emit success event
	s.emit(Event{
		Type:         EventSubscriptionRenewed,
		UserID:       sub.UserID,
		Subscription: sub,
		Plan:         plan,
	})
}

// applyPlanChanges applies scheduled plan changes
func (s *Scheduler) applyPlanChanges() {
	subs, err := s.db.Subscriptions.GetWithPendingPlanChange()
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get subscriptions with plan changes")
		return
	}

	for _, sub := range subs {
		if sub.NextPlanID == nil {
			continue
		}

		newPlan, err := s.db.Plans.GetByID(*sub.NextPlanID)
		if err != nil || newPlan == nil {
			s.log.Error().Err(err).Int64("plan_id", *sub.NextPlanID).Msg("Failed to get new plan")
			continue
		}

		oldPlanID := sub.PlanID

		s.log.Info().
			Int64("subscription_id", sub.ID).
			Int64("user_id", sub.UserID).
			Int64("old_plan_id", oldPlanID).
			Int64("new_plan_id", *sub.NextPlanID).
			Msg("Applying scheduled plan change")

		// Update subscription
		sub.PlanID = *sub.NextPlanID
		sub.NextPlanID = nil
		if err := s.db.Subscriptions.Update(sub); err != nil {
			s.log.Error().Err(err).Int64("id", sub.ID).Msg("Failed to update subscription")
			continue
		}

		// Update user plan
		user, err := s.db.Users.GetByID(sub.UserID)
		if err == nil && user != nil {
			user.PlanID = sub.PlanID
			if err := s.db.Users.Update(user); err != nil {
				s.log.Error().Err(err).Int64("user_id", sub.UserID).Msg("Failed to update user plan")
			}
		}

		// Log audit
		_ = s.db.Audit.Log(&sub.UserID, database.ActionSubscriptionChanged, map[string]interface{}{
			"subscription_id": sub.ID,
			"old_plan_id":     oldPlanID,
			"new_plan_id":     sub.PlanID,
		}, "scheduler")

		// Emit event
		s.emit(Event{
			Type:         EventPlanChanged,
			UserID:       sub.UserID,
			Subscription: sub,
			Plan:         newPlan,
		})
	}
}

// sendExpirationReminders sends reminders for expiring subscriptions
func (s *Scheduler) sendExpirationReminders() {
	// Check subscriptions expiring in 3 days
	s.checkExpiringSubscriptions(3)
	// Check subscriptions expiring in 1 day
	s.checkExpiringSubscriptions(1)
}

// checkExpiringSubscriptions checks for subscriptions expiring in given days
func (s *Scheduler) checkExpiringSubscriptions(daysAhead int) {
	subs, err := s.db.Subscriptions.GetForRenewalReminder(daysAhead)
	if err != nil {
		s.log.Error().Err(err).Int("days", daysAhead).Msg("Failed to get subscriptions for reminder")
		return
	}

	for _, sub := range subs {
		plan, _ := s.db.Plans.GetByID(sub.PlanID)

		s.log.Debug().
			Int64("subscription_id", sub.ID).
			Int64("user_id", sub.UserID).
			Int("days_left", daysAhead).
			Msg("Subscription expiring soon")

		// Emit event for notification
		s.emit(Event{
			Type:         EventSubscriptionExpiring,
			UserID:       sub.UserID,
			Subscription: sub,
			Plan:         plan,
			DaysLeft:     daysAhead,
		})
	}
}

// downgradeToFreePlan downgrades user to the free plan
func (s *Scheduler) downgradeToFreePlan(userID int64) error {
	// Find free plan (price = 0)
	freePlan, err := s.db.Plans.GetBySlug("free")
	if err != nil || freePlan == nil {
		// Fallback: get first plan with price 0
		plans, _, err := s.db.Plans.ListAll(100, 0)
		if err != nil {
			return err
		}
		for _, p := range plans {
			if p.Price == 0 {
				freePlan = p
				break
			}
		}
	}

	if freePlan == nil {
		s.log.Warn().Int64("user_id", userID).Msg("No free plan found, keeping current plan")
		return nil
	}

	user, err := s.db.Users.GetByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	user.PlanID = freePlan.ID
	return s.db.Users.Update(user)
}

// cleanupStalePendingPayments deletes pending payments older than 24 hours
func (s *Scheduler) cleanupStalePendingPayments() {
	deleted, err := s.db.Payments.DeleteStalePending(24 * time.Hour)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to cleanup stale pending payments")
		return
	}

	if deleted > 0 {
		s.log.Info().Int64("count", deleted).Msg("Cleaned up stale pending payments")
	}
}

// RunOnce runs all checks once (useful for testing)
func (s *Scheduler) RunOnce() {
	s.runChecks()
}
