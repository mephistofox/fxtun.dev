package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// UserRepository handles user database operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *User) error {
	query := `
		INSERT INTO users (phone, password_hash, display_name, is_admin, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		user.Phone,
		user.PasswordHash,
		user.DisplayName,
		user.IsAdmin,
		user.IsActive,
		now,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	user.ID = id
	user.CreatedAt = now
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id int64) (*User, error) {
	query := `
		SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at
		FROM users WHERE id = ?
	`

	user := &User{}
	var lastLoginAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Phone,
		&user.PasswordHash,
		&user.DisplayName,
		&user.IsAdmin,
		&user.IsActive,
		&user.CreatedAt,
		&lastLoginAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// GetByPhone retrieves a user by phone number
func (r *UserRepository) GetByPhone(phone string) (*User, error) {
	query := `
		SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at
		FROM users WHERE phone = ?
	`

	user := &User{}
	var lastLoginAt sql.NullTime

	err := r.db.QueryRow(query, phone).Scan(
		&user.ID,
		&user.Phone,
		&user.PasswordHash,
		&user.DisplayName,
		&user.IsAdmin,
		&user.IsActive,
		&user.CreatedAt,
		&lastLoginAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by phone: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// Update updates user information
func (r *UserRepository) Update(user *User) error {
	query := `
		UPDATE users
		SET display_name = ?, is_admin = ?, is_active = ?, last_login_at = ?
		WHERE id = ?
	`

	result, err := r.db.Exec(query,
		user.DisplayName,
		user.IsAdmin,
		user.IsActive,
		user.LastLoginAt,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdatePassword updates user's password hash
func (r *UserRepository) UpdatePassword(userID int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = ? WHERE id = ?`

	result, err := r.db.Exec(query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(userID int64) error {
	query := `UPDATE users SET last_login_at = ? WHERE id = ?`

	now := time.Now()
	_, err := r.db.Exec(query, now, userID)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}

	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(id int64) error {
	query := `DELETE FROM users WHERE id = ?`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// List returns all users with pagination
func (r *UserRepository) List(limit, offset int) ([]*User, int, error) {
	countQuery := `SELECT COUNT(*) FROM users`
	var total int
	if err := r.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	query := `
		SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at
		FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		var lastLoginAt sql.NullTime

		if err := rows.Scan(
			&user.ID,
			&user.Phone,
			&user.PasswordHash,
			&user.DisplayName,
			&user.IsAdmin,
			&user.IsActive,
			&user.CreatedAt,
			&lastLoginAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}

		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		users = append(users, user)
	}

	return users, total, nil
}

// GetByIDs retrieves multiple users by their IDs
func (r *UserRepository) GetByIDs(ids []int64) (map[int64]*User, error) {
	if len(ids) == 0 {
		return make(map[int64]*User), nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at
		FROM users WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("get users by ids: %w", err)
	}
	defer rows.Close()

	users := make(map[int64]*User)
	for rows.Next() {
		user := &User{}
		var lastLoginAt sql.NullTime
		if err := rows.Scan(
			&user.ID, &user.Phone, &user.PasswordHash, &user.DisplayName,
			&user.IsAdmin, &user.IsActive, &user.CreatedAt, &lastLoginAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}
		users[user.ID] = user
	}

	return users, nil
}

// Count returns the total number of users
func (r *UserRepository) Count() (int, error) {
	query := `SELECT COUNT(*) FROM users`
	var count int
	if err := r.db.QueryRow(query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

// isUniqueConstraintError checks if the error is a unique constraint violation
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return containsString(err.Error(), "UNIQUE constraint failed")
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
