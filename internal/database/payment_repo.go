package database

import (
	"database/sql"
	"fmt"
	"time"
)

// PaymentRepository handles payment database operations
type PaymentRepository struct {
	db *sql.DB
}

// NewPaymentRepository creates a new payment repository
func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create creates a new payment record
func (r *PaymentRepository) Create(p *Payment) error {
	result, err := r.db.Exec(`
		INSERT INTO payments (user_id, subscription_id, invoice_id, amount, status, is_recurring, robokassa_data)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.UserID, p.SubscriptionID, p.InvoiceID, p.Amount, p.Status, p.IsRecurring, p.RobokassaData)
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	p.ID = id
	p.CreatedAt = time.Now()

	return nil
}

// GetByID retrieves a payment by ID
func (r *PaymentRepository) GetByID(id int64) (*Payment, error) {
	p := &Payment{}
	err := r.db.QueryRow(`
		SELECT id, user_id, subscription_id, invoice_id, amount, status, is_recurring, robokassa_data, created_at
		FROM payments WHERE id = ?`, id).Scan(
		&p.ID, &p.UserID, &p.SubscriptionID, &p.InvoiceID, &p.Amount, &p.Status,
		&p.IsRecurring, &p.RobokassaData, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payment by id: %w", err)
	}
	return p, nil
}

// GetByInvoiceID retrieves a payment by Robokassa invoice ID
func (r *PaymentRepository) GetByInvoiceID(invoiceID int64) (*Payment, error) {
	p := &Payment{}
	err := r.db.QueryRow(`
		SELECT id, user_id, subscription_id, invoice_id, amount, status, is_recurring, robokassa_data, created_at
		FROM payments WHERE invoice_id = ?`, invoiceID).Scan(
		&p.ID, &p.UserID, &p.SubscriptionID, &p.InvoiceID, &p.Amount, &p.Status,
		&p.IsRecurring, &p.RobokassaData, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payment by invoice id: %w", err)
	}
	return p, nil
}

// Update updates a payment record
func (r *PaymentRepository) Update(p *Payment) error {
	_, err := r.db.Exec(`
		UPDATE payments
		SET subscription_id = ?, status = ?, robokassa_data = ?
		WHERE id = ?`,
		p.SubscriptionID, p.Status, p.RobokassaData, p.ID)
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	return nil
}

// GetByUserID retrieves payments for a user with pagination
func (r *PaymentRepository) GetByUserID(userID int64, limit, offset int) ([]*Payment, int, error) {
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM payments WHERE user_id = ?", userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count user payments: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT id, user_id, subscription_id, invoice_id, amount, status, is_recurring, robokassa_data, created_at
		FROM payments WHERE user_id = ?
		ORDER BY created_at DESC LIMIT ? OFFSET ?`, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get user payments: %w", err)
	}
	defer rows.Close()

	payments, err := r.scanMultiple(rows)
	if err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

// GetPendingBySubscriptionID retrieves pending payments for a subscription
func (r *PaymentRepository) GetPendingBySubscriptionID(subscriptionID int64) ([]*Payment, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, subscription_id, invoice_id, amount, status, is_recurring, robokassa_data, created_at
		FROM payments WHERE subscription_id = ? AND status = 'pending'
		ORDER BY created_at DESC`, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("get pending payments: %w", err)
	}
	defer rows.Close()

	return r.scanMultiple(rows)
}

// ListAll retrieves all payments with pagination
func (r *PaymentRepository) ListAll(limit, offset int) ([]*Payment, int, error) {
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM payments").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count payments: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT id, user_id, subscription_id, invoice_id, amount, status, is_recurring, robokassa_data, created_at
		FROM payments
		ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	payments, err := r.scanMultiple(rows)
	if err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

// GetNextInvoiceID generates the next unique invoice ID
func (r *PaymentRepository) GetNextInvoiceID() (int64, error) {
	var maxID sql.NullInt64
	err := r.db.QueryRow("SELECT MAX(invoice_id) FROM payments").Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("get max invoice id: %w", err)
	}

	if !maxID.Valid {
		// Start from a reasonably large number to avoid conflicts
		return 100001, nil
	}

	return maxID.Int64 + 1, nil
}

func (r *PaymentRepository) scanMultiple(rows *sql.Rows) ([]*Payment, error) {
	var payments []*Payment
	for rows.Next() {
		p := &Payment{}
		err := rows.Scan(
			&p.ID, &p.UserID, &p.SubscriptionID, &p.InvoiceID, &p.Amount, &p.Status,
			&p.IsRecurring, &p.RobokassaData, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, nil
}
