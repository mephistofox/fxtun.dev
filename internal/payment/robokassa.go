package payment

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const (
	// RobokassaPaymentURL is the URL for initiating payments
	RobokassaPaymentURL = "https://auth.robokassa.ru/Merchant/Index.aspx"
	// RobokassaRecurringURL is the URL for recurring payments
	RobokassaRecurringURL = "https://auth.robokassa.ru/Merchant/Recurring"
)

// Allowed Robokassa IPs for ResultURL callback verification
var robokassaIPs = []string{
	"185.59.216.65",
	"185.59.217.65",
}

// RobokassaConfig holds Robokassa configuration
type RobokassaConfig struct {
	MerchantLogin string
	Password1     string
	Password2     string
	TestPassword1 string
	TestPassword2 string
	TestMode      bool
	ResultURL     string
	SuccessURL    string
	FailURL       string
}

// Robokassa handles Robokassa payment operations
type Robokassa struct {
	config RobokassaConfig
}

// NewRobokassa creates a new Robokassa instance
func NewRobokassa(config RobokassaConfig) *Robokassa {
	return &Robokassa{config: config}
}

// getPassword1 returns the appropriate password1 based on test mode
func (r *Robokassa) getPassword1() string {
	if r.config.TestMode {
		return r.config.TestPassword1
	}
	return r.config.Password1
}

// getPassword2 returns the appropriate password2 based on test mode
func (r *Robokassa) getPassword2() string {
	if r.config.TestMode {
		return r.config.TestPassword2
	}
	return r.config.Password2
}

// GenerateSignature generates SHA512 signature for payment
func (r *Robokassa) GenerateSignature(parts ...string) string {
	data := strings.Join(parts, ":")
	hash := sha512.Sum512([]byte(data))
	return hex.EncodeToString(hash[:])
}

// PaymentParams holds parameters for creating a payment URL
type PaymentParams struct {
	InvoiceID   int64
	OutSum      float64
	Description string
	Email       string
	Recurring   bool
}

// GeneratePaymentURL generates URL for redirecting user to Robokassa payment page
func (r *Robokassa) GeneratePaymentURL(params PaymentParams) string {
	outSum := formatAmount(params.OutSum)
	invoiceID := strconv.FormatInt(params.InvoiceID, 10)

	// Signature: MerchantLogin:OutSum:InvId:Password1
	signature := r.GenerateSignature(
		r.config.MerchantLogin,
		outSum,
		invoiceID,
		r.getPassword1(),
	)

	values := url.Values{}
	values.Set("MerchantLogin", r.config.MerchantLogin)
	values.Set("OutSum", outSum)
	values.Set("InvId", invoiceID)
	values.Set("Description", params.Description)
	values.Set("SignatureValue", signature)

	if params.Email != "" {
		values.Set("Email", params.Email)
	}

	if params.Recurring {
		values.Set("Recurring", "true")
	}

	if r.config.TestMode {
		values.Set("IsTest", "1")
	}

	// Set culture to Russian
	values.Set("Culture", "ru")

	return RobokassaPaymentURL + "?" + values.Encode()
}

// RecurringPaymentParams holds parameters for recurring payment
type RecurringPaymentParams struct {
	InvoiceID         int64
	PreviousInvoiceID int64
	OutSum            float64
}

// GenerateRecurringPaymentURL generates URL for recurring payment API call
func (r *Robokassa) GenerateRecurringPaymentURL(params RecurringPaymentParams) (string, url.Values) {
	outSum := formatAmount(params.OutSum)
	invoiceID := strconv.FormatInt(params.InvoiceID, 10)

	// Signature: MerchantLogin:OutSum:InvId:Password1
	// Note: PreviousInvoiceID is NOT included in signature
	signature := r.GenerateSignature(
		r.config.MerchantLogin,
		outSum,
		invoiceID,
		r.getPassword1(),
	)

	values := url.Values{}
	values.Set("MerchantLogin", r.config.MerchantLogin)
	values.Set("OutSum", outSum)
	values.Set("InvId", invoiceID)
	values.Set("PreviousInvoiceID", strconv.FormatInt(params.PreviousInvoiceID, 10))
	values.Set("SignatureValue", signature)

	if r.config.TestMode {
		values.Set("IsTest", "1")
	}

	return RobokassaRecurringURL, values
}

// ResultParams holds parameters received on ResultURL callback
type ResultParams struct {
	OutSum         float64
	OutSumRaw      string // Raw string from callback for signature verification
	InvID          int64
	SignatureValue string
	PaymentMethod  string
	EMail          string
	IsTest         bool
}

// ParseResultParams parses ResultURL callback parameters
func ParseResultParams(values url.Values) (*ResultParams, error) {
	outSumRaw := values.Get("OutSum")
	outSum, err := strconv.ParseFloat(outSumRaw, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid OutSum: %w", err)
	}

	invID, err := strconv.ParseInt(values.Get("InvId"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid InvId: %w", err)
	}

	isTest := values.Get("IsTest") == "1"

	return &ResultParams{
		OutSum:         outSum,
		OutSumRaw:      outSumRaw,
		InvID:          invID,
		SignatureValue: values.Get("SignatureValue"),
		PaymentMethod:  values.Get("PaymentMethod"),
		EMail:          values.Get("EMail"),
		IsTest:         isTest,
	}, nil
}

// VerifyResultSignature verifies signature from ResultURL callback
func (r *Robokassa) VerifyResultSignature(params *ResultParams) bool {
	// Use raw OutSum string from callback (not reformatted)
	outSum := params.OutSumRaw
	invoiceID := strconv.FormatInt(params.InvID, 10)

	// Use password based on IsTest flag from callback, not server config
	password2 := r.config.Password2
	if params.IsTest {
		password2 = r.config.TestPassword2
	}

	// Signature: OutSum:InvId:Password2
	expected := r.GenerateSignature(outSum, invoiceID, password2)

	return strings.EqualFold(expected, params.SignatureValue)
}

// GetExpectedSignature returns the expected signature for debugging
func (r *Robokassa) GetExpectedSignature(params *ResultParams) string {
	outSum := params.OutSumRaw
	invoiceID := strconv.FormatInt(params.InvID, 10)
	password2 := r.config.Password2
	if params.IsTest {
		password2 = r.config.TestPassword2
	}
	return r.GenerateSignature(outSum, invoiceID, password2)
}

// SuccessParams holds parameters received on SuccessURL redirect
type SuccessParams struct {
	OutSum         float64
	InvID          int64
	SignatureValue string
	Culture        string
}

// ParseSuccessParams parses SuccessURL redirect parameters
func ParseSuccessParams(values url.Values) (*SuccessParams, error) {
	outSum, err := strconv.ParseFloat(values.Get("OutSum"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid OutSum: %w", err)
	}

	invID, err := strconv.ParseInt(values.Get("InvId"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid InvId: %w", err)
	}

	return &SuccessParams{
		OutSum:         outSum,
		InvID:          invID,
		SignatureValue: values.Get("SignatureValue"),
		Culture:        values.Get("Culture"),
	}, nil
}

// VerifySuccessSignature verifies signature from SuccessURL redirect
func (r *Robokassa) VerifySuccessSignature(params *SuccessParams) bool {
	outSum := formatAmount(params.OutSum)
	invoiceID := strconv.FormatInt(params.InvID, 10)

	// Signature: OutSum:InvId:Password1
	expected := r.GenerateSignature(outSum, invoiceID, r.getPassword1())

	return strings.EqualFold(expected, params.SignatureValue)
}

// IsRobokassaIP checks if the given IP is from Robokassa
func IsRobokassaIP(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	for _, allowed := range robokassaIPs {
		if host == allowed {
			return true
		}
	}
	return false
}

// IsTestMode returns whether test mode is enabled
func (r *Robokassa) IsTestMode() bool {
	return r.config.TestMode
}

// formatAmount formats amount as string with 2 decimal places
func formatAmount(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 2, 64)
}

// GenerateResultResponse generates the response for ResultURL callback
func GenerateResultResponse(invoiceID int64) string {
	return fmt.Sprintf("OK%d", invoiceID)
}
