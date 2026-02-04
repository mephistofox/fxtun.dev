package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/database"
	"github.com/mephistofox/fxtunnel/internal/exchange"
	"github.com/mephistofox/fxtunnel/internal/payment"
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

// handleCheckout creates a payment and returns Robokassa URL
func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if !s.cfg.Robokassa.Enabled {
		s.respondError(w, http.StatusServiceUnavailable, "payments are not enabled")
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
		// If user already has active subscription, they should use change endpoint
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

	// Generate Robokassa URL
	robokassa := payment.NewRobokassa(payment.RobokassaConfig{
		MerchantLogin: s.cfg.Robokassa.MerchantLogin,
		Password1:     s.cfg.Robokassa.Password1,
		Password2:     s.cfg.Robokassa.Password2,
		TestPassword1: s.cfg.Robokassa.TestPassword1,
		TestPassword2: s.cfg.Robokassa.TestPassword2,
		TestMode:      s.cfg.Robokassa.TestMode,
	})

	// Get user email from database
	dbUser, _ := s.db.Users.GetByID(user.ID)
	email := ""
	if dbUser != nil {
		email = dbUser.Email
	}

	// Convert USD to RUB for Robokassa
	priceRUB := exchange.ConvertUSDToRUB(plan.Price)

	// Create payment record with RUB amount (same as sent to Robokassa)
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

	paymentURL := robokassa.GeneratePaymentURL(payment.PaymentParams{
		InvoiceID:   invoiceID,
		OutSum:      priceRUB,
		Description: "fxTunnel " + plan.Name + " subscription",
		Email:       email,
		Recurring:   req.Recurring,
	})

	// Log audit
	_ = s.db.Audit.Log(&user.ID, "payment_initiated", map[string]interface{}{
		"invoice_id": invoiceID,
		"plan_id":    plan.ID,
		"amount":     plan.Price,
		"recurring":  req.Recurring,
	}, auth.GetClientIP(r))

	s.respondJSON(w, http.StatusOK, dto.CheckoutResponse{
		PaymentURL: paymentURL,
		InvoiceID:  invoiceID,
	})
}

// handlePaymentResult handles Robokassa ResultURL callback (POST)
func (s *Server) handlePaymentResult(w http.ResponseWriter, r *http.Request) {
	s.log.Info().
		Str("remote_addr", r.RemoteAddr).
		Str("method", r.Method).
		Msg("Payment result callback received")

	if !s.cfg.Robokassa.Enabled {
		http.Error(w, "payments disabled", http.StatusServiceUnavailable)
		return
	}

	// Verify IP (skip in test mode)
	robokassa := payment.NewRobokassa(payment.RobokassaConfig{
		MerchantLogin: s.cfg.Robokassa.MerchantLogin,
		Password1:     s.cfg.Robokassa.Password1,
		Password2:     s.cfg.Robokassa.Password2,
		TestPassword1: s.cfg.Robokassa.TestPassword1,
		TestPassword2: s.cfg.Robokassa.TestPassword2,
		TestMode:      s.cfg.Robokassa.TestMode,
	})

	if !robokassa.IsTestMode() {
		if !payment.IsRobokassaIP(r.RemoteAddr) {
			s.log.Warn().Str("ip", r.RemoteAddr).Msg("Payment result from unauthorized IP")
			http.Error(w, "unauthorized", http.StatusForbidden)
			return
		}
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.log.Error().Err(err).Msg("Failed to parse payment result form")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	params, err := payment.ParseResultParams(r.Form)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to parse payment result params")
		http.Error(w, "invalid params", http.StatusBadRequest)
		return
	}

	s.log.Info().
		Int64("invoice_id", params.InvID).
		Float64("out_sum", params.OutSum).
		Str("signature", params.SignatureValue[:16]+"...").
		Msg("Payment result params parsed")

	// Verify signature
	if !robokassa.VerifyResultSignature(params) {
		s.log.Warn().Int64("invoice_id", params.InvID).Msg("Invalid payment signature")
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	// Get payment record
	pmt, err := s.db.Payments.GetByInvoiceID(params.InvID)
	if err != nil || pmt == nil {
		s.log.Error().Err(err).Int64("invoice_id", params.InvID).Msg("Payment not found")
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}

	// Already processed
	if pmt.Status == database.PaymentStatusSuccess {
		_, _ = w.Write([]byte(payment.GenerateResultResponse(params.InvID)))
		return
	}

	// Verify amount
	if pmt.Amount != params.OutSum {
		s.log.Warn().
			Int64("invoice_id", params.InvID).
			Float64("expected", pmt.Amount).
			Float64("received", params.OutSum).
			Msg("Amount mismatch")
		http.Error(w, "amount mismatch", http.StatusBadRequest)
		return
	}

	// Update payment status
	pmt.Status = database.PaymentStatusSuccess
	robokassaData, _ := json.Marshal(map[string]interface{}{
		"payment_method": params.PaymentMethod,
		"email":          params.EMail,
	})
	pmt.RobokassaData = string(robokassaData)
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
			sub.RobokassaInvoiceID = &params.InvID

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
				Int64("invoice_id", params.InvID).
				Msg("Subscription activated")

			// Log audit
			_ = s.db.Audit.Log(&sub.UserID, "subscription_activated", map[string]interface{}{
				"invoice_id":      params.InvID,
				"plan_id":         sub.PlanID,
				"subscription_id": sub.ID,
			}, r.RemoteAddr)
		}
	}

	// Respond with OK{InvoiceID}
	_, _ = w.Write([]byte(payment.GenerateResultResponse(params.InvID)))
}

// handlePaymentSuccess handles Robokassa SuccessURL redirect (GET)
func (s *Server) handlePaymentSuccess(w http.ResponseWriter, r *http.Request) {
	// Redirect to frontend success page
	http.Redirect(w, r, "/payment/success", http.StatusFound)
}

// handlePaymentFail handles Robokassa FailURL redirect (GET)
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
