package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/api/dto"
	"github.com/mephistofox/fxtunnel/internal/server/auth"
	"github.com/mephistofox/fxtunnel/internal/server/database"
	"github.com/mephistofox/fxtunnel/internal/server/exchange"
	"github.com/mephistofox/fxtunnel/internal/server/payment"
)

// handleGetSubscription returns the current user's subscription
func (s *Server) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sub, err := s.db.Subscriptions.GetByUserID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get subscription")
		s.respondError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}

	resp := dto.SubscriptionResponse{
		HasActive: false,
	}

	if sub != nil {
		subDTO := dto.SubscriptionFromModel(sub)

		// Load plan
		if plan, err := s.db.Plans.GetByID(sub.PlanID); err == nil {
			subDTO.Plan = dto.PlanFromModel(plan)
		}

		// Load next plan if scheduled
		if sub.NextPlanID != nil {
			if nextPlan, err := s.db.Plans.GetByID(*sub.NextPlanID); err == nil {
				subDTO.NextPlan = dto.PlanFromModel(nextPlan)
			}
		}

		resp.Subscription = subDTO
		resp.HasActive = sub.Status == database.SubscriptionStatusActive ||
			(sub.Status == database.SubscriptionStatusCancelled && sub.CurrentPeriodEnd != nil && sub.CurrentPeriodEnd.After(time.Now()))
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// extractDomainFromHost extracts domain from host (removes port)
func extractDomainFromHost(host string) string {
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}

// isPaymentEnabledForDomain checks if payments are enabled for the given domain
func (s *Server) isPaymentEnabledForDomain(host string) (bool, string) {
	domain := extractDomainFromHost(host)

	// Check per-domain settings
	if s.cfg.Payments.Domains != nil {
		if settings, ok := s.cfg.Payments.Domains[domain]; ok {
			if !settings.Enabled {
				msg := settings.Message
				if msg == "" {
					msg = "payments are not available for this domain"
				}
				return false, msg
			}
		}
	}

	// Default: enabled if any provider is enabled
	return s.cfg.YooKassa.Enabled || s.cfg.Creem.Enabled, "payments are not enabled"
}

// getPaymentProvider resolves the payment provider for the given host
func (s *Server) getPaymentProvider(host string) (payment.Provider, error) {
	if s.paymentProviders == nil {
		return nil, fmt.Errorf("payment providers not configured")
	}

	domain := extractDomainFromHost(host)

	// Check per-domain settings
	if s.cfg.Payments.Domains != nil {
		if settings, ok := s.cfg.Payments.Domains[domain]; ok {
			if !settings.Enabled {
				return nil, fmt.Errorf("payments disabled for domain %s", domain)
			}
			return s.paymentProviders.Get(settings.Provider)
		}
	}

	// Default: try yookassa
	if s.cfg.YooKassa.Enabled {
		return s.paymentProviders.Get("yookassa")
	}

	return nil, fmt.Errorf("no payment provider configured")
}

// handleCheckout creates a payment and returns the payment URL
func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check if payments are enabled for this domain
	enabled, msg := s.isPaymentEnabledForDomain(r.Host)
	if !enabled {
		s.respondError(w, http.StatusServiceUnavailable, msg)
		return
	}

	// Resolve payment provider
	provider, err := s.getPaymentProvider(r.Host)
	if err != nil {
		s.log.Error().Err(err).Str("host", r.Host).Msg("Failed to resolve payment provider")
		s.respondError(w, http.StatusServiceUnavailable, "payment provider not available")
		return
	}

	s.log.Info().
		Str("host", r.Host).
		Str("domain", extractDomainFromHost(r.Host)).
		Str("provider", provider.Name()).
		Interface("payment_domains", s.cfg.Payments.Domains).
		Msg("Checkout: resolved payment provider")

	var req dto.CheckoutRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get the plan
	plan, err := s.db.Plans.GetByID(req.PlanID)
	if err != nil || plan == nil {
		s.respondError(w, http.StatusBadRequest, "invalid plan")
		return
	}

	// Check if plan is available
	if !plan.IsPublic && !user.IsAdmin {
		s.respondError(w, http.StatusForbidden, "plan not available")
		return
	}

	// Free plans don't need payment
	if plan.Price <= 0 {
		s.respondError(w, http.StatusBadRequest, "free plans don't require payment")
		return
	}

	// Check for existing active subscription
	existingSub, _ := s.db.Subscriptions.GetByUserID(user.ID)
	if existingSub != nil && existingSub.Status == database.SubscriptionStatusActive {
		s.respondError(w, http.StatusBadRequest, "active subscription exists, use plan change instead")
		return
	}

	// Check for existing pending subscription (separate query since GetByUserID only returns active/cancelled)
	pendingSub, _ := s.db.Subscriptions.GetPendingByUserID(user.ID)
	if pendingSub != nil {
		s.respondError(w, http.StatusBadRequest, "pending payment already exists, please complete or wait")
		return
	}

	// Generate invoice ID
	invoiceID, err := s.db.Payments.GetNextInvoiceID()
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to generate invoice ID")
		s.respondError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	// Creem manages subscriptions itself, so always mark as recurring
	recurring := req.Recurring
	if provider.Name() == "creem" {
		recurring = true
	}

	// Create subscription record (pending)
	sub := &database.Subscription{
		UserID:    user.ID,
		PlanID:    plan.ID,
		Status:    database.SubscriptionStatusPending,
		Recurring: recurring,
	}
	if err := s.db.Subscriptions.Create(sub); err != nil {
		s.log.Error().Err(err).Msg("Failed to create subscription")
		s.respondError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}

	// Get user email
	dbUser, _ := s.db.Users.GetByID(user.ID)
	email := ""
	if dbUser != nil {
		email = dbUser.Email
	}

	// Determine amount and currency based on provider
	var amount float64
	var currency string
	providerName := provider.Name()

	switch providerName {
	case "creem":
		amount = plan.Price // USD
		currency = "USD"
	default: // yookassa
		amount = exchange.ConvertUSDToRUB(plan.Price)
		currency = "RUB"
	}

	// Create payment record
	pmt := &database.Payment{
		UserID:         user.ID,
		SubscriptionID: &sub.ID,
		InvoiceID:      invoiceID,
		Amount:         amount,
		Status:         database.PaymentStatusPending,
		IsRecurring:    recurring,
		Provider:       providerName,
	}
	if err := s.db.Payments.Create(pmt); err != nil {
		s.log.Error().Err(err).Msg("Failed to create payment")
		s.respondError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	// Create checkout session via provider
	result, err := provider.CreateCheckoutSession(payment.CheckoutParams{
		ProductID:      plan.CreemProductID,
		InvoiceID:      invoiceID,
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		PlanID:         plan.ID,
		PlanName:       plan.Name,
		Amount:         amount,
		Currency:       currency,
		Email:          email,
		Recurring:      recurring,
		Description:    fmt.Sprintf("fxTunnel %s subscription", plan.Name),
	})
	if err != nil {
		s.log.Error().Err(err).Str("provider", providerName).Msg("Failed to create checkout session")
		s.respondError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	// Save provider data
	providerData, _ := json.Marshal(result.Metadata)
	pmt.ProviderData = string(providerData)
	if err := s.db.Payments.Update(pmt); err != nil {
		s.log.Error().Err(err).Msg("Failed to update payment with provider data")
	}

	// For Creem: save customer ID on subscription
	if providerName == "creem" && result.ProviderCustomerID != "" {
		sub.CreemCustomerID = &result.ProviderCustomerID
		if err := s.db.Subscriptions.Update(sub); err != nil {
			s.log.Error().Err(err).Msg("Failed to save Creem customer ID")
		}
	}

	if result.PaymentURL == "" {
		s.log.Error().Str("provider", providerName).Msg("No payment URL in checkout result")
		s.respondError(w, http.StatusInternalServerError, "failed to get payment URL")
		return
	}

	// Log audit
	_ = s.db.Audit.Log(&user.ID, "payment_initiated", map[string]interface{}{
		"invoice_id":          invoiceID,
		"provider":            providerName,
		"provider_payment_id": result.ProviderPaymentID,
		"plan_id":             plan.ID,
		"amount":              amount,
		"currency":            currency,
		"recurring":           recurring,
	}, auth.GetClientIP(r))

	s.respondJSON(w, http.StatusOK, dto.CheckoutResponse{
		PaymentURL: result.PaymentURL,
		InvoiceID:  invoiceID,
	})
}

// activateSubscription activates a subscription after successful payment
func (s *Server) activateSubscription(sub *database.Subscription, pmt *database.Payment, providerName string) {
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0) // +1 month
	sub.Status = database.SubscriptionStatusActive
	sub.CurrentPeriodStart = &now
	sub.CurrentPeriodEnd = &periodEnd

	if err := s.db.Subscriptions.Update(sub); err != nil {
		s.log.Error().Err(err).Msg("Failed to activate subscription")
		return
	}

	// Update user's plan
	if user, err := s.db.Users.GetByID(sub.UserID); err == nil && user != nil {
		user.PlanID = sub.PlanID
		if err := s.db.Users.Update(user); err != nil {
			s.log.Error().Err(err).Msg("Failed to update user plan")
		}
	}

	s.log.Info().
		Int64("user_id", sub.UserID).
		Int64("plan_id", sub.PlanID).
		Str("provider", providerName).
		Msg("Subscription activated")

	// Log audit
	_ = s.db.Audit.Log(&sub.UserID, "subscription_activated", map[string]interface{}{
		"invoice_id":      pmt.InvoiceID,
		"plan_id":         sub.PlanID,
		"subscription_id": sub.ID,
		"provider":        providerName,
	}, "webhook")

	// Send payment success email notification
	if s.notifier != nil {
		plan, _ := s.db.Plans.GetByID(sub.PlanID)
		planName := "Unknown"
		if plan != nil {
			planName = plan.Name
		}
		if err := s.notifier.SendPaymentSuccessNotification(sub.UserID, planName, pmt.Amount, providerName); err != nil {
			s.log.Error().Err(err).Int64("user_id", sub.UserID).Msg("Failed to send payment success email")
		}
	}

	// Send Telegram admin notification
	if s.telegramNotifier != nil {
		plan, _ := s.db.Plans.GetByID(sub.PlanID)
		planName := "Unknown"
		if plan != nil {
			planName = plan.Name
		}
		userName := ""
		if u, err := s.db.Users.GetByID(sub.UserID); err == nil {
			userName = u.DisplayName
		}
		s.telegramNotifier.NotifyNewSubscription(sub.UserID, userName, planName, pmt.Amount, providerName)
	}
}

// handlePaymentWebhook handles YooKassa webhook notifications (POST)
func (s *Server) handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	s.log.Info().
		Str("remote_addr", r.RemoteAddr).              // Post-RealIP = actual client IP from nginx
		Str("original_tcp_addr", getOriginalRemoteAddr(r)). // Raw TCP = nginx 127.0.0.1
		Str("method", r.Method).
		Msg("YooKassa webhook received")

	if !s.cfg.YooKassa.Enabled {
		http.Error(w, "payments disabled", http.StatusServiceUnavailable)
		return
	}

	// Verify IP using r.RemoteAddr which contains the real client IP.
	// Behind nginx (trusted proxy), nginx sets X-Real-IP from the actual
	// client address, and middleware.RealIP copies it into r.RemoteAddr.
	// Do NOT use getOriginalRemoteAddr() here — it returns the raw TCP
	// peer address which is 127.0.0.1 (nginx itself) behind a reverse proxy.
	if !s.cfg.YooKassa.TestMode {
		if !payment.IsYooKassaIP(r.RemoteAddr) {
			s.log.Warn().
				Str("remote_addr", r.RemoteAddr).
				Str("original_tcp_addr", getOriginalRemoteAddr(r)).
				Str("x_forwarded_for", r.Header.Get("X-Forwarded-For")).
				Msg("Webhook from unauthorized IP")
			http.Error(w, "unauthorized", http.StatusForbidden)
			return
		}
	}

	// Limit request body to 1MB to prevent abuse
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to read webhook body")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Parse webhook event
	event, err := payment.ParseWebhookEvent(body)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to parse webhook event")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	s.log.Info().
		Str("type", event.Type).
		Str("event", event.Event).
		Str("payment_id", event.Object.ID).
		Str("status", event.Object.Status).
		Msg("Webhook event parsed")

	// Handle different event types
	switch event.Event {
	case "payment.succeeded":
		s.handlePaymentSucceeded(w, event.Object)
	case "payment.canceled":
		s.handlePaymentCanceled(w, event.Object)
	case "payment.waiting_for_capture":
		// We use immediate capture, so this shouldn't happen
		s.log.Info().Str("payment_id", event.Object.ID).Msg("Payment waiting for capture (ignored)")
		w.WriteHeader(http.StatusOK)
	default:
		s.log.Info().Str("event", event.Event).Msg("Unknown webhook event (ignored)")
		w.WriteHeader(http.StatusOK)
	}
}

// parseYooKassaMetadata extracts user_id, subscription_id, plan_id from YooKassa payment metadata
func parseYooKassaMetadata(metadata map[string]string) (userID, subscriptionID, planID int64) {
	if s, ok := metadata["user_id"]; ok {
		_, _ = fmt.Sscanf(s, "%d", &userID)
	}
	if s, ok := metadata["subscription_id"]; ok {
		_, _ = fmt.Sscanf(s, "%d", &subscriptionID)
	}
	if s, ok := metadata["plan_id"]; ok {
		_, _ = fmt.Sscanf(s, "%d", &planID)
	}
	return
}

// handlePaymentSucceeded processes successful payment webhook
func (s *Server) handlePaymentSucceeded(w http.ResponseWriter, yooPayment *payment.Payment) {
	// Extract all metadata upfront for logging
	metaUserID, metaSubID, metaPlanID := parseYooKassaMetadata(yooPayment.Metadata)

	// Get invoice_id from metadata
	invoiceIDStr, ok := yooPayment.Metadata["invoice_id"]
	if !ok {
		s.log.Error().
			Str("payment_id", yooPayment.ID).
			Int64("meta_user_id", metaUserID).
			Msg("No invoice_id in payment metadata")
		http.Error(w, "invalid payment metadata", http.StatusBadRequest)
		return
	}

	var invoiceID int64
	if _, err := fmt.Sscanf(invoiceIDStr, "%d", &invoiceID); err != nil {
		s.log.Error().Err(err).Str("invoice_id_str", invoiceIDStr).Msg("Invalid invoice_id format")
		http.Error(w, "invalid invoice_id", http.StatusBadRequest)
		return
	}

	// Get payment record
	pmt, err := s.db.Payments.GetByInvoiceID(invoiceID)
	if err != nil || pmt == nil {
		s.log.Error().Err(err).
			Int64("invoice_id", invoiceID).
			Int64("meta_user_id", metaUserID).
			Int64("meta_subscription_id", metaSubID).
			Int64("meta_plan_id", metaPlanID).
			Str("yookassa_payment_id", yooPayment.ID).
			Msg("Payment not found — likely deleted by stale cleanup. Attempting recovery")

		// Recover: recreate payment and subscription from metadata
		if metaUserID > 0 && metaPlanID > 0 {
			pmt, err = s.recoverPaymentFromWebhook(invoiceID, metaUserID, metaSubID, metaPlanID, yooPayment)
			if err != nil {
				s.log.Error().Err(err).
					Int64("invoice_id", invoiceID).
					Int64("meta_user_id", metaUserID).
					Msg("Failed to recover payment from webhook")
				http.Error(w, "recovery failed", http.StatusInternalServerError)
				return
			}
			s.log.Info().
				Int64("invoice_id", invoiceID).
				Int64("user_id", metaUserID).
				Msg("Payment recovered from webhook metadata")
		} else {
			http.Error(w, "payment not found", http.StatusNotFound)
			return
		}
	}

	// Already processed
	if pmt.Status == database.PaymentStatusSuccess {
		s.log.Info().Int64("invoice_id", invoiceID).Msg("Payment already processed")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Verify payment amount matches expected amount
	if yooPayment.Amount.Value != "" {
		var webhookAmount float64
		if _, err := fmt.Sscanf(yooPayment.Amount.Value, "%f", &webhookAmount); err == nil {
			if webhookAmount < pmt.Amount*0.99 {
				s.log.Error().
					Float64("expected", pmt.Amount).
					Float64("received", webhookAmount).
					Int64("invoice_id", invoiceID).
					Int64("user_id", pmt.UserID).
					Msg("Payment amount mismatch")
				http.Error(w, "amount mismatch", http.StatusBadRequest)
				return
			}
		}
	}

	// Update payment status
	pmt.Status = database.PaymentStatusSuccess

	// Save YooKassa data including payment_method_id for recurring
	yookassaData := map[string]interface{}{
		"yookassa_payment_id": yooPayment.ID,
		"paid":                yooPayment.Paid,
		"test":                yooPayment.Test,
	}
	if yooPayment.PaymentMethod != nil {
		yookassaData["payment_method_type"] = yooPayment.PaymentMethod.Type
		yookassaData["payment_method_title"] = yooPayment.PaymentMethod.Title
		if yooPayment.PaymentMethod.Saved {
			yookassaData["payment_method_id"] = yooPayment.PaymentMethod.ID
		}
	}
	data, _ := json.Marshal(yookassaData)
	pmt.YooKassaData = string(data)

	if err := s.db.Payments.Update(pmt); err != nil {
		s.log.Error().Err(err).Int64("user_id", pmt.UserID).Msg("Failed to update payment")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Activate subscription
	if pmt.SubscriptionID != nil {
		sub, err := s.db.Subscriptions.GetByID(*pmt.SubscriptionID)
		if err != nil {
			s.log.Error().Err(err).
				Int64("subscription_id", *pmt.SubscriptionID).
				Int64("user_id", pmt.UserID).
				Int64("invoice_id", invoiceID).
				Msg("Failed to get subscription for activation")
			http.Error(w, "subscription lookup failed", http.StatusInternalServerError)
			return
		}
		if sub == nil {
			s.log.Error().
				Int64("subscription_id", *pmt.SubscriptionID).
				Int64("user_id", pmt.UserID).
				Int64("invoice_id", invoiceID).
				Msg("Subscription not found — was deleted. Creating new subscription")

			// Recover: create new subscription
			sub, err = s.recoverSubscription(pmt, metaPlanID, yooPayment)
			if err != nil {
				s.log.Error().Err(err).
					Int64("user_id", pmt.UserID).
					Int64("invoice_id", invoiceID).
					Msg("Failed to recover subscription")
				http.Error(w, "subscription recovery failed", http.StatusInternalServerError)
				return
			}
		} else {
			// Save payment_method_id for recurring payments
			if yooPayment.PaymentMethod != nil && yooPayment.PaymentMethod.Saved {
				sub.YooKassaPaymentMethodID = &yooPayment.PaymentMethod.ID
			}
		}

		s.activateSubscription(sub, pmt, "yookassa")
	} else {
		s.log.Error().
			Int64("invoice_id", invoiceID).
			Int64("user_id", pmt.UserID).
			Msg("Payment has no subscription_id — cannot activate subscription")
		http.Error(w, "no subscription linked", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// recoverPaymentFromWebhook recreates a payment record from YooKassa webhook metadata
// when the original was deleted by stale cleanup
func (s *Server) recoverPaymentFromWebhook(invoiceID, userID, subID, planID int64, yooPayment *payment.Payment) (*database.Payment, error) {
	// Parse amount from webhook
	var amount float64
	if yooPayment.Amount.Value != "" {
		_, _ = fmt.Sscanf(yooPayment.Amount.Value, "%f", &amount)
	}

	// Check if subscription still exists
	var subscriptionID *int64
	if subID > 0 {
		sub, _ := s.db.Subscriptions.GetByID(subID)
		if sub != nil {
			subscriptionID = &sub.ID
		}
	}

	// If no subscription, create one
	if subscriptionID == nil {
		sub := &database.Subscription{
			UserID: userID,
			PlanID: planID,
			Status: database.SubscriptionStatusPending,
		}
		if err := s.db.Subscriptions.Create(sub); err != nil {
			return nil, fmt.Errorf("create recovery subscription: %w", err)
		}
		subscriptionID = &sub.ID
		s.log.Info().
			Int64("subscription_id", sub.ID).
			Int64("user_id", userID).
			Int64("plan_id", planID).
			Msg("Recovery subscription created")
	}

	pmt := &database.Payment{
		UserID:         userID,
		SubscriptionID: subscriptionID,
		InvoiceID:      invoiceID,
		Amount:         amount,
		Status:         database.PaymentStatusPending,
		Provider:       "yookassa",
	}
	if err := s.db.Payments.Create(pmt); err != nil {
		return nil, fmt.Errorf("create recovery payment: %w", err)
	}

	s.log.Warn().
		Int64("invoice_id", invoiceID).
		Int64("user_id", userID).
		Int64("payment_id", pmt.ID).
		Msg("Payment record recovered from webhook metadata")

	return pmt, nil
}

// recoverSubscription creates a new subscription when the original was deleted
func (s *Server) recoverSubscription(pmt *database.Payment, planID int64, yooPayment *payment.Payment) (*database.Subscription, error) {
	if planID == 0 {
		return nil, fmt.Errorf("no plan_id available for recovery")
	}

	sub := &database.Subscription{
		UserID: pmt.UserID,
		PlanID: planID,
		Status: database.SubscriptionStatusPending,
	}

	// Save payment_method_id if available
	if yooPayment.PaymentMethod != nil && yooPayment.PaymentMethod.Saved {
		sub.YooKassaPaymentMethodID = &yooPayment.PaymentMethod.ID
	}

	if err := s.db.Subscriptions.Create(sub); err != nil {
		return nil, fmt.Errorf("create recovery subscription: %w", err)
	}

	// Link payment to new subscription
	pmt.SubscriptionID = &sub.ID
	if err := s.db.Payments.Update(pmt); err != nil {
		s.log.Error().Err(err).Msg("Failed to link payment to recovered subscription")
	}

	s.log.Warn().
		Int64("subscription_id", sub.ID).
		Int64("user_id", pmt.UserID).
		Int64("plan_id", planID).
		Int64("invoice_id", pmt.InvoiceID).
		Msg("Subscription recovered during webhook processing")

	return sub, nil
}

// handlePaymentCanceled processes canceled payment webhook
func (s *Server) handlePaymentCanceled(w http.ResponseWriter, yooPayment *payment.Payment) {
	metaUserID, _, _ := parseYooKassaMetadata(yooPayment.Metadata)

	// Get invoice_id from metadata
	invoiceIDStr, ok := yooPayment.Metadata["invoice_id"]
	if !ok {
		s.log.Warn().
			Str("payment_id", yooPayment.ID).
			Int64("meta_user_id", metaUserID).
			Msg("No invoice_id in canceled payment metadata")
		w.WriteHeader(http.StatusOK)
		return
	}

	var invoiceID int64
	if _, err := fmt.Sscanf(invoiceIDStr, "%d", &invoiceID); err != nil {
		s.log.Warn().Str("invoice_id_str", invoiceIDStr).Msg("Invalid invoice_id in canceled payment")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get payment record
	pmt, err := s.db.Payments.GetByInvoiceID(invoiceID)
	if err != nil || pmt == nil {
		s.log.Warn().
			Int64("invoice_id", invoiceID).
			Int64("meta_user_id", metaUserID).
			Msg("Canceled payment not found in DB (may have been cleaned up)")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Update payment status
	pmt.Status = database.PaymentStatusFailed

	// Save cancellation details
	yookassaData := map[string]interface{}{
		"yookassa_payment_id": yooPayment.ID,
		"status":              "canceled",
	}
	if yooPayment.CancellationDetails != nil {
		yookassaData["cancellation_party"] = yooPayment.CancellationDetails.Party
		yookassaData["cancellation_reason"] = yooPayment.CancellationDetails.Reason
	}
	data, _ := json.Marshal(yookassaData)
	pmt.YooKassaData = string(data)

	if err := s.db.Payments.Update(pmt); err != nil {
		s.log.Error().Err(err).Msg("Failed to update canceled payment")
	}

	s.log.Info().
		Int64("invoice_id", invoiceID).
		Int64("user_id", pmt.UserID).
		Str("yookassa_payment_id", yooPayment.ID).
		Msg("Payment canceled")

	w.WriteHeader(http.StatusOK)
}

// handlePaymentSuccess handles redirect after successful payment (GET)
func (s *Server) handlePaymentSuccess(w http.ResponseWriter, r *http.Request) {
	// Redirect to frontend success page
	http.Redirect(w, r, "/payment/success", http.StatusFound)
}

// handlePaymentFail handles redirect after failed payment (GET)
func (s *Server) handlePaymentFail(w http.ResponseWriter, r *http.Request) {
	// Redirect to frontend fail page
	http.Redirect(w, r, "/payment/fail", http.StatusFound)
}

// handleCancelSubscription cancels auto-renewal
func (s *Server) handleCancelSubscription(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sub, err := s.db.Subscriptions.GetByUserID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get subscription")
		s.respondError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}

	if sub == nil {
		s.respondError(w, http.StatusNotFound, "no active subscription")
		return
	}

	if sub.Status != database.SubscriptionStatusActive {
		s.respondError(w, http.StatusBadRequest, "subscription is not active")
		return
	}

	// Mark as cancelled (will expire at period end)
	sub.Status = database.SubscriptionStatusCancelled
	sub.Recurring = false
	// Clear payment method to prevent autopayments
	sub.YooKassaPaymentMethodID = nil

	// Cancel Creem subscription if applicable
	if sub.CreemSubscriptionID != nil && *sub.CreemSubscriptionID != "" {
		if provider, err := s.getPaymentProvider(r.Host); err == nil {
			if cancelErr := provider.CancelSubscription(*sub.CreemSubscriptionID); cancelErr != nil {
				s.log.Error().Err(cancelErr).Msg("Failed to cancel Creem subscription")
				s.respondError(w, http.StatusInternalServerError, "failed to cancel subscription")
				return
			}
		}
		sub.CreemSubscriptionID = nil
	}

	if err := s.db.Subscriptions.Update(sub); err != nil {
		s.log.Error().Err(err).Msg("Failed to cancel subscription")
		s.respondError(w, http.StatusInternalServerError, "failed to cancel subscription")
		return
	}

	// Log audit
	_ = s.db.Audit.Log(&user.ID, "subscription_cancelled", map[string]interface{}{
		"subscription_id": sub.ID,
		"expires_at":      sub.CurrentPeriodEnd,
	}, auth.GetClientIP(r))

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "subscription will not renew",
	})
}

// handleChangePlan schedules a plan change for next period
func (s *Server) handleChangePlan(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ChangePlanRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get new plan
	newPlan, err := s.db.Plans.GetByID(req.PlanID)
	if err != nil || newPlan == nil {
		s.respondError(w, http.StatusBadRequest, "invalid plan")
		return
	}

	if !newPlan.IsPublic && !user.IsAdmin {
		s.respondError(w, http.StatusForbidden, "plan not available")
		return
	}

	sub, err := s.db.Subscriptions.GetByUserID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get subscription")
		s.respondError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}

	if sub == nil || sub.Status != database.SubscriptionStatusActive {
		s.respondError(w, http.StatusBadRequest, "no active subscription")
		return
	}

	// Same plan - remove scheduled change
	if sub.PlanID == req.PlanID {
		sub.NextPlanID = nil
		if err := s.db.Subscriptions.Update(sub); err != nil {
			s.log.Error().Err(err).Msg("Failed to update subscription")
			s.respondError(w, http.StatusInternalServerError, "failed to update subscription")
			return
		}
		s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
			Success: true,
			Message: "plan change cancelled",
		})
		return
	}

	// Schedule plan change for next period
	sub.NextPlanID = &req.PlanID
	if err := s.db.Subscriptions.Update(sub); err != nil {
		s.log.Error().Err(err).Msg("Failed to schedule plan change")
		s.respondError(w, http.StatusInternalServerError, "failed to schedule plan change")
		return
	}

	// Log audit
	_ = s.db.Audit.Log(&user.ID, "plan_change_scheduled", map[string]interface{}{
		"subscription_id": sub.ID,
		"current_plan_id": sub.PlanID,
		"next_plan_id":    req.PlanID,
		"effective_at":    sub.CurrentPeriodEnd,
	}, auth.GetClientIP(r))

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "plan change scheduled for next billing period",
	})
}

// handleGetPayments returns user's payment history
func (s *Server) handleGetPayments(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	payments, total, err := s.db.Payments.GetByUserID(user.ID, 50, 0)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get payments")
		s.respondError(w, http.StatusInternalServerError, "failed to get payments")
		return
	}

	paymentDTOs := make([]*dto.PaymentDTO, len(payments))
	for i, p := range payments {
		paymentDTOs[i] = dto.PaymentFromModel(p)
	}

	s.respondJSON(w, http.StatusOK, dto.PaymentsListResponse{
		Payments: paymentDTOs,
		Total:    total,
	})
}

// handleCreemWebhook handles Creem webhook notifications
func (s *Server) handleCreemWebhook(w http.ResponseWriter, r *http.Request) {
	s.log.Info().Msg("Creem webhook received")

	if !s.cfg.Creem.Enabled {
		http.Error(w, "creem payments disabled", http.StatusServiceUnavailable)
		return
	}

	provider, err := s.paymentProviders.Get("creem")
	if err != nil {
		s.log.Error().Err(err).Msg("Creem provider not registered")
		http.Error(w, "creem not configured", http.StatusInternalServerError)
		return
	}

	events, err := provider.HandleWebhook(r)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to handle Creem webhook")
		http.Error(w, "webhook error", http.StatusBadRequest)
		return
	}

	for _, evt := range events {
		s.log.Info().
			Str("type", string(evt.Type)).
			Int64("invoice_id", evt.InvoiceID).
			Msg("Creem webhook event")

		switch evt.Type {
		case payment.WebhookEventPaymentSucceeded:
			s.handleCreemPaymentSucceeded(evt)
		case payment.WebhookEventSubscriptionRenewed:
			s.handleCreemSubscriptionRenewed(evt)
		case payment.WebhookEventPaymentFailed:
			s.handleCreemPaymentFailed(evt)
		case payment.WebhookEventSubscriptionDeleted:
			s.handleCreemSubscriptionDeleted(evt)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleCreemPaymentSucceeded handles payment succeeded event from Creem
func (s *Server) handleCreemPaymentSucceeded(evt payment.WebhookEvent) {
	if evt.InvoiceID == 0 {
		s.log.Warn().Str("payment_id", evt.ProviderPaymentID).Msg("No invoice_id in Creem event")
		return
	}

	pmt, err := s.db.Payments.GetByInvoiceID(evt.InvoiceID)
	if err != nil || pmt == nil {
		s.log.Error().Err(err).Int64("invoice_id", evt.InvoiceID).Msg("Payment not found for Creem event")
		return
	}

	if pmt.Status == database.PaymentStatusSuccess {
		s.log.Info().Int64("invoice_id", evt.InvoiceID).Msg("Payment already processed")
		return
	}

	// Update payment
	pmt.Status = database.PaymentStatusSuccess
	providerData, _ := json.Marshal(evt.ProviderData)
	pmt.ProviderData = string(providerData)
	if err := s.db.Payments.Update(pmt); err != nil {
		s.log.Error().Err(err).Msg("Failed to update payment")
		return
	}

	// Activate subscription and save Creem IDs
	if pmt.SubscriptionID != nil {
		sub, err := s.db.Subscriptions.GetByID(*pmt.SubscriptionID)
		if err == nil && sub != nil {
			if evt.ProviderCustomerID != "" {
				sub.CreemCustomerID = &evt.ProviderCustomerID
			}
			if evt.ProviderSubscriptionID != "" {
				sub.CreemSubscriptionID = &evt.ProviderSubscriptionID
			}
			s.activateSubscription(sub, pmt, "creem")
		}
	}
}

// handleCreemSubscriptionRenewed handles subscription renewed event from Creem
func (s *Server) handleCreemSubscriptionRenewed(evt payment.WebhookEvent) {
	if evt.ProviderSubscriptionID == "" {
		s.log.Warn().Msg("No subscription ID in Creem renewal event")
		return
	}

	// Find subscription by Creem subscription ID
	sub, err := s.db.Subscriptions.GetByCreemSubscriptionID(evt.ProviderSubscriptionID)
	if err != nil || sub == nil {
		s.log.Error().Err(err).Str("creem_sub_id", evt.ProviderSubscriptionID).Msg("Subscription not found for Creem renewal")
		return
	}

	// Extend subscription period
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0)
	sub.CurrentPeriodStart = &now
	sub.CurrentPeriodEnd = &periodEnd
	sub.Status = database.SubscriptionStatusActive

	if err := s.db.Subscriptions.Update(sub); err != nil {
		s.log.Error().Err(err).Msg("Failed to extend subscription")
		return
	}

	s.log.Info().
		Int64("subscription_id", sub.ID).
		Int64("user_id", sub.UserID).
		Str("creem_subscription_id", evt.ProviderSubscriptionID).
		Msg("Subscription renewed via Creem")

	_ = s.db.Audit.Log(&sub.UserID, "subscription_renewed", map[string]interface{}{
		"subscription_id":       sub.ID,
		"creem_subscription_id": evt.ProviderSubscriptionID,
		"plan_id":               sub.PlanID,
		"amount":                evt.Amount,
	}, "webhook")
}

// handleCreemPaymentFailed handles payment failed event from Creem
func (s *Server) handleCreemPaymentFailed(evt payment.WebhookEvent) {
	if evt.ProviderSubscriptionID == "" {
		return
	}

	sub, err := s.db.Subscriptions.GetByCreemSubscriptionID(evt.ProviderSubscriptionID)
	if err != nil || sub == nil {
		s.log.Warn().Str("creem_sub_id", evt.ProviderSubscriptionID).Msg("Subscription not found for failed payment")
		return
	}

	s.log.Warn().
		Int64("subscription_id", sub.ID).
		Int64("user_id", sub.UserID).
		Msg("Creem payment failed")

	_ = s.db.Audit.Log(&sub.UserID, "payment_failed", map[string]interface{}{
		"subscription_id":       sub.ID,
		"creem_subscription_id": evt.ProviderSubscriptionID,
		"provider":              "creem",
	}, "webhook")
}

// handleCreemSubscriptionDeleted handles subscription deleted event from Creem
func (s *Server) handleCreemSubscriptionDeleted(evt payment.WebhookEvent) {
	if evt.ProviderSubscriptionID == "" {
		return
	}

	sub, err := s.db.Subscriptions.GetByCreemSubscriptionID(evt.ProviderSubscriptionID)
	if err != nil || sub == nil {
		s.log.Warn().Str("creem_sub_id", evt.ProviderSubscriptionID).Msg("Subscription not found for deletion")
		return
	}

	sub.Status = database.SubscriptionStatusExpired
	sub.CreemSubscriptionID = nil
	if err := s.db.Subscriptions.Update(sub); err != nil {
		s.log.Error().Err(err).Msg("Failed to update subscription after Creem deletion")
		return
	}

	// Downgrade to free plan
	freePlan, _ := s.db.Plans.GetBySlug("free")
	if freePlan != nil {
		if user, err := s.db.Users.GetByID(sub.UserID); err == nil && user != nil {
			user.PlanID = freePlan.ID
			_ = s.db.Users.Update(user)
		}
	}

	s.log.Info().
		Int64("subscription_id", sub.ID).
		Int64("user_id", sub.UserID).
		Msg("Subscription deleted via Creem webhook")

	_ = s.db.Audit.Log(&sub.UserID, "subscription_expired", map[string]interface{}{
		"subscription_id":       sub.ID,
		"creem_subscription_id": evt.ProviderSubscriptionID,
		"reason":                "creem_subscription_deleted",
	}, "webhook")
}
