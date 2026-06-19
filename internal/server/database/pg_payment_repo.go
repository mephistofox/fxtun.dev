package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mephistofox/fxtun.dev/internal/server/database/sqlc"
)

// PaymentRepository handles payment database operations using PostgreSQL via sqlc.
type PaymentRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

// sqlcPaymentToDomain converts a sqlc.Payment to a domain Payment.
func sqlcPaymentToDomain(p sqlc.Payment) *Payment {
	return &Payment{
		ID:             p.ID,
		UserID:         p.UserID,
		SubscriptionID: int8ToInt64Ptr(p.SubscriptionID),
		InvoiceID:      p.InvoiceID,
		Amount:         p.Amount,
		Status:         PaymentStatus(p.Status),
		IsRecurring:    p.IsRecurring,
		YooKassaData:   textToString(p.YookassaData),
		Provider:       p.Provider,
		ProviderData:   textToString(p.ProviderData),
		CreatedAt:      tsToTime(p.CreatedAt),
	}
}

// Create creates a new payment and populates the ID and CreatedAt.
func (r *PaymentRepository) Create(p *Payment) error {
	ctx := context.Background()
	row, err := r.q.CreatePayment(ctx, sqlc.CreatePaymentParams{
		UserID:         p.UserID,
		SubscriptionID: int64PtrToPgint8(p.SubscriptionID),
		InvoiceID:      p.InvoiceID,
		Amount:         p.Amount,
		Status:         string(p.Status),
		IsRecurring:    p.IsRecurring,
		YookassaData:   stringToPgtext(p.YooKassaData),
		Provider:       p.Provider,
		ProviderData:   stringToPgtext(p.ProviderData),
	})
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	p.ID = row.ID
	p.CreatedAt = tsToTime(row.CreatedAt)
	return nil
}

// GetByID retrieves a payment by ID. Returns nil, nil if not found.
func (r *PaymentRepository) GetByID(id int64) (*Payment, error) {
	ctx := context.Background()
	p, err := r.q.GetPaymentByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get payment by id: %w", err)
	}
	return sqlcPaymentToDomain(p), nil
}

// GetByInvoiceID retrieves a payment by invoice ID. Returns nil, nil if not found.
func (r *PaymentRepository) GetByInvoiceID(invoiceID int64) (*Payment, error) {
	ctx := context.Background()
	p, err := r.q.GetPaymentByInvoiceID(ctx, invoiceID)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get payment by invoice id: %w", err)
	}
	return sqlcPaymentToDomain(p), nil
}

// GetByUserID returns payments for a user with pagination and total count.
func (r *PaymentRepository) GetByUserID(userID int64, limit, offset int) ([]*Payment, int, error) {
	ctx := context.Background()
	total, err := r.q.CountPaymentsByUserID(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("count payments by user: %w", err)
	}

	rows, err := r.q.ListPaymentsByUserID(ctx, sqlc.ListPaymentsByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list payments by user id: %w", err)
	}
	payments := make([]*Payment, 0, len(rows))
	for _, p := range rows {
		payments = append(payments, sqlcPaymentToDomain(p))
	}
	return payments, int(total), nil
}

// GetPendingBySubscriptionID returns pending payments for a subscription.
func (r *PaymentRepository) GetPendingBySubscriptionID(subscriptionID int64) ([]*Payment, error) {
	ctx := context.Background()
	rows, err := r.q.GetPendingPaymentsBySubscriptionID(ctx, int64ToPgint8(subscriptionID))
	if err != nil {
		return nil, fmt.Errorf("get pending payments by subscription id: %w", err)
	}
	payments := make([]*Payment, 0, len(rows))
	for _, p := range rows {
		payments = append(payments, sqlcPaymentToDomain(p))
	}
	return payments, nil
}

// ListAll returns all payments with pagination and total count.
func (r *PaymentRepository) ListAll(limit, offset int) ([]*Payment, int, error) {
	ctx := context.Background()
	total, err := r.q.CountAllPayments(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count payments: %w", err)
	}

	rows, err := r.q.ListAllPayments(ctx, sqlc.ListAllPaymentsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list all payments: %w", err)
	}
	payments := make([]*Payment, 0, len(rows))
	for _, p := range rows {
		payments = append(payments, sqlcPaymentToDomain(p))
	}
	return payments, int(total), nil
}

// Update updates an existing payment.
func (r *PaymentRepository) Update(p *Payment) error {
	ctx := context.Background()
	err := r.q.UpdatePayment(ctx, sqlc.UpdatePaymentParams{
		ID:             p.ID,
		SubscriptionID: int64PtrToPgint8(p.SubscriptionID),
		Status:         string(p.Status),
		YookassaData:   stringToPgtext(p.YooKassaData),
		Provider:       p.Provider,
		ProviderData:   stringToPgtext(p.ProviderData),
	})
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	return nil
}

// GetNextInvoiceID returns the next available invoice ID.
func (r *PaymentRepository) GetNextInvoiceID() (int64, error) {
	ctx := context.Background()
	nextID, err := r.q.GetNextInvoiceID(ctx)
	if err != nil {
		return 0, fmt.Errorf("get next invoice id: %w", err)
	}
	return int64(nextID), nil
}

// DeleteStalePending expires stale pending subscriptions and fails stale pending payments
// older than the given duration. Returns the number of failed payments.
func (r *PaymentRepository) DeleteStalePending(olderThan time.Duration) (int64, error) {
	ctx := context.Background()
	cutoff := timeToPgtz(time.Now().Add(-olderThan))

	err := r.q.ExpireStalePendingSubscriptions(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("expire stale pending subscriptions: %w", err)
	}

	count, err := r.q.FailStalePendingPayments(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("fail stale pending payments: %w", err)
	}
	return count, nil
}

// PaymentsByDay returns successful payment amounts grouped by day for the given number of days.
func (r *PaymentRepository) PaymentsByDay(days int) ([]DailyStat, error) {
	ctx := context.Background()
	query := `SELECT DATE(created_at AT TIME ZONE 'UTC') AS date, COALESCE(SUM(amount), 0) AS value
		FROM payments
		WHERE status = 'success' AND created_at >= NOW() - make_interval(days := $1)
		GROUP BY DATE(created_at AT TIME ZONE 'UTC')
		ORDER BY date`

	rows, err := r.pool.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("payments by day: %w", err)
	}
	defer rows.Close()

	var results []DailyStat
	for rows.Next() {
		var item DailyStat
		if err := rows.Scan(&item.Date, &item.Value); err != nil {
			return nil, fmt.Errorf("scan payments by day: %w", err)
		}
		results = append(results, item)
	}
	return results, rows.Err()
}
