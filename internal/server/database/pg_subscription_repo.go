package database

import (
	"context"
	"fmt"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/database/sqlc"
)

// SubscriptionRepository handles subscription database operations using PostgreSQL via sqlc.
type SubscriptionRepository struct {
	q *sqlc.Queries
}

// sqlcSubscriptionToDomain converts a sqlc.Subscription to a domain Subscription.
func sqlcSubscriptionToDomain(s sqlc.Subscription) *Subscription {
	return &Subscription{
		ID:                      s.ID,
		UserID:                  s.UserID,
		PlanID:                  s.PlanID,
		NextPlanID:              int8ToInt64Ptr(s.NextPlanID),
		Status:                  SubscriptionStatus(s.Status),
		Recurring:               s.Recurring,
		CurrentPeriodStart:      tsToTimePtr(s.CurrentPeriodStart),
		CurrentPeriodEnd:        tsToTimePtr(s.CurrentPeriodEnd),
		YooKassaPaymentMethodID: textToStringPtr(s.YookassaPaymentMethodID),
		CreemCustomerID:         textToStringPtr(s.CreemCustomerID),
		CreemSubscriptionID:     textToStringPtr(s.CreemSubscriptionID),
		CreatedAt:               tsToTime(s.CreatedAt),
		UpdatedAt:               tsToTime(s.UpdatedAt),
	}
}

// Create creates a new subscription and populates the ID and timestamps.
func (r *SubscriptionRepository) Create(sub *Subscription) error {
	ctx := context.Background()
	row, err := r.q.CreateSubscription(ctx, sqlc.CreateSubscriptionParams{
		UserID:                  sub.UserID,
		PlanID:                  sub.PlanID,
		NextPlanID:              int64PtrToPgint8(sub.NextPlanID),
		Status:                  string(sub.Status),
		Recurring:               sub.Recurring,
		CurrentPeriodStart:      timePtrToPgtz(sub.CurrentPeriodStart),
		CurrentPeriodEnd:        timePtrToPgtz(sub.CurrentPeriodEnd),
		YookassaPaymentMethodID: stringPtrToPgtext(sub.YooKassaPaymentMethodID),
		CreemCustomerID:         stringPtrToPgtext(sub.CreemCustomerID),
		CreemSubscriptionID:     stringPtrToPgtext(sub.CreemSubscriptionID),
	})
	if err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}
	sub.ID = row.ID
	sub.CreatedAt = tsToTime(row.CreatedAt)
	sub.UpdatedAt = tsToTime(row.UpdatedAt)
	return nil
}

// GetByID retrieves a subscription by ID. Returns nil, nil if not found.
func (r *SubscriptionRepository) GetByID(id int64) (*Subscription, error) {
	ctx := context.Background()
	s, err := r.q.GetSubscriptionByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get subscription by id: %w", err)
	}
	return sqlcSubscriptionToDomain(s), nil
}

// GetByUserID retrieves the active or cancelled subscription for a user. Returns nil, nil if not found.
func (r *SubscriptionRepository) GetByUserID(userID int64) (*Subscription, error) {
	ctx := context.Background()
	s, err := r.q.GetActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get subscription by user id: %w", err)
	}
	return sqlcSubscriptionToDomain(s), nil
}

// GetByCreemSubscriptionID retrieves a subscription by Creem subscription ID. Returns nil, nil if not found.
func (r *SubscriptionRepository) GetByCreemSubscriptionID(creemSubID string) (*Subscription, error) {
	ctx := context.Background()
	s, err := r.q.GetSubscriptionByCreemID(ctx, stringToPgtext(creemSubID))
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get subscription by creem id: %w", err)
	}
	return sqlcSubscriptionToDomain(s), nil
}

// GetPendingByUserID retrieves the most recent pending subscription for a user. Returns nil, nil if not found.
func (r *SubscriptionRepository) GetPendingByUserID(userID int64) (*Subscription, error) {
	ctx := context.Background()
	s, err := r.q.GetPendingSubscriptionByUserID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get pending subscription by user id: %w", err)
	}
	return sqlcSubscriptionToDomain(s), nil
}

// ListByUserID returns all subscriptions for a user.
func (r *SubscriptionRepository) ListByUserID(userID int64) ([]*Subscription, error) {
	ctx := context.Background()
	rows, err := r.q.ListSubscriptionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions by user id: %w", err)
	}
	subs := make([]*Subscription, 0, len(rows))
	for _, s := range rows {
		subs = append(subs, sqlcSubscriptionToDomain(s))
	}
	return subs, nil
}

// ListAll returns all subscriptions with pagination and total count.
func (r *SubscriptionRepository) ListAll(limit, offset int) ([]*Subscription, int, error) {
	ctx := context.Background()
	total, err := r.q.CountAllSubscriptions(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count subscriptions: %w", err)
	}

	rows, err := r.q.ListAllSubscriptions(ctx, sqlc.ListAllSubscriptionsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list all subscriptions: %w", err)
	}
	subs := make([]*Subscription, 0, len(rows))
	for _, s := range rows {
		subs = append(subs, sqlcSubscriptionToDomain(s))
	}
	return subs, int(total), nil
}

// GetExpiring returns active recurring subscriptions expiring within the given duration.
func (r *SubscriptionRepository) GetExpiring(within time.Duration) ([]*Subscription, error) {
	ctx := context.Background()
	deadline := time.Now().Add(within)
	rows, err := r.q.GetExpiringSubscriptions(ctx, timeToPgtz(deadline))
	if err != nil {
		return nil, fmt.Errorf("get expiring subscriptions: %w", err)
	}
	subs := make([]*Subscription, 0, len(rows))
	for _, s := range rows {
		subs = append(subs, sqlcSubscriptionToDomain(s))
	}
	return subs, nil
}

// GetExpired returns subscriptions whose period has ended.
func (r *SubscriptionRepository) GetExpired() ([]*Subscription, error) {
	ctx := context.Background()
	rows, err := r.q.GetExpiredSubscriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get expired subscriptions: %w", err)
	}
	subs := make([]*Subscription, 0, len(rows))
	for _, s := range rows {
		subs = append(subs, sqlcSubscriptionToDomain(s))
	}
	return subs, nil
}

// GetWithPendingPlanChange returns subscriptions that have a next_plan_id and whose period has ended.
func (r *SubscriptionRepository) GetWithPendingPlanChange() ([]*Subscription, error) {
	ctx := context.Background()
	rows, err := r.q.GetSubscriptionsWithPendingPlanChange(ctx)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions with pending plan change: %w", err)
	}
	subs := make([]*Subscription, 0, len(rows))
	for _, s := range rows {
		subs = append(subs, sqlcSubscriptionToDomain(s))
	}
	return subs, nil
}

// GetForRenewalReminder returns active recurring subscriptions expiring within daysAhead days.
func (r *SubscriptionRepository) GetForRenewalReminder(daysAhead int) ([]*Subscription, error) {
	ctx := context.Background()
	now := time.Now()
	rangeEnd := now.Add(time.Duration(daysAhead) * 24 * time.Hour)
	rows, err := r.q.GetSubscriptionsForRenewalReminder(ctx, sqlc.GetSubscriptionsForRenewalReminderParams{
		CurrentPeriodEnd:   timeToPgtz(now),
		CurrentPeriodEnd_2: timeToPgtz(rangeEnd),
	})
	if err != nil {
		return nil, fmt.Errorf("get subscriptions for renewal reminder: %w", err)
	}
	subs := make([]*Subscription, 0, len(rows))
	for _, s := range rows {
		subs = append(subs, sqlcSubscriptionToDomain(s))
	}
	return subs, nil
}

// Update updates an existing subscription.
func (r *SubscriptionRepository) Update(sub *Subscription) error {
	ctx := context.Background()
	err := r.q.UpdateSubscription(ctx, sqlc.UpdateSubscriptionParams{
		ID:                      sub.ID,
		PlanID:                  sub.PlanID,
		NextPlanID:              int64PtrToPgint8(sub.NextPlanID),
		Status:                  string(sub.Status),
		Recurring:               sub.Recurring,
		CurrentPeriodStart:      timePtrToPgtz(sub.CurrentPeriodStart),
		CurrentPeriodEnd:        timePtrToPgtz(sub.CurrentPeriodEnd),
		YookassaPaymentMethodID: stringPtrToPgtext(sub.YooKassaPaymentMethodID),
		CreemCustomerID:         stringPtrToPgtext(sub.CreemCustomerID),
		CreemSubscriptionID:     stringPtrToPgtext(sub.CreemSubscriptionID),
	})
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

// Delete removes a subscription by ID.
func (r *SubscriptionRepository) Delete(id int64) error {
	ctx := context.Background()
	err := r.q.DeleteSubscription(ctx, id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	return nil
}
