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

// CreateTx creates a new user within a transaction
func (r *UserRepository) CreateTx(tx *sql.Tx, user *User) error {
	query := `
		INSERT INTO users (phone, password_hash, display_name, is_admin, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := tx.Exec(query,
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
	user, err := scanUser(r.db.QueryRow(
		`SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id
		FROM users WHERE id = ?`, id,
	))
	if err != nil {
		return nil, notFoundOrError(err, ErrUserNotFound, "get user by id")
	}
	return user, nil
}

// GetByPhone retrieves a user by phone number
func (r *UserRepository) GetByPhone(phone string) (*User, error) {
	user, err := scanUser(r.db.QueryRow(
		`SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id
		FROM users WHERE phone = ?`, phone,
	))
	if err != nil {
		return nil, notFoundOrError(err, ErrUserNotFound, "get user by phone")
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

	rows, err := r.db.Query(
		`SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id
		FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user, err := scanUserRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
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

	query := fmt.Sprintf(`SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id FROM users WHERE id IN (%s)`, strings.Join(placeholders, ",")) //nolint:gosec // placeholders are all "?", no SQL injection

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("get users by ids: %w", err)
	}
	defer rows.Close()

	users := make(map[int64]*User)
	for rows.Next() {
		user, err := scanUserRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
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

// scanner is an interface satisfied by both *sql.Row and *sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanUserFromScanner(s scanner) (*User, error) {
	user := &User{}
	var lastLoginAt sql.NullTime
	var githubID sql.NullInt64
	var googleID sql.NullString
	var email sql.NullString
	var avatarURL sql.NullString

	err := s.Scan(
		&user.ID, &user.Phone, &user.PasswordHash, &user.DisplayName,
		&user.IsAdmin, &user.IsActive, &user.CreatedAt, &lastLoginAt,
		&githubID, &email, &avatarURL, &googleID,
	)
	if err != nil {
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if githubID.Valid {
		user.GitHubID = &githubID.Int64
	}
	if googleID.Valid {
		user.GoogleID = &googleID.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}

	return user, nil
}

func scanUser(row *sql.Row) (*User, error) {
	return scanUserFromScanner(row)
}

func scanUserRows(rows *sql.Rows) (*User, error) {
	return scanUserFromScanner(rows)
}

// GetByGitHubID retrieves a user by GitHub ID
func (r *UserRepository) GetByGitHubID(githubID int64) (*User, error) {
	user, err := scanUser(r.db.QueryRow(
		`SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id
		FROM users WHERE github_id = ?`, githubID,
	))
	if err != nil {
		return nil, notFoundOrError(err, ErrUserNotFound, "get user by github id")
	}
	return user, nil
}

// LinkGitHub links a GitHub account to an existing user
func (r *UserRepository) LinkGitHub(userID, githubID int64, email, avatarURL string) error {
	query := `
		UPDATE users
		SET github_id = ?, email = COALESCE(NULLIF(email, ''), ?), avatar_url = COALESCE(NULLIF(avatar_url, ''), ?)
		WHERE id = ?
	`
	result, err := r.db.Exec(query, githubID, email, avatarURL, userID)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("github account already linked to another user")
		}
		return fmt.Errorf("link github: %w", err)
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

// CreateOAuth creates a new user via OAuth (no phone/password required)
func (r *UserRepository) CreateOAuth(user *User) error {
	query := `
		INSERT INTO users (phone, password_hash, display_name, is_admin, is_active, github_id, google_id, email, avatar_url, created_at)
		VALUES (?, '', ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		user.Phone,
		user.DisplayName,
		user.IsAdmin,
		user.IsActive,
		user.GitHubID,
		user.GoogleID,
		user.Email,
		user.AvatarURL,
		now,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("create oauth user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	user.ID = id
	user.CreatedAt = now
	return nil
}

// GetByGoogleID retrieves a user by Google ID
func (r *UserRepository) GetByGoogleID(googleID string) (*User, error) {
	user, err := scanUser(r.db.QueryRow(
		`SELECT id, phone, password_hash, display_name, is_admin, is_active, created_at, last_login_at, github_id, email, avatar_url, google_id
		FROM users WHERE google_id = ?`, googleID,
	))
	if err != nil {
		return nil, notFoundOrError(err, ErrUserNotFound, "get user by google id")
	}
	return user, nil
}

// LinkGoogle links a Google account to an existing user
func (r *UserRepository) LinkGoogle(userID int64, googleID, email, avatarURL string) error {
	query := `
		UPDATE users
		SET google_id = ?, email = COALESCE(NULLIF(email, ''), ?), avatar_url = COALESCE(NULLIF(avatar_url, ''), ?)
		WHERE id = ?
	`
	result, err := r.db.Exec(query, googleID, email, avatarURL, userID)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("google account already linked to another user")
		}
		return fmt.Errorf("link google: %w", err)
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
