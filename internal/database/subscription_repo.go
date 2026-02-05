package database

import (
	"database/sql"
	"fmt"
	"time"
)

// SubscriptionRepository handles subscription database operations
type SubscriptionRepository struct {
	db *sql.DB
}

// NewSubscriptionRepository creates a new subscription repository
func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// Create creates a new subscription
func (r *SubscriptionRepository) Create(sub *Subscription) error {
	result, err := r.db.Exec(`
		INSERT INTO subscriptions (user_id, plan_id, next_plan_id, status, recurring, current_period_start, current_period_end, yookassa_payment_method_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sub.UserID, sub.PlanID, sub.NextPlanID, sub.Status, sub.Recurring,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.YooKassaPaymentMethodID)
	if err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	sub.ID = id
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = time.Now()

	return nil
}

// GetByID retrieves a subscription by ID
func (r *SubscriptionRepository) GetByID(id int64) (*Subscription, error) {
	sub := &Subscription{}
	err := r.db.QueryRow(`
		SELECT id, user_id, plan_id, next_plan_id, status, recurring,
		       current_period_start, current_period_end, yookassa_payment_method_id, created_at, updated_at
		FROM subscriptions WHERE id = ?`, id).Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.NextPlanID, &sub.Status, &sub.Recurring,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.YooKassaPaymentMethodID,
		&sub.CreatedAt, &sub.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription by id: %w", err)
	}
	return sub, nil
}

// GetByUserID retrieves active subscription for a user
func (r *SubscriptionRepository) GetByUserID(userID int64) (*Subscription, error) {
	sub := &Subscription{}
	err := r.db.QueryRow(`
		SELECT id, user_id, plan_id, next_plan_id, status, recurring,
		       current_period_start, current_period_end, yookassa_payment_method_id, created_at, updated_at
		FROM subscriptions
		WHERE user_id = ? AND status IN ('active', 'cancelled')
		ORDER BY created_at DESC LIMIT 1`, userID).Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.NextPlanID, &sub.Status, &sub.Recurring,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.YooKassaPaymentMethodID,
		&sub.CreatedAt, &sub.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription by user id: %w", err)
	}
	return sub, nil
}

// Update updates a subscription
func (r *SubscriptionRepository) Update(sub *Subscription) error {
	sub.UpdatedAt = time.Now()
	_, err := r.db.Exec(`
		UPDATE subscriptions
		SET plan_id = ?, next_plan_id = ?, status = ?, recurring = ?,
		    current_period_start = ?, current_period_end = ?, yookassa_payment_method_id = ?, updated_at = ?
		WHERE id = ?`,
		sub.PlanID, sub.NextPlanID, sub.Status, sub.Recurring,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.YooKassaPaymentMethodID, sub.UpdatedAt, sub.ID)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

// Delete deletes a subscription by ID
func (r *SubscriptionRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM subscriptions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	return nil
}

// GetExpiring retrieves subscriptions expiring within the given duration
func (r *SubscriptionRepository) GetExpiring(within time.Duration) ([]*Subscription, error) {
	threshold := time.Now().Add(within)
	rows, err := r.db.Query(`
		SELECT id, user_id, plan_id, next_plan_id, status, recurring,
		       current_period_start, current_period_end, yookassa_payment_method_id, created_at, updated_at
		FROM subscriptions
		WHERE status = 'active' AND recurring = 1 AND current_period_end <= ?`, threshold)
	if err != nil {
		return nil, fmt.Errorf("get expiring subscriptions: %w", err)
	}
	defer rows.Close()

	return r.scanMultiple(rows)
}

// GetExpired retrieves subscriptions that have expired
func (r *SubscriptionRepository) GetExpired() ([]*Subscription, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, plan_id, next_plan_id, status, recurring,
		       current_period_start, current_period_end, yookassa_payment_method_id, created_at, updated_at
		FROM subscriptions
		WHERE status = 'active' AND current_period_end < ?`, time.Now())
	if err != nil {
		return nil, fmt.Errorf("get expired subscriptions: %w", err)
	}
	defer rows.Close()

	return r.scanMultiple(rows)
}

// GetWithPendingPlanChange retrieves subscriptions with a pending plan change
func (r *SubscriptionRepository) GetWithPendingPlanChange() ([]*Subscription, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, plan_id, next_plan_id, status, recurring,
		       current_period_start, current_period_end, yookassa_payment_method_id, created_at, updated_at
		FROM subscriptions
		WHERE next_plan_id IS NOT NULL AND current_period_end < ?`, time.Now())
	if err != nil {
		return nil, fmt.Errorf("get subscriptions with plan change: %w", err)
	}
	defer rows.Close()

	return r.scanMultiple(rows)
}

// GetForRenewalReminder retrieves subscriptions that need renewal reminder
func (r *SubscriptionRepository) GetForRenewalReminder(daysAhead int) ([]*Subscription, error) {
	start := time.Now().AddDate(0, 0, daysAhead)
	end := start.AddDate(0, 0, 1)
	rows, err := r.db.Query(`
		SELECT id, user_id, plan_id, next_plan_id, status, recurring,
		       current_period_start, current_period_end, yookassa_payment_method_id, created_at, updated_at
		FROM subscriptions
		WHERE status = 'active' AND recurring = 1 AND current_period_end >= ? AND current_period_end < ?`,
		start, end)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions for reminder: %w", err)
	}
	defer rows.Close()

	return r.scanMultiple(rows)
}

// ListAll retrieves all subscriptions with pagination
func (r *SubscriptionRepository) ListAll(limit, offset int) ([]*Subscription, int, error) {
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM subscriptions").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count subscriptions: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT id, user_id, plan_id, next_plan_id, status, recurring,
		       current_period_start, current_period_end, yookassa_payment_method_id, created_at, updated_at
		FROM subscriptions
		ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	subs, err := r.scanMultiple(rows)
	if err != nil {
		return nil, 0, err
	}

	return subs, total, nil
}

func (r *SubscriptionRepository) scanMultiple(rows *sql.Rows) ([]*Subscription, error) {
	var subs []*Subscription
	for rows.Next() {
		sub := &Subscription{}
		err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.NextPlanID, &sub.Status, &sub.Recurring,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.YooKassaPaymentMethodID,
			&sub.CreatedAt, &sub.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	return subs, nil
}
