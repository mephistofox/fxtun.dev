package telegram

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// AdminNotifier wraps Bot and provides domain-specific notification methods
// for admin alerts (new users, subscriptions, first tunnels, etc.).
type AdminNotifier struct {
	bot    *Bot
	chatID string
	log    zerolog.Logger
}

// NewAdminNotifier creates a new AdminNotifier that sends messages to the given chat.
func NewAdminNotifier(bot *Bot, chatID string) *AdminNotifier {
	return &AdminNotifier{
		bot:    bot,
		chatID: chatID,
		log:    zerolog.Nop(),
	}
}

// SetLogger configures the logger used for reporting send errors.
func (n *AdminNotifier) SetLogger(log zerolog.Logger) {
	n.log = log.With().Str("component", "telegram-notifier").Logger()
}

// send dispatches the message in a fire-and-forget goroutine.
func (n *AdminNotifier) send(text string) {
	go func() {
		if err := n.bot.SendMessage(n.chatID, text); err != nil {
			n.log.Error().Err(err).Msg("failed to send telegram notification")
		}
	}()
}

// NotifyNewUser sends a notification about a new user registration.
func (n *AdminNotifier) NotifyNewUser(userID int64, displayName, email string) {
	msg := fmt.Sprintf(
		"🆕 <b>Новый пользователь</b>\nИмя: %s\nEmail: %s\nID: %d\nВремя: %s",
		escapeHTML(displayName),
		escapeHTML(email),
		userID,
		time.Now().UTC().Format("2006-01-02 15:04 UTC"),
	)
	n.send(msg)
}

// NotifyNewSubscription sends a notification about a subscription activation.
func (n *AdminNotifier) NotifyNewSubscription(userID int64, displayName, planName string, amount float64, provider string) {
	msg := fmt.Sprintf(
		"💳 <b>Новая подписка</b>\nПользователь: %s (ID: %d)\nПлан: %s\nСумма: %.2f\nПровайдер: %s",
		escapeHTML(displayName),
		userID,
		escapeHTML(planName),
		amount,
		escapeHTML(provider),
	)
	n.send(msg)
}

// NotifyRegistrationTarpit reports a bot that hit the disabled phone/password
// registration endpoint. We show everything the attacker sent, so the operator
// can see patterns and pick additional defenses.
func (n *AdminNotifier) NotifyRegistrationTarpit(phone, password, displayName, ip, userAgent string) {
	msg := fmt.Sprintf(
		"🕷 <b>Tarpit: попытка регистрации</b>\nPhone: <code>%s</code>\nPassword: <code>%s</code>\nName: <code>%s</code>\nIP: <code>%s</code>\nUser-Agent: <code>%s</code>\nВремя: %s",
		escapeHTML(phone),
		escapeHTML(password),
		escapeHTML(displayName),
		escapeHTML(ip),
		escapeHTML(userAgent),
		time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
	)
	n.send(msg)
}

// NotifyFirstTunnel sends a notification about a user's first-ever tunnel creation.
func (n *AdminNotifier) NotifyFirstTunnel(userID int64, displayName, tunnelType, address string, registeredAt time.Time) {
	duration := time.Since(registeredAt)
	msg := fmt.Sprintf(
		"🚀 <b>Первый туннель</b>\nПользователь: %s (ID: %d)\nТип: %s\nАдрес: %s\nЗарегистрирован: %s назад",
		escapeHTML(displayName),
		userID,
		escapeHTML(tunnelType),
		escapeHTML(address),
		formatDuration(duration),
	)
	n.send(msg)
}

// escapeHTML escapes &, <, > for Telegram HTML parse mode.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// formatDuration returns a human-readable Russian duration string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "менее минуты"
	}

	if d < time.Hour {
		mins := int(d.Minutes())
		return fmt.Sprintf("%d мин", mins)
	}

	if d < 24*time.Hour {
		hours := int(d.Hours())
		return fmt.Sprintf("%d ч", hours)
	}

	days := int(d.Hours() / 24)
	return fmt.Sprintf("%d д", days)
}
