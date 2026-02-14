package email

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
)

// loginAuth implements smtp.Auth for LOGIN mechanism (required by some providers like Beget)
type loginAuth struct {
	username, password string
}

func newLoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unknown server response")
		}
	}
	return nil, nil
}

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

	// Use LOGIN auth (works with more providers including Beget)
	auth := newLoginAuth(s.cfg.Username, s.cfg.Password)

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
	UserName        string
	UserEmail       string
	PlanName        string
	NewPlanName     string
	DaysLeft        int
	Amount          float64
	FormattedAmount string // Pre-formatted amount with currency (e.g. "350 ₽" or "$10")
	ExpiresAt       string
	RenewalDate     string
	DashboardURL    string
	CheckoutURL     string
	SupportEmail    string
	ErrorMessage    string
}

// LocalizedTemplateName returns the template name for the given language.
// For "en" it appends "_en" suffix, otherwise returns the base name (Russian).
func LocalizedTemplateName(base, lang string) string {
	if lang == "en" {
		return base + "_en"
	}
	return base
}

// templates holds email templates
var templates = map[string]*template.Template{}

// Email styles matching the landing page design system (Cyber-Industrial Noir)
const emailStyles = `
        @import url('https://fonts.googleapis.com/css2?family=Unbounded:wght@400;700;800&family=Onest:wght@400;600&display=swap');
        body { font-family: 'Onest', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #f2f2f2; background: #090a10; margin: 0; padding: 24px; -webkit-font-smoothing: antialiased; }
        .container { max-width: 600px; margin: 0 auto; border-radius: 18px; overflow: hidden; border: 1px solid #1f2330; background: #111319; }
        .accent-bar { height: 3px; background: linear-gradient(90deg, #80ff00, #b84dff, #80ff00); }
        .header { padding: 32px 32px 24px; text-align: center; }
        .logo { font-family: 'Unbounded', sans-serif; font-size: 26px; font-weight: 800; color: #80ff00; letter-spacing: -0.02em; margin: 0; line-height: 1; }
        .logo span { color: #f2f2f2; }
        .divider { height: 1px; background: linear-gradient(90deg, transparent, #1f2330 20%, rgba(128,255,0,0.3) 50%, #1f2330 80%, transparent); margin: 0; }
        .content { padding: 28px 32px 32px; }
        .content h2 { font-family: 'Unbounded', sans-serif; font-size: 20px; font-weight: 700; color: #f2f2f2; margin: 0 0 20px; letter-spacing: -0.01em; line-height: 1.3; }
        .content p { margin: 0 0 14px; color: #7f8694; font-size: 15px; line-height: 1.7; }
        .content strong { color: #f2f2f2; font-weight: 600; }
        .status-dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 10px; vertical-align: middle; }
        .dot-success { background: #80ff00; box-shadow: 0 0 8px rgba(128,255,0,0.5); }
        .dot-warning { background: #f0ad4e; box-shadow: 0 0 8px rgba(240,173,78,0.5); }
        .dot-error { background: #ff6b6b; box-shadow: 0 0 8px rgba(255,107,107,0.5); }
        .info-block { margin: 24px 0; background: #090a10; border: 1px solid #1f2330; border-radius: 12px; overflow: hidden; }
        .info-row { padding: 14px 20px; border-bottom: 1px solid #1f2330; }
        .info-row:last-child { border-bottom: none; }
        .info-label { color: #7f8694; font-size: 14px; }
        .info-value { color: #f2f2f2; font-weight: 600; font-size: 14px; float: right; }
        .button { display: inline-block; background: linear-gradient(135deg, #80ff00, #5c8a18); color: #090a10; padding: 14px 28px; text-decoration: none; border-radius: 10px; margin-top: 24px; font-weight: 700; font-size: 14px; letter-spacing: -0.01em; }
        .error-box { background: rgba(255,107,107,0.08); border: 1px solid rgba(255,107,107,0.2); padding: 14px 18px; border-radius: 10px; margin: 18px 0; color: #ff6b6b; font-size: 14px; line-height: 1.6; }
        .footer { padding: 20px 32px; text-align: center; }
        .footer p { margin: 0 0 6px; color: #4a4f5c; font-size: 12px; }
        .footer a { color: #80ff00; text-decoration: none; font-weight: 500; }
`

// emailHead is the shared HTML head with brand styles
const emailHead = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="color-scheme" content="dark">
    <meta name="supported-color-schemes" content="dark">
    <style>` + emailStyles + `</style>
</head>
<body>
    <div class="container">
        <div class="accent-bar"></div>
        <div class="header">
            <p class="logo">fx<span>Tunnel</span></p>
        </div>
        <div class="divider"></div>
        <div class="content">`

// emailFooterRU is the shared Russian footer
const emailFooterRU = `
        </div>
        <div class="divider"></div>
        <div class="footer">
            <p style="color:#7f8694;">fxTunnel — Reverse tunneling service</p>
            {{if .SupportEmail}}<p>Поддержка: <a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>{{end}}
        </div>
    </div>
</body>
</html>`

// emailFooterEN is the shared English footer
const emailFooterEN = `
        </div>
        <div class="divider"></div>
        <div class="footer">
            <p style="color:#7f8694;">fxTunnel — Reverse tunneling service</p>
            {{if .SupportEmail}}<p>Support: <a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>{{end}}
        </div>
    </div>
</body>
</html>`

func init() {
	// ── Russian templates ──────────────────────────────────────────────

	templates[TemplateSubscriptionExpiring] = template.Must(template.New("subscription_expiring").Parse(emailHead + `
            <h2><span class="status-dot dot-warning"></span>Подписка скоро истекает</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Ваша подписка на тариф <strong>{{.PlanName}}</strong> истекает через <strong>{{.DaysLeft}}</strong> {{if eq .DaysLeft 1}}день{{else if le .DaysLeft 4}}дня{{else}}дней{{end}}.</p>
            <p>Дата окончания: <strong>{{.ExpiresAt}}</strong></p>
            {{if .CheckoutURL}}<a href="{{.CheckoutURL}}" class="button">Продлить подписку</a>{{end}}` + emailFooterRU))

	templates[TemplateSubscriptionExpired] = template.Must(template.New("subscription_expired").Parse(emailHead + `
            <h2><span class="status-dot dot-error"></span>Подписка истекла</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Ваша подписка на тариф <strong>{{.PlanName}}</strong> истекла.</p>
            <p>Ваш аккаунт переведён на бесплатный тариф с ограниченными возможностями.</p>
            {{if .CheckoutURL}}<a href="{{.CheckoutURL}}" class="button">Оформить подписку</a>{{end}}` + emailFooterRU))

	templates[TemplateSubscriptionRenewed] = template.Must(template.New("subscription_renewed").Parse(emailHead + `
            <h2><span class="status-dot dot-success"></span>Подписка продлена</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Ваша подписка на тариф <strong>{{.PlanName}}</strong> успешно продлена.</p>
            <div class="info-block">
                <div class="info-row">
                    <span class="info-label">Сумма</span>
                    <span class="info-value">{{.FormattedAmount}}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">Следующее продление</span>
                    <span class="info-value">{{.RenewalDate}}</span>
                </div>
            </div>
            {{if .DashboardURL}}<a href="{{.DashboardURL}}" class="button">Перейти в личный кабинет</a>{{end}}` + emailFooterRU))

	templates[TemplateSubscriptionRenewFailed] = template.Must(template.New("subscription_renew_failed").Parse(emailHead + `
            <h2><span class="status-dot dot-error"></span>Ошибка продления подписки</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Не удалось автоматически продлить вашу подписку на тариф <strong>{{.PlanName}}</strong>.</p>
            {{if .ErrorMessage}}<div class="error-box"><strong>Причина:</strong> {{.ErrorMessage}}</div>{{end}}
            <p>Пожалуйста, проверьте платёжные данные и попробуйте продлить подписку вручную:</p>
            {{if .CheckoutURL}}<a href="{{.CheckoutURL}}" class="button">Продлить подписку</a>{{end}}` + emailFooterRU))

	templates[TemplatePlanChanged] = template.Must(template.New("plan_changed").Parse(emailHead + `
            <h2><span class="status-dot dot-success"></span>Тариф изменён</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Ваш тарифный план успешно изменён на <strong>{{.NewPlanName}}</strong>.</p>
            <p>Новые условия уже действуют.</p>
            {{if .DashboardURL}}<a href="{{.DashboardURL}}" class="button">Перейти в личный кабинет</a>{{end}}` + emailFooterRU))

	templates[TemplatePaymentSuccess] = template.Must(template.New("payment_success").Parse(emailHead + `
            <h2><span class="status-dot dot-success"></span>Оплата прошла успешно</h2>
            <p>Здравствуйте{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Благодарим за оплату!</p>
            <div class="info-block">
                <div class="info-row">
                    <span class="info-label">Тариф</span>
                    <span class="info-value">{{.PlanName}}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">Сумма</span>
                    <span class="info-value">{{.FormattedAmount}}</span>
                </div>
            </div>
            {{if .DashboardURL}}<a href="{{.DashboardURL}}" class="button">Перейти в личный кабинет</a>{{end}}` + emailFooterRU))

	// ── English templates ──────────────────────────────────────────────

	templates[TemplateSubscriptionExpiring+"_en"] = template.Must(template.New("subscription_expiring_en").Parse(emailHead + `
            <h2><span class="status-dot dot-warning"></span>Your subscription is expiring soon</h2>
            <p>Hello{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Your <strong>{{.PlanName}}</strong> subscription expires in <strong>{{.DaysLeft}}</strong> day{{if ne .DaysLeft 1}}s{{end}}.</p>
            <p>Expiration date: <strong>{{.ExpiresAt}}</strong></p>
            {{if .CheckoutURL}}<a href="{{.CheckoutURL}}" class="button">Renew Subscription</a>{{end}}` + emailFooterEN))

	templates[TemplateSubscriptionExpired+"_en"] = template.Must(template.New("subscription_expired_en").Parse(emailHead + `
            <h2><span class="status-dot dot-error"></span>Subscription expired</h2>
            <p>Hello{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Your <strong>{{.PlanName}}</strong> subscription has expired.</p>
            <p>Your account has been downgraded to the free plan with limited features.</p>
            {{if .CheckoutURL}}<a href="{{.CheckoutURL}}" class="button">Subscribe Now</a>{{end}}` + emailFooterEN))

	templates[TemplateSubscriptionRenewed+"_en"] = template.Must(template.New("subscription_renewed_en").Parse(emailHead + `
            <h2><span class="status-dot dot-success"></span>Subscription renewed</h2>
            <p>Hello{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Your <strong>{{.PlanName}}</strong> subscription has been successfully renewed.</p>
            <div class="info-block">
                <div class="info-row">
                    <span class="info-label">Amount</span>
                    <span class="info-value">{{.FormattedAmount}}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">Next renewal</span>
                    <span class="info-value">{{.RenewalDate}}</span>
                </div>
            </div>
            {{if .DashboardURL}}<a href="{{.DashboardURL}}" class="button">Go to Dashboard</a>{{end}}` + emailFooterEN))

	templates[TemplateSubscriptionRenewFailed+"_en"] = template.Must(template.New("subscription_renew_failed_en").Parse(emailHead + `
            <h2><span class="status-dot dot-error"></span>Subscription renewal failed</h2>
            <p>Hello{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>We couldn't automatically renew your <strong>{{.PlanName}}</strong> subscription.</p>
            {{if .ErrorMessage}}<div class="error-box"><strong>Reason:</strong> {{.ErrorMessage}}</div>{{end}}
            <p>Please check your payment details and try renewing manually:</p>
            {{if .CheckoutURL}}<a href="{{.CheckoutURL}}" class="button">Renew Subscription</a>{{end}}` + emailFooterEN))

	templates[TemplatePlanChanged+"_en"] = template.Must(template.New("plan_changed_en").Parse(emailHead + `
            <h2><span class="status-dot dot-success"></span>Plan changed</h2>
            <p>Hello{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Your plan has been changed to <strong>{{.NewPlanName}}</strong>.</p>
            <p>The new plan is now active.</p>
            {{if .DashboardURL}}<a href="{{.DashboardURL}}" class="button">Go to Dashboard</a>{{end}}` + emailFooterEN))

	templates[TemplatePaymentSuccess+"_en"] = template.Must(template.New("payment_success_en").Parse(emailHead + `
            <h2><span class="status-dot dot-success"></span>Payment successful</h2>
            <p>Hello{{if .UserName}}, {{.UserName}}{{end}}!</p>
            <p>Thank you for your payment!</p>
            <div class="info-block">
                <div class="info-row">
                    <span class="info-label">Plan</span>
                    <span class="info-value">{{.PlanName}}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">Amount</span>
                    <span class="info-value">{{.FormattedAmount}}</span>
                </div>
            </div>
            {{if .DashboardURL}}<a href="{{.DashboardURL}}" class="button">Go to Dashboard</a>{{end}}` + emailFooterEN))
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
