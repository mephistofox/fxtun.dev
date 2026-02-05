package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	// YooKassaAPIURL is the base URL for YooKassa API
	YooKassaAPIURL = "https://api.yookassa.ru/v3"
)

// YooKassa IP ranges for webhook verification
// https://yookassa.ru/developers/using-api/webhooks#ip
var yookassaCIDRs = []string{
	"185.71.76.0/27",
	"185.71.77.0/27",
	"77.75.153.0/25",
	"77.75.154.128/25",
}

var yookassaIPs = []string{
	"77.75.156.11",
	"77.75.156.35",
}

// YooKassaConfig holds YooKassa configuration
type YooKassaConfig struct {
	ShopID    string
	SecretKey string
	TestMode  bool
	ReturnURL string
}

// YooKassa handles YooKassa payment operations
type YooKassa struct {
	config YooKassaConfig
	client *http.Client
}

// NewYooKassa creates a new YooKassa instance
func NewYooKassa(config YooKassaConfig) *YooKassa {
	return &YooKassa{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Amount represents money amount
type Amount struct {
	Value    string `json:"value"`    // "100.00"
	Currency string `json:"currency"` // "RUB"
}

// Confirmation represents payment confirmation settings
type Confirmation struct {
	Type      string `json:"type"`                 // "redirect"
	ReturnURL string `json:"return_url,omitempty"` // URL to redirect after payment
}

// ConfirmationResponse represents confirmation in response
type ConfirmationResponse struct {
	Type            string `json:"type"`
	ConfirmationURL string `json:"confirmation_url,omitempty"`
}

// Recipient represents payment recipient
type Recipient struct {
	AccountID string `json:"account_id,omitempty"`
	GatewayID string `json:"gateway_id,omitempty"`
}

// PaymentMethod represents payment method in response
type PaymentMethod struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Saved bool   `json:"saved"`
	Title string `json:"title,omitempty"`
	Card  *struct {
		First6      string `json:"first6,omitempty"`
		Last4       string `json:"last4,omitempty"`
		ExpiryYear  string `json:"expiry_year,omitempty"`
		ExpiryMonth string `json:"expiry_month,omitempty"`
		CardType    string `json:"card_type,omitempty"`
	} `json:"card,omitempty"`
}

// CreatePaymentRequest represents payment creation request
type CreatePaymentRequest struct {
	Amount            Amount            `json:"amount"`
	Description       string            `json:"description,omitempty"`
	Confirmation      *Confirmation     `json:"confirmation,omitempty"`
	Capture           bool              `json:"capture"`                       // true = immediate capture
	SavePaymentMethod bool              `json:"save_payment_method,omitempty"` // for recurring payments
	PaymentMethodID   string            `json:"payment_method_id,omitempty"`   // for autopayment with saved method
	PaymentMethodData *PaymentMethodData `json:"payment_method_data,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	Receipt           *Receipt          `json:"receipt,omitempty"`
}

// PaymentMethodData for specifying payment method type
type PaymentMethodData struct {
	Type string `json:"type"` // "bank_card", "yoo_money", etc.
}

// Receipt for fiscal receipt (54-FZ compliance)
type Receipt struct {
	Customer *Customer     `json:"customer,omitempty"`
	Items    []ReceiptItem `json:"items"`
}

// Customer for receipt
type Customer struct {
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// ReceiptItem for receipt
type ReceiptItem struct {
	Description    string `json:"description"`
	Quantity       string `json:"quantity"`
	Amount         Amount `json:"amount"`
	VATCode        int    `json:"vat_code"`         // 1 = no VAT (for self-employed)
	PaymentSubject string `json:"payment_subject"`  // "service"
	PaymentMode    string `json:"payment_mode"`     // "full_payment"
}

// Payment represents payment object from API
type Payment struct {
	ID                string                `json:"id"`
	Status            string                `json:"status"` // pending, waiting_for_capture, succeeded, canceled
	Amount            Amount                `json:"amount"`
	IncomeAmount      *Amount               `json:"income_amount,omitempty"`
	Description       string                `json:"description,omitempty"`
	Recipient         *Recipient            `json:"recipient,omitempty"`
	PaymentMethod     *PaymentMethod        `json:"payment_method,omitempty"`
	Confirmation      *ConfirmationResponse `json:"confirmation,omitempty"`
	CapturedAt        string                `json:"captured_at,omitempty"`
	CreatedAt         string                `json:"created_at"`
	ExpiresAt         string                `json:"expires_at,omitempty"`
	Metadata          map[string]string     `json:"metadata,omitempty"`
	Paid              bool                  `json:"paid"`
	Refundable        bool                  `json:"refundable"`
	Test              bool                  `json:"test"`
	CancellationDetails *CancellationDetails `json:"cancellation_details,omitempty"`
}

// CancellationDetails for canceled payments
type CancellationDetails struct {
	Party  string `json:"party"`  // "yoo_money", "payment_network", "merchant"
	Reason string `json:"reason"` // "3d_secure_failed", "expired_on_confirmation", etc.
}

// WebhookEvent represents incoming webhook notification
type WebhookEvent struct {
	Type   string   `json:"type"`   // "notification"
	Event  string   `json:"event"`  // "payment.succeeded", "payment.canceled", etc.
	Object *Payment `json:"object"` // Payment data
}

// APIError represents error response from YooKassa
type APIError struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Parameter   string `json:"parameter,omitempty"`
}

func (e *APIError) Error() string {
	if e.Parameter != "" {
		return fmt.Sprintf("yookassa: %s - %s (parameter: %s)", e.Code, e.Description, e.Parameter)
	}
	return fmt.Sprintf("yookassa: %s - %s", e.Code, e.Description)
}

// CreatePayment creates a new payment
func (y *YooKassa) CreatePayment(req CreatePaymentRequest, idempotencyKey string) (*Payment, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", YooKassaAPIURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.SetBasicAuth(y.config.ShopID, y.config.SecretKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotence-Key", idempotencyKey)

	resp, err := y.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("yookassa error: status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var payment Payment
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &payment, nil
}

// GetPayment retrieves payment by ID
func (y *YooKassa) GetPayment(paymentID string) (*Payment, error) {
	httpReq, err := http.NewRequest("GET", YooKassaAPIURL+"/payments/"+paymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.SetBasicAuth(y.config.ShopID, y.config.SecretKey)

	resp, err := y.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("yookassa error: status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var payment Payment
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &payment, nil
}

// CapturePayment captures a payment that is waiting_for_capture
func (y *YooKassa) CapturePayment(paymentID string, amount *Amount, idempotencyKey string) (*Payment, error) {
	var body []byte
	var err error

	if amount != nil {
		body, err = json.Marshal(map[string]interface{}{"amount": amount})
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
	}

	httpReq, err := http.NewRequest("POST", YooKassaAPIURL+"/payments/"+paymentID+"/capture", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.SetBasicAuth(y.config.ShopID, y.config.SecretKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotence-Key", idempotencyKey)

	resp, err := y.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("yookassa error: status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var payment Payment
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &payment, nil
}

// CancelPayment cancels a payment
func (y *YooKassa) CancelPayment(paymentID string, idempotencyKey string) (*Payment, error) {
	httpReq, err := http.NewRequest("POST", YooKassaAPIURL+"/payments/"+paymentID+"/cancel", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.SetBasicAuth(y.config.ShopID, y.config.SecretKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotence-Key", idempotencyKey)

	resp, err := y.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("yookassa error: status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var payment Payment
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &payment, nil
}

// IsTestMode returns whether test mode is enabled
func (y *YooKassa) IsTestMode() bool {
	return y.config.TestMode
}

// GetReturnURL returns the configured return URL
func (y *YooKassa) GetReturnURL() string {
	return y.config.ReturnURL
}

// IsYooKassaIP checks if the given IP is from YooKassa
func IsYooKassaIP(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	// Check single IPs
	for _, allowed := range yookassaIPs {
		if host == allowed {
			return true
		}
	}

	// Check CIDR ranges
	for _, cidr := range yookassaCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// FormatAmount formats float64 amount to string with 2 decimal places
func FormatAmount(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

// ParseWebhookEvent parses webhook event from request body
func ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("unmarshal webhook: %w", err)
	}
	return &event, nil
}
