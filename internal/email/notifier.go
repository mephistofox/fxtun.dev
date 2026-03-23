package email

import (
	"fmt"
	"math"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/database"
	"github.com/mephistofox/fxtunnel/internal/scheduler"
)

// Notifier handles sending notifications based on scheduler events
type Notifier struct {
	email        *Service
	db           *database.Database
	log          zerolog.Logger
	baseURL      string // Base URL for Russian emails (e.g. https://fxtun.ru)
	baseURLEN    string // Base URL for English emails (e.g. https://fxtun.dev)
	supportEmail string
}

// NewNotifier creates a new notifier
func NewNotifier(email *Service, db *database.Database, baseURL, baseURLEN, supportEmail string, log zerolog.Logger) *Notifier {
	return &Notifier{
		email:        email,
		db:           db,
		log:          log.With().Str("component", "notifier").Logger(),
		baseURL:      baseURL,
		baseURLEN:    baseURLEN,
		supportEmail: supportEmail,
	}
}

// detectLang determines the email language from the subscription's payment provider.
// Creem subscriptions → English, everything else → Russian.
func detectLang(sub *database.Subscription) string {
	if sub == nil {
		return "ru"
	}
	if sub.CreemSubscriptionID != nil && *sub.CreemSubscriptionID != "" {
		return "en"
	}
	if sub.CreemCustomerID != nil && *sub.CreemCustomerID != "" {
		return "en"
	}
	return "ru"
}

// detectLangByProvider returns the language for a given payment provider name.
func detectLangByProvider(provider string) string {
	if provider == "creem" {
		return "en"
	}
	return "ru"
}

// getBaseURL returns the appropriate base URL for the language.
func (n *Notifier) getBaseURL(lang string) string {
	if lang == "en" && n.baseURLEN != "" {
		return n.baseURLEN
	}
	return n.baseURL
}

// formatAmount formats an amount with the appropriate currency symbol.
func formatAmount(amount float64, lang string) string {
	if lang == "en" {
		// USD: $10 or $10.50
		if amount == math.Trunc(amount) {
			return fmt.Sprintf("$%.0f", amount)
		}
		return fmt.Sprintf("$%.2f", amount)
	}
	// RUB: 350 ₽
	return fmt.Sprintf("%.0f ₽", amount)
}

// HandleSchedulerEvent handles events from the subscription scheduler
func (n *Notifier) HandleSchedulerEvent(event scheduler.Event) {
	if n.email == nil || !n.email.IsEnabled() {
		return
	}

	user, err := n.db.Users.GetByID(event.UserID)
	if err != nil || user == nil {
		n.log.Error().Err(err).Int64("user_id", event.UserID).Msg("Failed to get user")
		return
	}

	if user.Email == "" {
		n.log.Debug().Int64("user_id", event.UserID).Msg("User has no email, skipping notification")
		return
	}

	lang := detectLang(event.Subscription)
	base := n.getBaseURL(lang)

	var subject string
	var templateName string
	var data TemplateData

	data.UserName = user.DisplayName
	data.UserEmail = user.Email
	data.DashboardURL = base + "/dashboard"
	data.CheckoutURL = base + "/checkout"
	data.SupportEmail = n.supportEmail

	if event.Plan != nil {
		data.PlanName = event.Plan.Name
		data.Amount = event.Plan.Price
		data.FormattedAmount = formatAmount(event.Plan.Price, lang)
	}

	switch event.Type {
	case scheduler.EventSubscriptionExpiring:
		data.DaysLeft = event.DaysLeft
		if event.Subscription != nil && event.Subscription.CurrentPeriodEnd != nil {
			if lang == "en" {
				data.ExpiresAt = event.Subscription.CurrentPeriodEnd.Format("Jan 2, 2006")
			} else {
				data.ExpiresAt = event.Subscription.CurrentPeriodEnd.Format("02.01.2006")
			}
		}
		if lang == "en" {
			subject = fmt.Sprintf("Your subscription expires in %d day(s)", event.DaysLeft)
		} else {
			subject = fmt.Sprintf("Подписка истекает через %d дн.", event.DaysLeft)
		}
		templateName = LocalizedTemplateName(TemplateSubscriptionExpiring, lang)

	case scheduler.EventSubscriptionExpired:
		if lang == "en" {
			subject = "Your subscription has expired"
		} else {
			subject = "Подписка истекла"
		}
		templateName = LocalizedTemplateName(TemplateSubscriptionExpired, lang)

	case scheduler.EventSubscriptionRenewed:
		if event.Subscription != nil && event.Subscription.CurrentPeriodEnd != nil {
			if lang == "en" {
				data.RenewalDate = event.Subscription.CurrentPeriodEnd.Format("Jan 2, 2006")
			} else {
				data.RenewalDate = event.Subscription.CurrentPeriodEnd.Format("02.01.2006")
			}
		}
		if lang == "en" {
			subject = "Subscription renewed"
		} else {
			subject = "Подписка продлена"
		}
		templateName = LocalizedTemplateName(TemplateSubscriptionRenewed, lang)

	case scheduler.EventSubscriptionRenewFailed:
		if event.Error != nil {
			data.ErrorMessage = event.Error.Error()
		}
		if lang == "en" {
			subject = "Subscription renewal failed"
		} else {
			subject = "Ошибка продления подписки"
		}
		templateName = LocalizedTemplateName(TemplateSubscriptionRenewFailed, lang)

	case scheduler.EventPlanChanged:
		data.NewPlanName = data.PlanName
		if lang == "en" {
			subject = "Plan changed"
		} else {
			subject = "Тариф изменён"
		}
		templateName = LocalizedTemplateName(TemplatePlanChanged, lang)

	default:
		n.log.Debug().Str("type", string(event.Type)).Msg("Unknown event type, skipping")
		return
	}

	if err := n.email.SendTemplate(user.Email, subject, templateName, data); err != nil {
		n.log.Error().Err(err).
			Str("email", user.Email).
			Str("template", templateName).
			Str("lang", lang).
			Msg("Failed to send notification email")
	}
}

// SendPaymentSuccessNotification sends payment success notification
func (n *Notifier) SendPaymentSuccessNotification(userID int64, planName string, amount float64, provider string) error {
	if n.email == nil || !n.email.IsEnabled() {
		return nil
	}

	user, err := n.db.Users.GetByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("get user: %w", err)
	}

	if user.Email == "" {
		return nil
	}

	lang := detectLangByProvider(provider)
	base := n.getBaseURL(lang)

	data := TemplateData{
		UserName:        user.DisplayName,
		UserEmail:       user.Email,
		PlanName:        planName,
		Amount:          amount,
		FormattedAmount: formatAmount(amount, lang),
		DashboardURL:    base + "/dashboard",
		SupportEmail:    n.supportEmail,
	}

	var subject string
	if lang == "en" {
		subject = "Payment successful"
	} else {
		subject = "Оплата прошла успешно"
	}

	templateName := LocalizedTemplateName(TemplatePaymentSuccess, lang)
	return n.email.SendTemplate(user.Email, subject, templateName, data)
}

// SendExpirationReminder sends subscription expiration reminder
func (n *Notifier) SendExpirationReminder(sub *database.Subscription, plan *database.Plan, daysLeft int) error {
	if n.email == nil || !n.email.IsEnabled() {
		return nil
	}

	user, err := n.db.Users.GetByID(sub.UserID)
	if err != nil || user == nil {
		return fmt.Errorf("get user: %w", err)
	}

	if user.Email == "" {
		return nil
	}

	lang := detectLang(sub)
	base := n.getBaseURL(lang)

	expiresAt := ""
	if sub.CurrentPeriodEnd != nil {
		if lang == "en" {
			expiresAt = sub.CurrentPeriodEnd.Format("Jan 2, 2006")
		} else {
			expiresAt = sub.CurrentPeriodEnd.Format("02.01.2006")
		}
	}

	data := TemplateData{
		UserName:     user.DisplayName,
		UserEmail:    user.Email,
		PlanName:     plan.Name,
		DaysLeft:     daysLeft,
		ExpiresAt:    expiresAt,
		CheckoutURL:  base + "/checkout",
		SupportEmail: n.supportEmail,
	}

	var subject string
	if lang == "en" {
		subject = fmt.Sprintf("Your subscription expires in %d day(s)", daysLeft)
	} else {
		subject = fmt.Sprintf("Подписка истекает через %d дн.", daysLeft)
	}

	templateName := LocalizedTemplateName(TemplateSubscriptionExpiring, lang)
	return n.email.SendTemplate(user.Email, subject, templateName, data)
}
