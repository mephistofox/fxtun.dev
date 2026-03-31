package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	creemProdURL = "https://api.creem.io/v1"
	creemTestURL = "https://test-api.creem.io/v1"
)

// CreemConfig holds Creem.io configuration
type CreemConfig struct {
	APIKey        string
	WebhookSecret string
	TestMode      bool
	SuccessURL    string
	CancelURL     string
}

// Creem handles Creem.io payment operations
type Creem struct {
	config CreemConfig
	client *http.Client
}

// NewCreem creates a new Creem instance
func NewCreem(config CreemConfig) *Creem {
	return &Creem{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name
func (c *Creem) Name() string {
	return "creem"
}

// baseURL returns the API base URL based on test mode
func (c *Creem) baseURL() string {
	if c.config.TestMode {
		return creemTestURL
	}
	return creemProdURL
}

// creemCheckoutRequest represents the request body for creating a checkout
type creemCheckoutRequest struct {
	ProductID  string            `json:"product_id"`
	SuccessURL string            `json:"success_url"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// creemCheckoutResponse represents the response from creating a checkout
type creemCheckoutResponse struct {
	ID             string `json:"id"`
	CheckoutURL    string `json:"checkout_url"`
	CustomerID     string `json:"customer_id,omitempty"`
	SubscriptionID string `json:"subscription_id,omitempty"`
}

// creemWebhookPayload represents the incoming webhook payload from Creem
type creemWebhookPayload struct {
	EventType string          `json:"eventType"`
	Object    json.RawMessage `json:"object"`
}

// creemWebhookObject represents the object inside a webhook payload
type creemWebhookObject struct {
	ID             string            `json:"id"`
	CustomerID     string            `json:"customer_id,omitempty"`
	SubscriptionID string            `json:"subscription_id,omitempty"`
	Amount         float64           `json:"amount,omitempty"`
	Currency       string            `json:"currency,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// creemErrorResponse represents an error response from Creem API
type creemErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// CreateCheckoutSession creates a Creem checkout session
func (c *Creem) CreateCheckoutSession(params CheckoutParams) (*CheckoutResult, error) {
	req := creemCheckoutRequest{
		ProductID:  params.ProductID,
		SuccessURL: c.config.SuccessURL,
		Metadata: map[string]string{
			"invoice_id":      fmt.Sprintf("%d", params.InvoiceID),
			"user_id":         fmt.Sprintf("%d", params.UserID),
			"subscription_id": fmt.Sprintf("%d", params.SubscriptionID),
			"plan_id":         fmt.Sprintf("%d", params.PlanID),
		},
	}

	var resp creemCheckoutResponse
	if err := c.doRequest("POST", "/checkouts", req, &resp); err != nil {
		return nil, fmt.Errorf("create checkout: %w", err)
	}

	return &CheckoutResult{
		PaymentURL:             resp.CheckoutURL,
		ProviderPaymentID:      resp.ID,
		ProviderCustomerID:     resp.CustomerID,
		ProviderSubscriptionID: resp.SubscriptionID,
	}, nil
}

// HandleWebhook parses and verifies an incoming Creem webhook request
func (c *Creem) HandleWebhook(r *http.Request) ([]WebhookEvent, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read webhook body: %w", err)
	}

	// Verify signature
	signature := r.Header.Get("creem-signature")
	if err := c.verifySignature(body, signature); err != nil {
		return nil, fmt.Errorf("verify webhook signature: %w", err)
	}

	// Parse payload
	var payload creemWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse webhook payload: %w", err)
	}

	// Parse object
	var obj creemWebhookObject
	if err := json.Unmarshal(payload.Object, &obj); err != nil {
		return nil, fmt.Errorf("parse webhook object: %w", err)
	}

	// Build base event from metadata
	event := WebhookEvent{
		ProviderPaymentID:      obj.ID,
		ProviderCustomerID:     obj.CustomerID,
		ProviderSubscriptionID: obj.SubscriptionID,
		Amount:                 obj.Amount,
		Currency:               obj.Currency,
		ProviderData: map[string]interface{}{
			"event_type": payload.EventType,
		},
	}

	parseMetadata(&event, obj.Metadata)

	// Map Creem event type to generic event type
	switch payload.EventType {
	case "checkout.completed":
		event.Type = WebhookEventPaymentSucceeded
	case "subscription.paid":
		event.Type = WebhookEventSubscriptionRenewed
	case "subscription.canceled", "subscription.expired":
		event.Type = WebhookEventSubscriptionDeleted
	case "subscription.scheduled_cancel":
		event.Type = WebhookEventSubscriptionDeleted
	case "subscription.past_due":
		event.Type = WebhookEventPaymentFailed
	default:
		event.Type = WebhookEventType(payload.EventType)
	}

	return []WebhookEvent{event}, nil
}

// CancelSubscription cancels a Creem subscription
func (c *Creem) CancelSubscription(providerSubscriptionID string) error {
	if providerSubscriptionID == "" {
		return nil
	}

	path := fmt.Sprintf("/subscriptions/%s/cancel", providerSubscriptionID)
	if err := c.doRequest("POST", path, nil, nil); err != nil {
		return fmt.Errorf("cancel creem subscription: %w", err)
	}

	return nil
}

// verifySignature verifies the HMAC-SHA256 webhook signature
func (c *Creem) verifySignature(body []byte, signature string) error {
	if c.config.WebhookSecret == "" {
		return fmt.Errorf("webhook secret not configured")
	}
	if signature == "" {
		return fmt.Errorf("missing creem-signature header")
	}

	mac := hmac.New(sha256.New, []byte(c.config.WebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// parseMetadata extracts standard metadata fields into the webhook event
func parseMetadata(event *WebhookEvent, metadata map[string]string) {
	if metadata == nil {
		return
	}
	if v, ok := metadata["invoice_id"]; ok {
		_, _ = fmt.Sscanf(v, "%d", &event.InvoiceID)
	}
	if v, ok := metadata["user_id"]; ok {
		_, _ = fmt.Sscanf(v, "%d", &event.UserID)
	}
	if v, ok := metadata["subscription_id"]; ok {
		_, _ = fmt.Sscanf(v, "%d", &event.SubscriptionID)
	}
	if v, ok := metadata["plan_id"]; ok {
		_, _ = fmt.Sscanf(v, "%d", &event.PlanID)
	}
}

// doRequest performs an HTTP request to the Creem API
func (c *Creem) doRequest(method, path string, reqBody interface{}, respBody interface{}) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequest(method, c.baseURL()+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr creemErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return fmt.Errorf("creem error: status %d", resp.StatusCode)
		}
		return fmt.Errorf("creem: %s - %s", apiErr.Error, apiErr.Message)
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
