package email

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/database"
	"github.com/mephistofox/fxtun.dev/internal/scheduler"
)

// Notifier handles sending notifications based on scheduler events
type Notifier struct {
	email        *Service
	db           *database.Database
	log          zerolog.Logger
	baseURL      string
	supportEmail string
}

// NewNotifier creates a new notifier
func NewNotifier(email *Service, db *database.Database, baseURL string, supportEmail string, log zerolog.Logger) *Notifier {
	return &Notifier{
		email:        email,
		db:           db,
		log:          log.With().Str("component", "notifier").Logger(),
		baseURL:      baseURL,
		supportEmail: supportEmail,
	}
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

	var subject string
	var templateName string
	var data TemplateData

	data.UserName = user.DisplayName
	data.UserEmail = user.Email
	data.DashboardURL = n.baseURL + "/dashboard"
	data.CheckoutURL = n.baseURL + "/checkout"
	data.SupportEmail = n.supportEmail

	if event.Plan != nil {
		data.PlanName = event.Plan.Name
		data.Amount = event.Plan.Price
	}

	switch event.Type {
	case scheduler.EventSubscriptionExpiring:
		subject = fmt.Sprintf("Подписка истекает через %d дн.", event.DaysLeft)
		templateName = TemplateSubscriptionExpiring
		data.DaysLeft = event.DaysLeft
		if event.Subscription != nil && event.Subscription.CurrentPeriodEnd != nil {
			data.ExpiresAt = event.Subscription.CurrentPeriodEnd.Format("02.01.2006")
		}

	case scheduler.EventSubscriptionExpired:
		subject = "Подписка истекла"
		templateName = TemplateSubscriptionExpired

	case scheduler.EventSubscriptionRenewed:
		subject = "Подписка продлена"
		templateName = TemplateSubscriptionRenewed
		if event.Subscription != nil && event.Subscription.CurrentPeriodEnd != nil {
			data.RenewalDate = event.Subscription.CurrentPeriodEnd.Format("02.01.2006")
		}

	case scheduler.EventSubscriptionRenewFailed:
		subject = "Ошибка продления подписки"
		templateName = TemplateSubscriptionRenewFailed
		if event.Error != nil {
			data.ErrorMessage = event.Error.Error()
		}

	case scheduler.EventPlanChanged:
		subject = "Тариф изменён"
		templateName = TemplatePlanChanged
		data.NewPlanName = data.PlanName

	default:
		n.log.Debug().Str("type", string(event.Type)).Msg("Unknown event type, skipping")
		return
	}

	if err := n.email.SendTemplate(user.Email, subject, templateName, data); err != nil {
		n.log.Error().Err(err).
			Str("email", user.Email).
			Str("template", templateName).
			Msg("Failed to send notification email")
	}
}

// SendPaymentSuccessNotification sends payment success notification
func (n *Notifier) SendPaymentSuccessNotification(userID int64, planName string, amount float64) error {
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

	data := TemplateData{
		UserName:     user.DisplayName,
		UserEmail:    user.Email,
		PlanName:     planName,
		Amount:       amount,
		DashboardURL: n.baseURL + "/dashboard",
		SupportEmail: n.supportEmail,
	}

	return n.email.SendTemplate(user.Email, "Оплата прошла успешно", TemplatePaymentSuccess, data)
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

	expiresAt := ""
	if sub.CurrentPeriodEnd != nil {
		expiresAt = sub.CurrentPeriodEnd.Format("02.01.2006")
	}

	data := TemplateData{
		UserName:     user.DisplayName,
		UserEmail:    user.Email,
		PlanName:     plan.Name,
		DaysLeft:     daysLeft,
		ExpiresAt:    expiresAt,
		CheckoutURL:  n.baseURL + "/checkout",
		SupportEmail: n.supportEmail,
	}

	subject := fmt.Sprintf("Подписка истекает через %d дн.", daysLeft)
	return n.email.SendTemplate(user.Email, subject, TemplateSubscriptionExpiring, data)
}
