package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mephistofox/fxtunnel/internal/server/database/sqlc"
)

// UserRepository handles user database operations using PostgreSQL via sqlc.
type UserRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

// sqlcUserToDomain converts a sqlc.User to a domain User.
func sqlcUserToDomain(u sqlc.User) *User {
	return &User{
		ID:            u.ID,
		Phone:         textToString(u.Phone),
		PasswordHash:  u.PasswordHash,
		DisplayName:   textToString(u.DisplayName),
		IsAdmin:       u.IsAdmin,
		IsActive:      u.IsActive,
		CreatedAt:     tsToTime(u.CreatedAt),
		LastLoginAt:   tsToTimePtr(u.LastLoginAt),
		GitHubID:      int8ToInt64Ptr(u.GithubID),
		GoogleID:      textToStringPtr(u.GoogleID),
		Email:         textToString(u.Email),
		AvatarURL:     textToString(u.AvatarUrl),
		PlanID:        int8ToInt64(u.PlanID),
		FirstTunnelAt: tsToTimePtr(u.FirstTunnelAt),
	}
}

// Create creates a new user.
func (r *UserRepository) Create(user *User) error {
	ctx := context.Background()
	row, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		Phone:        stringToPgtext(user.Phone),
		PasswordHash: user.PasswordHash,
		DisplayName:  stringToPgtext(user.DisplayName),
		IsAdmin:      user.IsAdmin,
		IsActive:     user.IsActive,
		PlanID:       int64ToPgint8(user.PlanID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	user.ID = row.ID
	user.CreatedAt = tsToTime(row.CreatedAt)
	return nil
}

// CreateTx creates a new user within a transaction.
func (r *UserRepository) CreateTx(tx pgx.Tx, user *User) error {
	ctx := context.Background()
	row, err := r.q.WithTx(tx).CreateUser(ctx, sqlc.CreateUserParams{
		Phone:        stringToPgtext(user.Phone),
		PasswordHash: user.PasswordHash,
		DisplayName:  stringToPgtext(user.DisplayName),
		IsAdmin:      user.IsAdmin,
		IsActive:     user.IsActive,
		PlanID:       int64ToPgint8(user.PlanID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	user.ID = row.ID
	user.CreatedAt = tsToTime(row.CreatedAt)
	return nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(id int64) (*User, error) {
	ctx := context.Background()
	u, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return sqlcUserToDomain(u), nil
}

// GetByPhone retrieves a user by phone number.
func (r *UserRepository) GetByPhone(phone string) (*User, error) {
	ctx := context.Background()
	u, err := r.q.GetUserByPhone(ctx, stringToPgtext(phone))
	if err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by phone: %w", err)
	}
	return sqlcUserToDomain(u), nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(email string) (*User, error) {
	ctx := context.Background()
	u, err := r.q.GetUserByEmail(ctx, stringToPgtext(email))
	if err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return sqlcUserToDomain(u), nil
}

// GetByGitHubID retrieves a user by GitHub ID.
func (r *UserRepository) GetByGitHubID(githubID int64) (*User, error) {
	ctx := context.Background()
	u, err := r.q.GetUserByGitHubID(ctx, int64ToPgint8(githubID))
	if err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by github id: %w", err)
	}
	return sqlcUserToDomain(u), nil
}

// GetByGoogleID retrieves a user by Google ID.
func (r *UserRepository) GetByGoogleID(googleID string) (*User, error) {
	ctx := context.Background()
	u, err := r.q.GetUserByGoogleID(ctx, stringToPgtext(googleID))
	if err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by google id: %w", err)
	}
	return sqlcUserToDomain(u), nil
}

// GetByIDs retrieves multiple users by their IDs.
func (r *UserRepository) GetByIDs(ids []int64) (map[int64]*User, error) {
	if len(ids) == 0 {
		return make(map[int64]*User), nil
	}
	ctx := context.Background()
	rows, err := r.q.GetUsersByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get users by ids: %w", err)
	}
	users := make(map[int64]*User, len(rows))
	for _, u := range rows {
		users[u.ID] = sqlcUserToDomain(u)
	}
	return users, nil
}

// Update updates user information.
func (r *UserRepository) Update(user *User) error {
	ctx := context.Background()
	err := r.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:          user.ID,
		DisplayName: stringToPgtext(user.DisplayName),
		IsAdmin:     user.IsAdmin,
		IsActive:    user.IsActive,
		LastLoginAt: timePtrToPgtz(user.LastLoginAt),
		PlanID:      int64ToPgint8(user.PlanID),
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// UpdatePassword updates user's password hash.
func (r *UserRepository) UpdatePassword(userID int64, passwordHash string) error {
	ctx := context.Background()
	err := r.q.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

// UpdateEmail updates user's email.
func (r *UserRepository) UpdateEmail(userID int64, email string) error {
	ctx := context.Background()
	err := r.q.UpdateUserEmail(ctx, sqlc.UpdateUserEmailParams{
		ID:    userID,
		Email: stringToPgtext(email),
	})
	if err != nil {
		return fmt.Errorf("update email: %w", err)
	}
	return nil
}

// UpdatePhone updates a user's phone field.
func (r *UserRepository) UpdatePhone(userID int64, phone string) error {
	ctx := context.Background()
	err := r.q.UpdateUserPhone(ctx, sqlc.UpdateUserPhoneParams{
		ID:    userID,
		Phone: stringToPgtext(phone),
	})
	if err != nil {
		return fmt.Errorf("update phone: %w", err)
	}
	return nil
}

// UpdateLastLogin updates the last login timestamp.
func (r *UserRepository) UpdateLastLogin(userID int64) error {
	ctx := context.Background()
	err := r.q.UpdateUserLastLogin(ctx, userID)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}

// UpdatePlan updates the user's plan.
func (r *UserRepository) UpdatePlan(userID int64, planID int64) error {
	ctx := context.Background()
	err := r.q.UpdateUserPlan(ctx, sqlc.UpdateUserPlanParams{
		ID:     userID,
		PlanID: int64ToPgint8(planID),
	})
	if err != nil {
		return fmt.Errorf("update plan: %w", err)
	}
	return nil
}

// LinkGitHub links a GitHub account to an existing user.
func (r *UserRepository) LinkGitHub(userID, githubID int64, email, avatarURL string) error {
	ctx := context.Background()
	err := r.q.LinkGitHub(ctx, sqlc.LinkGitHubParams{
		ID:        userID,
		GithubID:  int64ToPgint8(githubID),
		Email:     stringToPgtext(email),
		AvatarUrl: stringToPgtext(avatarURL),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("github account already linked to another user")
		}
		return fmt.Errorf("link github: %w", err)
	}
	return nil
}

// LinkGoogle links a Google account to an existing user.
func (r *UserRepository) LinkGoogle(userID int64, googleID, email, avatarURL string) error {
	ctx := context.Background()
	err := r.q.LinkGoogle(ctx, sqlc.LinkGoogleParams{
		ID:        userID,
		GoogleID:  stringToPgtext(googleID),
		Email:     stringToPgtext(email),
		AvatarUrl: stringToPgtext(avatarURL),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("google account already linked to another user")
		}
		return fmt.Errorf("link google: %w", err)
	}
	return nil
}

// CreateOAuth creates a new user via OAuth (no phone/password required).
func (r *UserRepository) CreateOAuth(user *User) error {
	ctx := context.Background()
	row, err := r.q.CreateOAuthUser(ctx, sqlc.CreateOAuthUserParams{
		Phone:       stringToPgtext(user.Phone),
		DisplayName: stringToPgtext(user.DisplayName),
		IsAdmin:     user.IsAdmin,
		IsActive:    user.IsActive,
		GithubID:    int64PtrToPgint8(user.GitHubID),
		GoogleID:    stringPtrToPgtext(user.GoogleID),
		Email:       stringToPgtext(user.Email),
		AvatarUrl:   stringToPgtext(user.AvatarURL),
		PlanID:      int64ToPgint8(user.PlanID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("create oauth user: %w", err)
	}
	user.ID = row.ID
	user.CreatedAt = tsToTime(row.CreatedAt)
	return nil
}

// Delete deletes a user.
func (r *UserRepository) Delete(id int64) error {
	ctx := context.Background()
	err := r.q.DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// DeleteTx deletes a user within a transaction.
func (r *UserRepository) DeleteTx(tx pgx.Tx, id int64) error {
	ctx := context.Background()
	_, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user tx: %w", err)
	}
	return nil
}

// BulkUpdateActive sets is_active for multiple users, excluding the given admin user ID.
func (r *UserRepository) BulkUpdateActive(userIDs []int64, isActive bool, excludeUserID int64) (int64, error) {
	ctx := context.Background()
	query := `UPDATE users SET is_active = $1 WHERE id = ANY($2) AND id != $3`
	result, err := r.pool.Exec(ctx, query, isActive, userIDs, excludeUserID)
	if err != nil {
		return 0, fmt.Errorf("bulk update active: %w", err)
	}
	return result.RowsAffected(), nil
}

// BulkUpdatePlan sets plan_id for multiple users, excluding the given admin user ID.
func (r *UserRepository) BulkUpdatePlan(userIDs []int64, planID int64, excludeUserID int64) (int64, error) {
	ctx := context.Background()
	query := `UPDATE users SET plan_id = $1 WHERE id = ANY($2) AND id != $3`
	result, err := r.pool.Exec(ctx, query, planID, userIDs, excludeUserID)
	if err != nil {
		return 0, fmt.Errorf("bulk update plan: %w", err)
	}
	return result.RowsAffected(), nil
}

// BulkDelete deletes multiple users in a single transaction, excluding the given admin user ID.
// Returns the number of successfully deleted users.
func (r *UserRepository) BulkDelete(userIDs []int64, excludeUserID int64) (int64, []string, error) {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var successCount int64
	var errs []string
	for _, uid := range userIDs {
		if uid == excludeUserID {
			errs = append(errs, fmt.Sprintf("user %d: cannot modify your own account", uid))
			continue
		}
		_, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, uid)
		if err != nil {
			errs = append(errs, fmt.Sprintf("user %d: operation failed", uid))
			continue
		}
		successCount++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, nil, fmt.Errorf("commit transaction: %w", err)
	}
	return successCount, errs, nil
}

// buildFilterParams converts UserListParams to the sqlc filter/count parameters.
func buildFilterParams(params UserListParams) (isActive pgtype.Bool, isAdmin pgtype.Bool, search pgtype.Text) {
	switch params.Filter {
	case "active":
		isActive = pgtype.Bool{Bool: true, Valid: true}
	case "blocked":
		isActive = pgtype.Bool{Bool: false, Valid: true}
	case "admins":
		isAdmin = pgtype.Bool{Bool: true, Valid: true}
	}
	if params.Search != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(params.Search))
		s := "%" + escaped + "%"
		search = pgtype.Text{String: s, Valid: true}
	}
	return
}

// allowedSortColumns is a whitelist of columns that can be used for sorting users.
var allowedSortColumns = map[string]string{
	"created_at":   "created_at",
	"last_login_at": "last_login_at",
	"email":        "email",
	"display_name": "display_name",
	"id":           "id",
}

// List returns users with filtering, search, and pagination.
func (r *UserRepository) List(params UserListParams) ([]*User, int, error) {
	ctx := context.Background()
	isActive, isAdmin, search := buildFilterParams(params)

	total, err := r.q.CountUsersFiltered(ctx, sqlc.CountUsersFilteredParams{
		IsActive: isActive,
		IsAdmin:  isAdmin,
		Search:   search,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// If sorting is requested and valid, use raw query with ORDER BY
	if params.SortBy != "" {
		col, ok := allowedSortColumns[params.SortBy]
		if ok {
			return r.listWithSort(ctx, params, isActive, isAdmin, search, col, int(total))
		}
	}

	rows, err := r.q.ListUsersFiltered(ctx, sqlc.ListUsersFilteredParams{
		Limit:    int32(params.Limit),
		Offset:   int32(params.Offset),
		IsActive: isActive,
		IsAdmin:  isAdmin,
		Search:   search,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	users := make([]*User, 0, len(rows))
	for _, u := range rows {
		users = append(users, sqlcUserToDomain(u))
	}
	return users, int(total), nil
}

// listWithSort performs a raw query with dynamic ORDER BY (whitelisted column).
func (r *UserRepository) listWithSort(ctx context.Context, params UserListParams, isActive pgtype.Bool, isAdmin pgtype.Bool, search pgtype.Text, sortCol string, total int) ([]*User, int, error) {
	order := "ASC"
	if strings.EqualFold(params.Order, "desc") {
		order = "DESC"
	}

	//nolint:gosec // sortCol is from allowedSortColumns whitelist, order is hardcoded ASC/DESC
	query := fmt.Sprintf(`SELECT id, phone, password_hash, display_name, is_admin, is_active,
		created_at, last_login_at, github_id, google_id, email, avatar_url, plan_id, first_tunnel_at
		FROM users
		WHERE ($1::boolean IS NULL OR is_active = $1)
		  AND ($2::boolean IS NULL OR is_admin = $2)
		  AND ($3::text IS NULL OR LOWER(email) LIKE $3 ESCAPE '\' OR LOWER(phone) LIKE $3 ESCAPE '\' OR LOWER(display_name) LIKE $3 ESCAPE '\')
		ORDER BY %s %s
		LIMIT $4 OFFSET $5`, sortCol, order)

	rows, err := r.pool.Query(ctx, query, isActive, isAdmin, search, params.Limit, params.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users sorted: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u sqlc.User
		if err := rows.Scan(
			&u.ID, &u.Phone, &u.PasswordHash, &u.DisplayName,
			&u.IsAdmin, &u.IsActive, &u.CreatedAt, &u.LastLoginAt,
			&u.GithubID, &u.GoogleID, &u.Email, &u.AvatarUrl,
			&u.PlanID, &u.FirstTunnelAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan sorted user: %w", err)
		}
		users = append(users, sqlcUserToDomain(u))
	}
	return users, total, rows.Err()
}

// Stats returns aggregate user counts, optionally scoped by search term.
func (r *UserRepository) Stats(search string) (*UserStats, error) {
	ctx := context.Background()
	var searchParam pgtype.Text
	if search != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(search))
		s := "%" + escaped + "%"
		searchParam = pgtype.Text{String: s, Valid: true}
	}

	row, err := r.q.GetUserStats(ctx, searchParam)
	if err != nil {
		return nil, fmt.Errorf("user stats: %w", err)
	}
	return &UserStats{
		Total:   int(row.Total),
		Active:  int(row.Active),
		Blocked: int(row.Blocked),
		Admins:  int(row.Admins),
	}, nil
}

// SetFirstTunnelAt sets the first tunnel creation timestamp if not already set.
// Returns true if this was the first tunnel (value was NULL and got updated).
func (r *UserRepository) SetFirstTunnelAt(userID int64) (bool, error) {
	ctx := context.Background()
	rows, err := r.q.SetFirstTunnelAt(ctx, sqlc.SetFirstTunnelAtParams{
		ID:            userID,
		FirstTunnelAt: timeToPgtz(time.Now()),
	})
	if err != nil {
		return false, fmt.Errorf("set first_tunnel_at: %w", err)
	}
	return rows > 0, nil
}

// Count returns the total number of users.
func (r *UserRepository) Count() (int, error) {
	ctx := context.Background()
	count, err := r.q.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return int(count), nil
}

// MergeUsers transfers all data from secondary user to primary user and deletes the secondary user.
// OAuth fields are copied to primary if they are empty.
func (r *UserRepository) MergeUsers(primaryID, secondaryID int64) error {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Transfer simple foreign key tables
	tables := []string{"sessions", "api_tokens", "reserved_domains", "totp_secrets", "custom_domains", "audit_logs", "user_history"}
	for _, table := range tables {
		//nolint:gosec // table names are hardcoded constants
		_, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET user_id = $1 WHERE user_id = $2`, table), primaryID, secondaryID)
		if err != nil {
			return fmt.Errorf("transfer %s: %w", table, err)
		}
	}

	// Transfer user_bundles (has UNIQUE(user_id, name) constraint)
	_, err = tx.Exec(ctx,
		`UPDATE user_bundles SET user_id = $1 WHERE user_id = $2 AND name NOT IN (SELECT name FROM user_bundles WHERE user_id = $1)`,
		primaryID, secondaryID)
	if err != nil {
		return fmt.Errorf("transfer user_bundles: %w", err)
	}
	_, err = tx.Exec(ctx, `DELETE FROM user_bundles WHERE user_id = $1`, secondaryID)
	if err != nil {
		return fmt.Errorf("cleanup user_bundles: %w", err)
	}

	// Transfer user_settings (has PRIMARY KEY(user_id, key))
	_, err = tx.Exec(ctx,
		`UPDATE user_settings SET user_id = $1 WHERE user_id = $2 AND key NOT IN (SELECT key FROM user_settings WHERE user_id = $1)`,
		primaryID, secondaryID)
	if err != nil {
		return fmt.Errorf("transfer user_settings: %w", err)
	}
	_, err = tx.Exec(ctx, `DELETE FROM user_settings WHERE user_id = $1`, secondaryID)
	if err != nil {
		return fmt.Errorf("cleanup user_settings: %w", err)
	}

	// Copy OAuth fields from secondary to primary if primary's are empty
	_, err = tx.Exec(ctx, `
		UPDATE users SET
			github_id = COALESCE(github_id, (SELECT github_id FROM users WHERE id = $1)),
			google_id = COALESCE(google_id, (SELECT google_id FROM users WHERE id = $1)),
			email = CASE WHEN email = '' OR email IS NULL THEN (SELECT email FROM users WHERE id = $1) ELSE email END,
			avatar_url = CASE WHEN avatar_url = '' OR avatar_url IS NULL THEN (SELECT avatar_url FROM users WHERE id = $1) ELSE avatar_url END
		WHERE id = $2
	`, secondaryID, primaryID)
	if err != nil {
		return fmt.Errorf("merge oauth fields: %w", err)
	}

	// Delete secondary user
	_, err = tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, secondaryID)
	if err != nil {
		return fmt.Errorf("delete secondary user: %w", err)
	}

	return tx.Commit(ctx)
}

// RegistrationsByDay returns user registration counts grouped by day for the given number of days.
func (r *UserRepository) RegistrationsByDay(days int) ([]DailyStat, error) {
	ctx := context.Background()
	query := `SELECT DATE(created_at AT TIME ZONE 'UTC') AS date, COUNT(*)::float8 AS value
		FROM users
		WHERE created_at >= NOW() - make_interval(days := $1)
		GROUP BY DATE(created_at AT TIME ZONE 'UTC')
		ORDER BY date`

	rows, err := r.pool.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("registrations by day: %w", err)
	}
	defer rows.Close()

	var results []DailyStat
	for rows.Next() {
		var item DailyStat
		if err := rows.Scan(&item.Date, &item.Value); err != nil {
			return nil, fmt.Errorf("scan registrations by day: %w", err)
		}
		results = append(results, item)
	}
	return results, rows.Err()
}
