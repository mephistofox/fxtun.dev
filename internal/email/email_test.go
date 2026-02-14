package email

import (
	"testing"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
)

func TestService_New(t *testing.T) {
	cfg := &config.SMTPSettings{
		Enabled:  true,
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "test@example.com",
		FromName: "Test",
	}
	log := zerolog.New(zerolog.NewTestWriter(t))

	s := New(cfg, log)
	if s == nil {
		t.Fatal("Expected service to be created")
	}
}

func TestService_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.SMTPSettings
		expected bool
	}{
		{
			name: "enabled with all settings",
			cfg: &config.SMTPSettings{
				Enabled: true,
				Host:    "smtp.example.com",
				From:    "test@example.com",
			},
			expected: true,
		},
		{
			name: "disabled",
			cfg: &config.SMTPSettings{
				Enabled: false,
				Host:    "smtp.example.com",
				From:    "test@example.com",
			},
			expected: false,
		},
		{
			name: "no host",
			cfg: &config.SMTPSettings{
				Enabled: true,
				Host:    "",
				From:    "test@example.com",
			},
			expected: false,
		},
		{
			name: "no from",
			cfg: &config.SMTPSettings{
				Enabled: true,
				Host:    "smtp.example.com",
				From:    "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := zerolog.New(zerolog.NewTestWriter(t))
			s := New(tt.cfg, log)
			if s.IsEnabled() != tt.expected {
				t.Errorf("Expected IsEnabled() = %v, got %v", tt.expected, s.IsEnabled())
			}
		})
	}
}

func TestRenderTemplate_SubscriptionExpiring(t *testing.T) {
	data := TemplateData{
		UserName:    "John",
		PlanName:    "Pro",
		DaysLeft:    3,
		ExpiresAt:   "15.02.2026",
		CheckoutURL: "https://example.com/checkout",
	}

	html, err := RenderTemplate(TemplateSubscriptionExpiring, data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if html == "" {
		t.Fatal("Expected non-empty HTML")
	}

	// Check that data is rendered
	if !contains(html, "John") {
		t.Error("Expected HTML to contain user name")
	}
	if !contains(html, "Pro") {
		t.Error("Expected HTML to contain plan name")
	}
	if !contains(html, "3") {
		t.Error("Expected HTML to contain days left")
	}
	if !contains(html, "15.02.2026") {
		t.Error("Expected HTML to contain expiration date")
	}
	if !contains(html, "https://example.com/checkout") {
		t.Error("Expected HTML to contain checkout URL")
	}
}

func TestRenderTemplate_SubscriptionExpired(t *testing.T) {
	data := TemplateData{
		UserName:    "Jane",
		PlanName:    "Business",
		CheckoutURL: "https://example.com/checkout",
	}

	html, err := RenderTemplate(TemplateSubscriptionExpired, data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "Jane") {
		t.Error("Expected HTML to contain user name")
	}
	if !contains(html, "Business") {
		t.Error("Expected HTML to contain plan name")
	}
	if !contains(html, "истекла") {
		t.Error("Expected HTML to contain 'истекла'")
	}
}

func TestRenderTemplate_SubscriptionRenewed(t *testing.T) {
	data := TemplateData{
		UserName:     "Bob",
		PlanName:     "Pro",
		Amount:       999.00,
		RenewalDate:  "15.03.2026",
		DashboardURL: "https://example.com/dashboard",
	}

	html, err := RenderTemplate(TemplateSubscriptionRenewed, data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "Bob") {
		t.Error("Expected HTML to contain user name")
	}
	if !contains(html, "продлена") {
		t.Error("Expected HTML to contain 'продлена'")
	}
}

func TestRenderTemplate_RenewFailed(t *testing.T) {
	data := TemplateData{
		UserName:     "Alice",
		PlanName:     "Pro",
		ErrorMessage: "Insufficient funds",
		CheckoutURL:  "https://example.com/checkout",
	}

	html, err := RenderTemplate(TemplateSubscriptionRenewFailed, data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "Insufficient funds") {
		t.Error("Expected HTML to contain error message")
	}
}

func TestRenderTemplate_PlanChanged(t *testing.T) {
	data := TemplateData{
		UserName:     "Charlie",
		NewPlanName:  "Business",
		DashboardURL: "https://example.com/dashboard",
	}

	html, err := RenderTemplate(TemplatePlanChanged, data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "Business") {
		t.Error("Expected HTML to contain new plan name")
	}
	if !contains(html, "изменён") {
		t.Error("Expected HTML to contain 'изменён'")
	}
}

func TestRenderTemplate_PaymentSuccess(t *testing.T) {
	data := TemplateData{
		UserName:     "Diana",
		PlanName:     "Pro",
		Amount:       500.00,
		DashboardURL: "https://example.com/dashboard",
	}

	html, err := RenderTemplate(TemplatePaymentSuccess, data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "Diana") {
		t.Error("Expected HTML to contain user name")
	}
	if !contains(html, "успешно") {
		t.Error("Expected HTML to contain 'успешно'")
	}
}

func TestRenderTemplate_NotFound(t *testing.T) {
	_, err := RenderTemplate("nonexistent_template", TemplateData{})
	if err == nil {
		t.Error("Expected error for nonexistent template")
	}
}

// ── English template tests ──

func TestRenderTemplate_SubscriptionExpiring_EN(t *testing.T) {
	data := TemplateData{
		UserName:    "John",
		PlanName:    "Pro",
		DaysLeft:    3,
		ExpiresAt:   "Feb 15, 2026",
		CheckoutURL: "https://fxtun.dev/checkout",
	}

	html, err := RenderTemplate(LocalizedTemplateName(TemplateSubscriptionExpiring, "en"), data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "John") {
		t.Error("Expected HTML to contain user name")
	}
	if !contains(html, "expiring soon") {
		t.Error("Expected HTML to contain 'expiring soon'")
	}
	if !contains(html, "3") {
		t.Error("Expected HTML to contain days left")
	}
	if !contains(html, "Feb 15, 2026") {
		t.Error("Expected HTML to contain expiration date")
	}
	if !contains(html, "fxtun.dev") {
		t.Error("Expected HTML to contain fxtun.dev URL")
	}
}

func TestRenderTemplate_SubscriptionExpired_EN(t *testing.T) {
	data := TemplateData{
		UserName:    "Jane",
		PlanName:    "Business",
		CheckoutURL: "https://fxtun.dev/checkout",
	}

	html, err := RenderTemplate(LocalizedTemplateName(TemplateSubscriptionExpired, "en"), data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "Jane") {
		t.Error("Expected HTML to contain user name")
	}
	if !contains(html, "expired") {
		t.Error("Expected HTML to contain 'expired'")
	}
	if !contains(html, "downgraded") {
		t.Error("Expected HTML to contain 'downgraded'")
	}
}

func TestRenderTemplate_PaymentSuccess_EN(t *testing.T) {
	data := TemplateData{
		UserName:        "Diana",
		PlanName:        "Pro",
		Amount:          10.00,
		FormattedAmount: "$10",
		DashboardURL:    "https://fxtun.dev/dashboard",
	}

	html, err := RenderTemplate(LocalizedTemplateName(TemplatePaymentSuccess, "en"), data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "Diana") {
		t.Error("Expected HTML to contain user name")
	}
	if !contains(html, "Payment successful") {
		t.Error("Expected HTML to contain 'Payment successful'")
	}
	if !contains(html, "$10") {
		t.Error("Expected HTML to contain '$10'")
	}
	if !contains(html, "fxtun.dev") {
		t.Error("Expected HTML to contain fxtun.dev URL")
	}
}

func TestRenderTemplate_SubscriptionRenewed_EN(t *testing.T) {
	data := TemplateData{
		UserName:        "Bob",
		PlanName:        "Pro",
		FormattedAmount: "$10",
		RenewalDate:     "Mar 15, 2026",
		DashboardURL:    "https://fxtun.dev/dashboard",
	}

	html, err := RenderTemplate(LocalizedTemplateName(TemplateSubscriptionRenewed, "en"), data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "renewed") {
		t.Error("Expected HTML to contain 'renewed'")
	}
	if !contains(html, "$10") {
		t.Error("Expected HTML to contain '$10'")
	}
}

func TestRenderTemplate_RenewFailed_EN(t *testing.T) {
	data := TemplateData{
		UserName:     "Alice",
		PlanName:     "Pro",
		ErrorMessage: "Card declined",
		CheckoutURL:  "https://fxtun.dev/checkout",
	}

	html, err := RenderTemplate(LocalizedTemplateName(TemplateSubscriptionRenewFailed, "en"), data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "renewal failed") {
		t.Error("Expected HTML to contain 'renewal failed'")
	}
	if !contains(html, "Card declined") {
		t.Error("Expected HTML to contain error message")
	}
}

func TestRenderTemplate_PlanChanged_EN(t *testing.T) {
	data := TemplateData{
		UserName:     "Charlie",
		NewPlanName:  "Business",
		DashboardURL: "https://fxtun.dev/dashboard",
	}

	html, err := RenderTemplate(LocalizedTemplateName(TemplatePlanChanged, "en"), data)
	if err != nil {
		t.Fatalf("RenderTemplate error: %v", err)
	}

	if !contains(html, "Plan changed") {
		t.Error("Expected HTML to contain 'Plan changed'")
	}
	if !contains(html, "Business") {
		t.Error("Expected HTML to contain new plan name")
	}
}

func TestLocalizedTemplateName(t *testing.T) {
	if LocalizedTemplateName("payment_success", "en") != "payment_success_en" {
		t.Error("Expected _en suffix for English")
	}
	if LocalizedTemplateName("payment_success", "ru") != "payment_success" {
		t.Error("Expected no suffix for Russian")
	}
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		amount   float64
		lang     string
		expected string
	}{
		{10, "en", "$10"},
		{10.50, "en", "$10.50"},
		{350, "ru", "350 ₽"},
		{999.99, "ru", "1000 ₽"},
	}
	for _, tt := range tests {
		got := formatAmount(tt.amount, tt.lang)
		if got != tt.expected {
			t.Errorf("formatAmount(%v, %q) = %q, want %q", tt.amount, tt.lang, got, tt.expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && substr != "" && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
