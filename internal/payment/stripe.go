package payment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	stripesubscription "github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"
)

// StripeConfig holds Stripe configuration
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	TestMode      bool
	SuccessURL    string
	CancelURL     string
}

// Stripe handles Stripe payment operations
type Stripe struct {
	config StripeConfig
}

// NewStripe creates a new Stripe instance
func NewStripe(config StripeConfig) *Stripe {
	stripe.Key = config.SecretKey
	return &Stripe{
		config: config,
	}
}

// Name returns the provider name
func (s *Stripe) Name() string {
	return "stripe"
}

// CreateCheckoutSession creates a Stripe Checkout Session
func (s *Stripe) CreateCheckoutSession(params CheckoutParams) (*CheckoutResult, error) {
	// Find or create customer
	customerID, err := s.findOrCreateCustomer(params.Email, params.UserID)
	if err != nil {
		return nil, fmt.Errorf("find/create customer: %w", err)
	}

	// Build line items with ad-hoc pricing
	lineItems := []*stripe.CheckoutSessionLineItemParams{
		{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(params.Currency),
				UnitAmount: stripe.Int64(int64(params.Amount * 100)), // cents
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(params.Description),
				},
				Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
					Interval: stripe.String("month"),
				},
			},
			Quantity: stripe.Int64(1),
		},
	}

	// Create checkout session
	sessionParams := &stripe.CheckoutSessionParams{
		Customer:   stripe.String(customerID),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems:  lineItems,
		SuccessURL: stripe.String(s.config.SuccessURL),
		CancelURL:  stripe.String(s.config.CancelURL),
	}

	sessionParams.AddMetadata("invoice_id", fmt.Sprintf("%d", params.InvoiceID))
	sessionParams.AddMetadata("user_id", fmt.Sprintf("%d", params.UserID))
	sessionParams.AddMetadata("subscription_id", fmt.Sprintf("%d", params.SubscriptionID))
	sessionParams.AddMetadata("plan_id", fmt.Sprintf("%d", params.PlanID))

	sess, err := session.New(sessionParams)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	return &CheckoutResult{
		PaymentURL:             sess.URL,
		ProviderPaymentID:      sess.ID,
		ProviderCustomerID:     customerID,
		ProviderSubscriptionID: "", // Set after checkout.session.completed
	}, nil
}

// findOrCreateCustomer finds existing Stripe customer by email or creates a new one
func (s *Stripe) findOrCreateCustomer(email string, userID int64) (string, error) {
	if email != "" {
		// Search for existing customer
		params := &stripe.CustomerSearchParams{
			SearchParams: stripe.SearchParams{
				Query: fmt.Sprintf("email:'%s'", email),
			},
		}
		iter := customer.Search(params)
		if iter.Next() {
			return iter.Customer().ID, nil
		}
	}

	// Create new customer
	customerParams := &stripe.CustomerParams{
		Email: stripe.String(email),
	}
	customerParams.AddMetadata("user_id", fmt.Sprintf("%d", userID))

	c, err := customer.New(customerParams)
	if err != nil {
		return "", fmt.Errorf("create customer: %w", err)
	}

	return c.ID, nil
}

// HandleWebhook parses and verifies a Stripe webhook request
func (s *Stripe) HandleWebhook(r *http.Request) ([]WebhookEvent, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read webhook body: %w", err)
	}

	// Verify webhook signature (ignore API version mismatch between Stripe account and stripe-go)
	event, err := webhook.ConstructEventWithOptions(body, r.Header.Get("Stripe-Signature"), s.config.WebhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, fmt.Errorf("verify webhook signature: %w", err)
	}

	var events []WebhookEvent

	switch event.Type {
	case "checkout.session.completed":
		evt, err := s.handleCheckoutCompleted(event.Data.Raw)
		if err != nil {
			return nil, err
		}
		events = append(events, *evt)

	case "invoice.payment_succeeded":
		evt, err := s.handleInvoicePaymentSucceeded(event.Data.Raw)
		if err != nil {
			return nil, err
		}
		if evt != nil {
			events = append(events, *evt)
		}

	case "invoice.payment_failed":
		evt, err := s.handleInvoicePaymentFailed(event.Data.Raw)
		if err != nil {
			return nil, err
		}
		if evt != nil {
			events = append(events, *evt)
		}

	case "customer.subscription.deleted":
		evt, err := s.handleSubscriptionDeleted(event.Data.Raw)
		if err != nil {
			return nil, err
		}
		events = append(events, *evt)
	}

	return events, nil
}

// handleCheckoutCompleted processes checkout.session.completed
func (s *Stripe) handleCheckoutCompleted(raw json.RawMessage) (*WebhookEvent, error) {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(raw, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal checkout session: %w", err)
	}

	evt := &WebhookEvent{
		Type:                   WebhookEventPaymentSucceeded,
		ProviderPaymentID:      sess.ID,
		ProviderCustomerID:     sess.Customer.ID,
		ProviderSubscriptionID: sess.Subscription.ID,
		ProviderData: map[string]interface{}{
			"stripe_session_id": sess.ID,
		},
	}

	// Parse metadata
	if sess.Metadata != nil {
		if v, ok := sess.Metadata["invoice_id"]; ok {
			_, _ = fmt.Sscanf(v, "%d", &evt.InvoiceID)
		}
		if v, ok := sess.Metadata["user_id"]; ok {
			_, _ = fmt.Sscanf(v, "%d", &evt.UserID)
		}
		if v, ok := sess.Metadata["subscription_id"]; ok {
			_, _ = fmt.Sscanf(v, "%d", &evt.SubscriptionID)
		}
		if v, ok := sess.Metadata["plan_id"]; ok {
			_, _ = fmt.Sscanf(v, "%d", &evt.PlanID)
		}
	}

	// Amount from the session
	evt.Amount = float64(sess.AmountTotal) / 100
	evt.Currency = string(sess.Currency)

	return evt, nil
}

// handleInvoicePaymentSucceeded processes invoice.payment_succeeded (renewals)
func (s *Stripe) handleInvoicePaymentSucceeded(raw json.RawMessage) (*WebhookEvent, error) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(raw, &invoice); err != nil {
		return nil, fmt.Errorf("unmarshal invoice: %w", err)
	}

	// Skip the first invoice (handled by checkout.session.completed)
	if invoice.BillingReason == stripe.InvoiceBillingReasonSubscriptionCreate {
		return nil, nil
	}

	evt := &WebhookEvent{
		Type:               WebhookEventSubscriptionRenewed,
		ProviderPaymentID:  invoice.ID,
		ProviderCustomerID: invoice.Customer.ID,
		Amount:             float64(invoice.AmountPaid) / 100,
		Currency:           string(invoice.Currency),
		ProviderData: map[string]interface{}{
			"stripe_invoice_id": invoice.ID,
			"billing_reason":    string(invoice.BillingReason),
		},
	}

	// Get subscription info from parent
	if invoice.Parent != nil && invoice.Parent.SubscriptionDetails != nil {
		if invoice.Parent.SubscriptionDetails.Subscription != nil {
			evt.ProviderSubscriptionID = invoice.Parent.SubscriptionDetails.Subscription.ID
		}
		if invoice.Parent.SubscriptionDetails.Metadata != nil {
			meta := invoice.Parent.SubscriptionDetails.Metadata
			if v, ok := meta["user_id"]; ok {
				_, _ = fmt.Sscanf(v, "%d", &evt.UserID)
			}
			if v, ok := meta["subscription_id"]; ok {
				_, _ = fmt.Sscanf(v, "%d", &evt.SubscriptionID)
			}
			if v, ok := meta["plan_id"]; ok {
				_, _ = fmt.Sscanf(v, "%d", &evt.PlanID)
			}
		}
	}

	return evt, nil
}

// handleInvoicePaymentFailed processes invoice.payment_failed
func (s *Stripe) handleInvoicePaymentFailed(raw json.RawMessage) (*WebhookEvent, error) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(raw, &invoice); err != nil {
		return nil, fmt.Errorf("unmarshal invoice: %w", err)
	}

	evt := &WebhookEvent{
		Type:              WebhookEventPaymentFailed,
		ProviderPaymentID: invoice.ID,
		ProviderData: map[string]interface{}{
			"stripe_invoice_id": invoice.ID,
		},
	}

	if invoice.Customer != nil {
		evt.ProviderCustomerID = invoice.Customer.ID
	}

	// Get subscription info from parent
	if invoice.Parent != nil && invoice.Parent.SubscriptionDetails != nil {
		if invoice.Parent.SubscriptionDetails.Subscription != nil {
			evt.ProviderSubscriptionID = invoice.Parent.SubscriptionDetails.Subscription.ID
		}
		if invoice.Parent.SubscriptionDetails.Metadata != nil {
			meta := invoice.Parent.SubscriptionDetails.Metadata
			if v, ok := meta["user_id"]; ok {
				_, _ = fmt.Sscanf(v, "%d", &evt.UserID)
			}
			if v, ok := meta["subscription_id"]; ok {
				_, _ = fmt.Sscanf(v, "%d", &evt.SubscriptionID)
			}
		}
	}

	return evt, nil
}

// handleSubscriptionDeleted processes customer.subscription.deleted
func (s *Stripe) handleSubscriptionDeleted(raw json.RawMessage) (*WebhookEvent, error) {
	var sub stripe.Subscription
	if err := json.Unmarshal(raw, &sub); err != nil {
		return nil, fmt.Errorf("unmarshal subscription: %w", err)
	}

	evt := &WebhookEvent{
		Type:                   WebhookEventSubscriptionDeleted,
		ProviderSubscriptionID: sub.ID,
		ProviderCustomerID:     sub.Customer.ID,
		ProviderData: map[string]interface{}{
			"stripe_subscription_id": sub.ID,
			"cancel_reason":          sub.CancellationDetails,
		},
	}

	if sub.Metadata != nil {
		if v, ok := sub.Metadata["user_id"]; ok {
			_, _ = fmt.Sscanf(v, "%d", &evt.UserID)
		}
		if v, ok := sub.Metadata["subscription_id"]; ok {
			_, _ = fmt.Sscanf(v, "%d", &evt.SubscriptionID)
		}
	}

	return evt, nil
}

// CancelSubscription cancels a Stripe subscription
func (s *Stripe) CancelSubscription(providerSubscriptionID string) error {
	if providerSubscriptionID == "" {
		return nil
	}

	_, err := stripesubscription.Cancel(providerSubscriptionID, nil)
	if err != nil {
		return fmt.Errorf("cancel stripe subscription: %w", err)
	}

	return nil
}
