package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mephistofox/fxtun.dev/internal/api/dto"
	"github.com/mephistofox/fxtun.dev/internal/auth"
	"github.com/mephistofox/fxtun.dev/internal/database"
	"github.com/mephistofox/fxtun.dev/internal/exchange"
	"github.com/mephistofox/fxtun.dev/internal/payment"
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
		resp.HasActive = sub.Status == database.SubscriptionStatusActive
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

	// Default: enabled if YooKassa is enabled
	return s.cfg.YooKassa.Enabled, "payments are not enabled"
}

// handleCheckout creates a payment and returns YooKassa payment URL
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

	// Generate invoice ID
	invoiceID, err := s.db.Payments.GetNextInvoiceID()
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to generate invoice ID")
		s.respondError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	// Create subscription record (pending)
	sub := &database.Subscription{
		UserID:    user.ID,
		PlanID:    plan.ID,
		Status:    database.SubscriptionStatusPending,
		Recurring: req.Recurring,
	}
	if err := s.db.Subscriptions.Create(sub); err != nil {
		s.log.Error().Err(err).Msg("Failed to create subscription")
		s.respondError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}

	// Get user email from database
	dbUser, _ := s.db.Users.GetByID(user.ID)
	email := ""
	if dbUser != nil {
		email = dbUser.Email
	}

	// Convert USD to RUB
	priceRUB := exchange.ConvertUSDToRUB(plan.Price)

	// Create payment record
	pmt := &database.Payment{
		UserID:         user.ID,
		SubscriptionID: &sub.ID,
		InvoiceID:      invoiceID,
		Amount:         priceRUB,
		Status:         database.PaymentStatusPending,
		IsRecurring:    req.Recurring,
	}
	if err := s.db.Payments.Create(pmt); err != nil {
		s.log.Error().Err(err).Msg("Failed to create payment")
		s.respondError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	// Create YooKassa payment
	yookassa := payment.NewYooKassa(payment.YooKassaConfig{
		ShopID:    s.cfg.YooKassa.ShopID,
		SecretKey: s.cfg.YooKassa.SecretKey,
		TestMode:  s.cfg.YooKassa.TestMode,
		ReturnURL: s.cfg.YooKassa.ReturnURL,
	})

	// Generate idempotency key
	idempotencyKey := uuid.New().String()

	// Build payment request
	paymentReq := payment.CreatePaymentRequest{
		Amount: payment.Amount{
			Value:    payment.FormatAmount(priceRUB),
			Currency: "RUB",
		},
		Description: fmt.Sprintf("fxTunnel %s subscription", plan.Name),
		Capture:     true, // Immediate capture
		Confirmation: &payment.Confirmation{
			Type:      "redirect",
			ReturnURL: yookassa.GetReturnURL(),
		},
		SavePaymentMethod: req.Recurring, // Save for recurring if requested
		Metadata: map[string]string{
			"invoice_id":      fmt.Sprintf("%d", invoiceID),
			"user_id":         fmt.Sprintf("%d", user.ID),
			"subscription_id": fmt.Sprintf("%d", sub.ID),
			"plan_id":         fmt.Sprintf("%d", plan.ID),
		},
	}

	// Add receipt for 54-FZ compliance (self-employed = no VAT)
	if email != "" {
		paymentReq.Receipt = &payment.Receipt{
			Customer: &payment.Customer{
				Email: email,
			},
			Items: []payment.ReceiptItem{
				{
					Description:    fmt.Sprintf("Подписка fxTunnel %s (1 месяц)", plan.Name),
					Quantity:       "1",
					Amount:         paymentReq.Amount,
					VATCode:        1, // No VAT (self-employed)
					PaymentSubject: "service",
					PaymentMode:    "full_payment",
				},
			},
		}
	}

	// Create payment via YooKassa API
	yooPayment, err := yookassa.CreatePayment(paymentReq, idempotencyKey)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to create YooKassa payment")
		s.respondError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	// Save YooKassa payment ID
	yookassaData, _ := json.Marshal(map[string]interface{}{
		"yookassa_payment_id": yooPayment.ID,
		"idempotency_key":     idempotencyKey,
	})
	pmt.YooKassaData = string(yookassaData)
	if err := s.db.Payments.Update(pmt); err != nil {
		s.log.Error().Err(err).Msg("Failed to update payment with YooKassa ID")
	}

	// Get confirmation URL
	paymentURL := ""
	if yooPayment.Confirmation != nil {
		paymentURL = yooPayment.Confirmation.ConfirmationURL
	}

	if paymentURL == "" {
		s.log.Error().Str("payment_id", yooPayment.ID).Msg("No confirmation URL in YooKassa response")
		s.respondError(w, http.StatusInternalServerError, "failed to get payment URL")
		return
	}

	// Log audit
	_ = s.db.Audit.Log(&user.ID, "payment_initiated", map[string]interface{}{
		"invoice_id":          invoiceID,
		"yookassa_payment_id": yooPayment.ID,
		"plan_id":             plan.ID,
		"amount":              priceRUB,
		"recurring":           req.Recurring,
	}, auth.GetClientIP(r))

	s.respondJSON(w, http.StatusOK, dto.CheckoutResponse{
		PaymentURL: paymentURL,
		InvoiceID:  invoiceID,
	})
}

// handlePaymentWebhook handles YooKassa webhook notifications (POST)
func (s *Server) handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	s.log.Info().
		Str("remote_addr", r.RemoteAddr).
		Str("method", r.Method).
		Msg("YooKassa webhook received")

	if !s.cfg.YooKassa.Enabled {
		http.Error(w, "payments disabled", http.StatusServiceUnavailable)
		return
	}

	// Verify IP (skip in test mode)
	if !s.cfg.YooKassa.TestMode {
		if !payment.IsYooKassaIP(r.RemoteAddr) {
			s.log.Warn().Str("ip", r.RemoteAddr).Msg("Webhook from unauthorized IP")
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

// handlePaymentSucceeded processes successful payment webhook
func (s *Server) handlePaymentSucceeded(w http.ResponseWriter, yooPayment *payment.Payment) {
	// Get invoice_id from metadata
	invoiceIDStr, ok := yooPayment.Metadata["invoice_id"]
	if !ok {
		s.log.Error().Str("payment_id", yooPayment.ID).Msg("No invoice_id in payment metadata")
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
		s.log.Error().Err(err).Int64("invoice_id", invoiceID).Msg("Payment not found")
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}

	// Already processed
	if pmt.Status == database.PaymentStatusSuccess {
		s.log.Info().Int64("invoice_id", invoiceID).Msg("Payment already processed")
		w.WriteHeader(http.StatusOK)
		return
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
		s.log.Error().Err(err).Msg("Failed to update payment")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Activate subscription
	if pmt.SubscriptionID != nil {
		sub, err := s.db.Subscriptions.GetByID(*pmt.SubscriptionID)
		if err == nil && sub != nil {
			now := time.Now()
			periodEnd := now.AddDate(0, 1, 0) // +1 month
			sub.Status = database.SubscriptionStatusActive
			sub.CurrentPeriodStart = &now
			sub.CurrentPeriodEnd = &periodEnd

			// Save payment_method_id for recurring payments
			if yooPayment.PaymentMethod != nil && yooPayment.PaymentMethod.Saved {
				sub.YooKassaPaymentMethodID = &yooPayment.PaymentMethod.ID
			}

			if err := s.db.Subscriptions.Update(sub); err != nil {
				s.log.Error().Err(err).Msg("Failed to activate subscription")
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
				Str("yookassa_payment_id", yooPayment.ID).
				Msg("Subscription activated")

			// Log audit
			_ = s.db.Audit.Log(&sub.UserID, "subscription_activated", map[string]interface{}{
				"invoice_id":          invoiceID,
				"yookassa_payment_id": yooPayment.ID,
				"plan_id":             sub.PlanID,
				"subscription_id":     sub.ID,
			}, "webhook")

			// Send payment success email notification
			if s.notifier != nil {
				plan, _ := s.db.Plans.GetByID(sub.PlanID)
				planName := "Unknown"
				if plan != nil {
					planName = plan.Name
				}
				amount, _ := strconv.ParseFloat(yooPayment.Amount.Value, 64)
				if err := s.notifier.SendPaymentSuccessNotification(sub.UserID, planName, amount); err != nil {
					s.log.Error().Err(err).Int64("user_id", sub.UserID).Msg("Failed to send payment success email")
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handlePaymentCanceled processes canceled payment webhook
func (s *Server) handlePaymentCanceled(w http.ResponseWriter, yooPayment *payment.Payment) {
	// Get invoice_id from metadata
	invoiceIDStr, ok := yooPayment.Metadata["invoice_id"]
	if !ok {
		s.log.Warn().Str("payment_id", yooPayment.ID).Msg("No invoice_id in canceled payment metadata")
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
		s.log.Warn().Int64("invoice_id", invoiceID).Msg("Canceled payment not found")
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
