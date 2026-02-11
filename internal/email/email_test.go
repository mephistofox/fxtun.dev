package email

import (
	"testing"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/config"
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
