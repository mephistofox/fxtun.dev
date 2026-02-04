package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
)

// Service handles email sending
type Service struct {
	cfg *config.SMTPSettings
	log zerolog.Logger
}

// New creates a new email service
func New(cfg *config.SMTPSettings, log zerolog.Logger) *Service {
	return &Service{
		cfg: cfg,
		log: log.With().Str("component", "email").Logger(),
	}
}

// IsEnabled returns true if email service is enabled
func (s *Service) IsEnabled() bool {
	return s.cfg.Enabled && s.cfg.Host != "" && s.cfg.From != ""
}

// Message represents an email message
type Message struct {
	To       string
	Subject  string
	Body     string
	HTMLBody string
}

// Send sends an email message
func (s *Service) Send(msg Message) error {
	if !s.IsEnabled() {
		s.log.Debug().Str("to", msg.To).Msg("Email service disabled, skipping send")
		return nil
	}

	from := s.cfg.From
	if s.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.From)
	}

	// Build email content
	var body strings.Builder
	body.WriteString(fmt.Sprintf("From: %s\r\n", from))
	body.WriteString(fmt.Sprintf("To: %s\r\n", msg.To))
	body.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))

	if msg.HTMLBody != "" {
		boundary := "----=_Part_0_1234567890.1234567890"
		body.WriteString("MIME-Version: 1.0\r\n")
		body.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		body.WriteString("\r\n")

		// Plain text part
		if msg.Body != "" {
			body.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			body.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			body.WriteString("\r\n")
			body.WriteString(msg.Body)
			body.WriteString("\r\n")
		}

		// HTML part
		body.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		body.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		body.WriteString("\r\n")
		body.WriteString(msg.HTMLBody)
		body.WriteString("\r\n")
		body.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		body.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		body.WriteString("\r\n")
		body.WriteString(msg.Body)
	}

	// Send email
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	var err error
	if s.cfg.Port == s.cfg.SSLPort || s.cfg.Port == 465 {
		// Use SSL/TLS directly
		err = s.sendTLS(addr, auth, s.cfg.From, msg.To, []byte(body.String()))
	} else {
		// Use STARTTLS
		err = s.sendStartTLS(addr, auth, s.cfg.From, msg.To, []byte(body.String()))
	}

	if err != nil {
		s.log.Error().Err(err).
			Str("to", msg.To).
			Str("subject", msg.Subject).
			Msg("Failed to send email")
		return fmt.Errorf("send email: %w", err)
	}

	s.log.Info().
		Str("to", msg.To).
		Str("subject", msg.Subject).
		Msg("Email sent successfully")

	return nil
}

// sendTLS sends email using direct TLS connection (port 465)
func (s *Service) sendTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: s.cfg.Host,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("new smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}

	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if err := wc.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return client.Quit()
}

// sendStartTLS sends email using STARTTLS (port 587)
func (s *Service) sendStartTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: s.cfg.Host,
			MinVersion: tls.VersionTLS12,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}

	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if err := wc.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return client.Quit()
}

// Template names
const (
	TemplateSubscriptionExpiring    = "subscription_expiring"
	TemplateSubscriptionExpired     = "subscription_expired"
	TemplateSubscriptionRenewed     = "subscription_renewed"
	TemplateSubscriptionRenewFailed = "subscription_renew_failed"
	TemplatePlanChanged             = "plan_changed"
	TemplatePaymentSuccess          = "payment_success"
	TemplatePaymentFailed           = "payment_failed"
)

// TemplateData holds data for email templates
type TemplateData struct {
	UserName      string
	UserEmail     string
	PlanName      string
	NewPlanName   string
	DaysLeft      int
	Amount        float64
	ExpiresAt     string
	RenewalDate   string
	DashboardURL  string
	CheckoutURL   string
	SupportEmail  string
	ErrorMessage  string
}

// templates holds email templates
var templates = map[string]*template.Template{}

func init() {
	// Subscription expiring template
	templates[TemplateSubscriptionExpiring] = template.Must(template.New("subscription_expiring").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4F46E5; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 20px; border: 1px solid #e5e7eb; }
        .button { display: inline-block; background: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin-top: 16px; }
        .footer { text-align: center; padding: 20px; color: #6b7280; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>fxTunnel</h1>
        </div>
        <div class="content">
            <h2>Подписка скоро истекает</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Ваша подписка на тариф <strong>{{.PlanName}}</strong> истекает через <strong>{{.DaysLeft}}</strong> {{if eq .DaysLeft 1}}день{{else if le .DaysLeft 4}}дня{{else}}дней{{end}}.</p>
            <p>Дата окончания: <strong>{{.ExpiresAt}}</strong></p>
            {{if .CheckoutURL}}
            <p>Чтобы продлить подписку, нажмите кнопку ниже:</p>
            <a href="{{.CheckoutURL}}" class="button">Продлить подписку</a>
            {{end}}
        </div>
        <div class="footer">
            <p>fxTunnel — Self-hosted reverse tunneling</p>
            {{if .SupportEmail}}<p>Поддержка: {{.SupportEmail}}</p>{{end}}
        </div>
    </div>
</body>
</html>
`))

	// Subscription expired template
	templates[TemplateSubscriptionExpired] = template.Must(template.New("subscription_expired").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #DC2626; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 20px; border: 1px solid #e5e7eb; }
        .button { display: inline-block; background: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin-top: 16px; }
        .footer { text-align: center; padding: 20px; color: #6b7280; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>fxTunnel</h1>
        </div>
        <div class="content">
            <h2>Подписка истекла</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Ваша подписка на тариф <strong>{{.PlanName}}</strong> истекла.</p>
            <p>Ваш аккаунт переведён на бесплатный тариф с ограниченными возможностями.</p>
            {{if .CheckoutURL}}
            <p>Чтобы восстановить доступ ко всем функциям, оформите новую подписку:</p>
            <a href="{{.CheckoutURL}}" class="button">Оформить подписку</a>
            {{end}}
        </div>
        <div class="footer">
            <p>fxTunnel — Self-hosted reverse tunneling</p>
            {{if .SupportEmail}}<p>Поддержка: {{.SupportEmail}}</p>{{end}}
        </div>
    </div>
</body>
</html>
`))

	// Subscription renewed template
	templates[TemplateSubscriptionRenewed] = template.Must(template.New("subscription_renewed").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #059669; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 20px; border: 1px solid #e5e7eb; }
        .button { display: inline-block; background: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin-top: 16px; }
        .footer { text-align: center; padding: 20px; color: #6b7280; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>fxTunnel</h1>
        </div>
        <div class="content">
            <h2>Подписка продлена</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Ваша подписка на тариф <strong>{{.PlanName}}</strong> успешно продлена.</p>
            <p>Сумма списания: <strong>{{.Amount}} ₽</strong></p>
            <p>Следующее продление: <strong>{{.RenewalDate}}</strong></p>
            {{if .DashboardURL}}
            <a href="{{.DashboardURL}}" class="button">Перейти в личный кабинет</a>
            {{end}}
        </div>
        <div class="footer">
            <p>fxTunnel — Self-hosted reverse tunneling</p>
            {{if .SupportEmail}}<p>Поддержка: {{.SupportEmail}}</p>{{end}}
        </div>
    </div>
</body>
</html>
`))

	// Subscription renewal failed template
	templates[TemplateSubscriptionRenewFailed] = template.Must(template.New("subscription_renew_failed").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #DC2626; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 20px; border: 1px solid #e5e7eb; }
        .button { display: inline-block; background: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin-top: 16px; }
        .footer { text-align: center; padding: 20px; color: #6b7280; font-size: 14px; }
        .error { background: #FEE2E2; border: 1px solid #FECACA; padding: 12px; border-radius: 6px; margin: 16px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>fxTunnel</h1>
        </div>
        <div class="content">
            <h2>Ошибка продления подписки</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Не удалось автоматически продлить вашу подписку на тариф <strong>{{.PlanName}}</strong>.</p>
            {{if .ErrorMessage}}
            <div class="error">
                <strong>Причина:</strong> {{.ErrorMessage}}
            </div>
            {{end}}
            <p>Пожалуйста, проверьте платёжные данные и попробуйте продлить подписку вручную:</p>
            {{if .CheckoutURL}}
            <a href="{{.CheckoutURL}}" class="button">Продлить подписку</a>
            {{end}}
        </div>
        <div class="footer">
            <p>fxTunnel — Self-hosted reverse tunneling</p>
            {{if .SupportEmail}}<p>Поддержка: {{.SupportEmail}}</p>{{end}}
        </div>
    </div>
</body>
</html>
`))

	// Plan changed template
	templates[TemplatePlanChanged] = template.Must(template.New("plan_changed").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4F46E5; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 20px; border: 1px solid #e5e7eb; }
        .button { display: inline-block; background: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin-top: 16px; }
        .footer { text-align: center; padding: 20px; color: #6b7280; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>fxTunnel</h1>
        </div>
        <div class="content">
            <h2>Тариф изменён</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Ваш тарифный план успешно изменён на <strong>{{.NewPlanName}}</strong>.</p>
            <p>Новые условия уже действуют.</p>
            {{if .DashboardURL}}
            <a href="{{.DashboardURL}}" class="button">Перейти в личный кабинет</a>
            {{end}}
        </div>
        <div class="footer">
            <p>fxTunnel — Self-hosted reverse tunneling</p>
            {{if .SupportEmail}}<p>Поддержка: {{.SupportEmail}}</p>{{end}}
        </div>
    </div>
</body>
</html>
`))

	// Payment success template
	templates[TemplatePaymentSuccess] = template.Must(template.New("payment_success").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #059669; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 20px; border: 1px solid #e5e7eb; }
        .button { display: inline-block; background: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin-top: 16px; }
        .footer { text-align: center; padding: 20px; color: #6b7280; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>fxTunnel</h1>
        </div>
        <div class="content">
            <h2>Оплата прошла успешно</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Благодарим за оплату!</p>
            <p>Тариф: <strong>{{.PlanName}}</strong></p>
            <p>Сумма: <strong>{{.Amount}} ₽</strong></p>
            {{if .DashboardURL}}
            <a href="{{.DashboardURL}}" class="button">Перейти в личный кабинет</a>
            {{end}}
        </div>
        <div class="footer">
            <p>fxTunnel — Self-hosted reverse tunneling</p>
            {{if .SupportEmail}}<p>Поддержка: {{.SupportEmail}}</p>{{end}}
        </div>
    </div>
</body>
</html>
`))
}

// RenderTemplate renders an email template with data
func RenderTemplate(name string, data TemplateData) (string, error) {
	tmpl, ok := templates[name]
	if !ok {
		return "", fmt.Errorf("template not found: %s", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// SendTemplate sends an email using a template
func (s *Service) SendTemplate(to, subject, templateName string, data TemplateData) error {
	html, err := RenderTemplate(templateName, data)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	return s.Send(Message{
		To:       to,
		Subject:  subject,
		HTMLBody: html,
	})
}
