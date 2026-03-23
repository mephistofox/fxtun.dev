package payment

import "net/http"

// CheckoutParams contains parameters for creating a checkout session
type CheckoutParams struct {
	ProductID      string // Creem product ID (from plans.creem_product_id)
	InvoiceID      int64
	UserID         int64
	SubscriptionID int64
	PlanID         int64
	PlanName       string
	Amount         float64
	Currency       string // "RUB" or "USD"
	Email          string
	Recurring      bool
	Description    string
}

// CheckoutResult contains the result of creating a checkout session
type CheckoutResult struct {
	PaymentURL             string            // URL to redirect user to
	ProviderPaymentID      string            // Provider-specific payment ID
	ProviderCustomerID     string            // Provider-specific customer ID
	ProviderSubscriptionID string            // Provider-specific subscription ID
	Metadata               map[string]string // Additional provider-specific data
}

// WebhookEventType represents the type of webhook event
type WebhookEventType string

const (
	WebhookEventPaymentSucceeded    WebhookEventType = "payment.succeeded"
	WebhookEventPaymentFailed       WebhookEventType = "payment.failed"
	WebhookEventSubscriptionRenewed WebhookEventType = "subscription.renewed"
	WebhookEventSubscriptionDeleted WebhookEventType = "subscription.deleted"
)

// WebhookEvent represents a parsed webhook event from any provider
type WebhookEvent struct {
	Type                   WebhookEventType
	InvoiceID              int64
	UserID                 int64
	SubscriptionID         int64
	PlanID                 int64
	ProviderPaymentID      string
	ProviderCustomerID     string
	ProviderSubscriptionID string
	Amount                 float64
	Currency               string
	PaymentMethodSaved     bool
	PaymentMethodID        string
	PaymentMethodTitle     string
	ProviderData           map[string]interface{}
}

// Provider defines the interface for payment providers
type Provider interface {
	// Name returns the provider name (e.g., "yookassa", "stripe")
	Name() string

	// CreateCheckoutSession creates a new checkout/payment session
	CreateCheckoutSession(params CheckoutParams) (*CheckoutResult, error)

	// HandleWebhook parses and verifies an incoming webhook request
	HandleWebhook(r *http.Request) ([]WebhookEvent, error)

	// CancelSubscription cancels a subscription by provider-specific ID
	CancelSubscription(providerSubscriptionID string) error
}
