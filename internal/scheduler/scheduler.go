package scheduler

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/database"
	"github.com/mephistofox/fxtunnel/internal/payment"
)

// EventType represents the type of scheduler event
type EventType string

const (
	EventSubscriptionExpiring   EventType = "subscription_expiring"
	EventSubscriptionExpired    EventType = "subscription_expired"
	EventSubscriptionRenewed    EventType = "subscription_renewed"
	EventSubscriptionRenewFailed EventType = "subscription_renew_failed"
	EventPlanChanged            EventType = "plan_changed"
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
	db        *database.Database
	cfg       *config.ServerConfig
	log       zerolog.Logger
	robokassa *payment.Robokassa
	handlers  []EventHandler

	// Check intervals
	checkInterval time.Duration
}

// New creates a new scheduler
func New(db *database.Database, cfg *config.ServerConfig, log zerolog.Logger) *Scheduler {
	var robokassa *payment.Robokassa
	if cfg.Robokassa.Enabled {
		robokassa = payment.NewRobokassa(payment.RobokassaConfig{
			MerchantLogin: cfg.Robokassa.MerchantLogin,
			Password1:     cfg.Robokassa.Password1,
			Password2:     cfg.Robokassa.Password2,
			TestPassword1: cfg.Robokassa.TestPassword1,
			TestPassword2: cfg.Robokassa.TestPassword2,
			TestMode:      cfg.Robokassa.TestMode,
		})
	}

	return &Scheduler{
		db:            db,
		cfg:           cfg,
		log:           log.With().Str("component", "scheduler").Logger(),
		robokassa:     robokassa,
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
		s.db.Audit.Log(&sub.UserID, database.ActionSubscriptionExpired, map[string]interface{}{
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
	if s.robokassa == nil {
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

		// Check if subscription has previous invoice for recurring
		if sub.RobokassaInvoiceID == nil {
			s.log.Warn().Int64("subscription_id", sub.ID).Msg("Recurring subscription without invoice ID")
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

		// Create payment record
		pmt := &database.Payment{
			UserID:         sub.UserID,
			SubscriptionID: &sub.ID,
			InvoiceID:      invoiceID,
			Amount:         plan.Price,
			Status:         database.PaymentStatusPending,
			IsRecurring:    true,
		}
		if err := s.db.Payments.Create(pmt); err != nil {
			s.log.Error().Err(err).Msg("Failed to create payment record")
			continue
		}

		// Call Robokassa recurring API
		success, err := s.callRecurringAPI(invoiceID, *sub.RobokassaInvoiceID, plan.Price)
		if err != nil {
			s.log.Error().Err(err).
				Int64("invoice_id", invoiceID).
				Int64("prev_invoice_id", *sub.RobokassaInvoiceID).
				Msg("Recurring payment API call failed")

			pmt.Status = database.PaymentStatusFailed
			s.db.Payments.Update(pmt)

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

		if success {
			// Payment initiated - Robokassa will send ResultURL callback
			s.log.Info().
				Int64("subscription_id", sub.ID).
				Int64("invoice_id", invoiceID).
				Msg("Recurring payment initiated")
		}
	}
}

// callRecurringAPI calls Robokassa recurring payment API
func (s *Scheduler) callRecurringAPI(invoiceID, previousInvoiceID int64, amount float64) (bool, error) {
	url, values := s.robokassa.GenerateRecurringPaymentURL(payment.RecurringPaymentParams{
		InvoiceID:         invoiceID,
		PreviousInvoiceID: previousInvoiceID,
		OutSum:            amount,
	})

	resp, err := http.PostForm(url, values)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		s.log.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("Robokassa recurring API error")
		return false, nil
	}

	s.log.Debug().
		Int64("invoice_id", invoiceID).
		Str("response", string(body)).
		Msg("Robokassa recurring API response")

	return true, nil
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
		s.db.Audit.Log(&sub.UserID, database.ActionSubscriptionChanged, map[string]interface{}{
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

// RunOnce runs all checks once (useful for testing)
func (s *Scheduler) RunOnce() {
	s.runChecks()
}
