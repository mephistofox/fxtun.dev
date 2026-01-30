package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrBundleNotFound = errors.New("bundle not found")
var ErrBundleAlreadyExists = errors.New("bundle already exists")

// UserBundleRepository handles user bundle database operations
type UserBundleRepository struct {
	db *sql.DB
}

// NewUserBundleRepository creates a new user bundle repository
func NewUserBundleRepository(db *sql.DB) *UserBundleRepository {
	return &UserBundleRepository{db: db}
}

// Create creates a new user bundle
func (r *UserBundleRepository) Create(bundle *UserBundle) error {
	query := `
		INSERT INTO user_bundles (user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		bundle.UserID,
		bundle.Name,
		bundle.Type,
		bundle.LocalPort,
		bundle.Subdomain,
		bundle.RemotePort,
		bundle.AutoConnect,
		now,
		now,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrBundleAlreadyExists
		}
		return fmt.Errorf("create user bundle: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	bundle.ID = id
	bundle.CreatedAt = now
	bundle.UpdatedAt = now
	return nil
}

// Update updates an existing user bundle
func (r *UserBundleRepository) Update(bundle *UserBundle) error {
	query := `
		UPDATE user_bundles
		SET name = ?, type = ?, local_port = ?, subdomain = ?, remote_port = ?, auto_connect = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		bundle.Name,
		bundle.Type,
		bundle.LocalPort,
		bundle.Subdomain,
		bundle.RemotePort,
		bundle.AutoConnect,
		now,
		bundle.ID,
		bundle.UserID,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrBundleAlreadyExists
		}
		return fmt.Errorf("update user bundle: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrBundleNotFound
	}

	bundle.UpdatedAt = now
	return nil
}

// Delete deletes a user bundle
func (r *UserBundleRepository) Delete(id, userID int64) error {
	query := `DELETE FROM user_bundles WHERE id = ? AND user_id = ?`

	result, err := r.db.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("delete user bundle: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrBundleNotFound
	}

	return nil
}

// DeleteByName deletes a user bundle by name
func (r *UserBundleRepository) DeleteByName(userID int64, name string) error {
	query := `DELETE FROM user_bundles WHERE user_id = ? AND name = ?`

	result, err := r.db.Exec(query, userID, name)
	if err != nil {
		return fmt.Errorf("delete user bundle by name: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrBundleNotFound
	}

	return nil
}

// GetByID retrieves a user bundle by ID
func (r *UserBundleRepository) GetByID(id, userID int64) (*UserBundle, error) {
	query := `
		SELECT id, user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
		FROM user_bundles WHERE id = ? AND user_id = ?
	`

	bundle := &UserBundle{}
	var subdomain sql.NullString
	var remotePort sql.NullInt64

	err := r.db.QueryRow(query, id, userID).Scan(
		&bundle.ID,
		&bundle.UserID,
		&bundle.Name,
		&bundle.Type,
		&bundle.LocalPort,
		&subdomain,
		&remotePort,
		&bundle.AutoConnect,
		&bundle.CreatedAt,
		&bundle.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBundleNotFound
		}
		return nil, fmt.Errorf("get user bundle by id: %w", err)
	}

	if subdomain.Valid {
		bundle.Subdomain = subdomain.String
	}
	if remotePort.Valid {
		bundle.RemotePort = int(remotePort.Int64)
	}

	return bundle, nil
}

// GetByName retrieves a user bundle by name
func (r *UserBundleRepository) GetByName(userID int64, name string) (*UserBundle, error) {
	query := `
		SELECT id, user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
		FROM user_bundles WHERE user_id = ? AND name = ?
	`

	bundle := &UserBundle{}
	var subdomain sql.NullString
	var remotePort sql.NullInt64

	err := r.db.QueryRow(query, userID, name).Scan(
		&bundle.ID,
		&bundle.UserID,
		&bundle.Name,
		&bundle.Type,
		&bundle.LocalPort,
		&subdomain,
		&remotePort,
		&bundle.AutoConnect,
		&bundle.CreatedAt,
		&bundle.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBundleNotFound
		}
		return nil, fmt.Errorf("get user bundle by name: %w", err)
	}

	if subdomain.Valid {
		bundle.Subdomain = subdomain.String
	}
	if remotePort.Valid {
		bundle.RemotePort = int(remotePort.Int64)
	}

	return bundle, nil
}

// GetByUserID retrieves all bundles for a user
func (r *UserBundleRepository) GetByUserID(userID int64) ([]*UserBundle, error) {
	query := `
		SELECT id, user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
		FROM user_bundles WHERE user_id = ? ORDER BY name
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user bundles: %w", err)
	}
	defer rows.Close()

	var bundles []*UserBundle
	for rows.Next() {
		bundle := &UserBundle{}
		var subdomain sql.NullString
		var remotePort sql.NullInt64

		if err := rows.Scan(
			&bundle.ID,
			&bundle.UserID,
			&bundle.Name,
			&bundle.Type,
			&bundle.LocalPort,
			&subdomain,
			&remotePort,
			&bundle.AutoConnect,
			&bundle.CreatedAt,
			&bundle.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user bundle: %w", err)
		}

		if subdomain.Valid {
			bundle.Subdomain = subdomain.String
		}
		if remotePort.Valid {
			bundle.RemotePort = int(remotePort.Int64)
		}

		bundles = append(bundles, bundle)
	}

	return bundles, nil
}

// SyncBulk synchronizes bundles for a user (upsert with conflict resolution by updated_at)
func (r *UserBundleRepository) SyncBulk(userID int64, bundles []*UserBundle) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, bundle := range bundles {
		bundle.UserID = userID

		// Check if bundle exists
		existing, err := r.getByNameTx(tx, userID, bundle.Name)
		if err != nil && !errors.Is(err, ErrBundleNotFound) {
			return fmt.Errorf("check existing bundle: %w", err)
		}

		if existing == nil {
			// Insert new bundle
			if err := r.createTx(tx, bundle); err != nil {
				return fmt.Errorf("create bundle: %w", err)
			}
		} else {
			// Update if incoming is newer
			if bundle.UpdatedAt.After(existing.UpdatedAt) {
				bundle.ID = existing.ID
				if err := r.updateTx(tx, bundle); err != nil {
					return fmt.Errorf("update bundle: %w", err)
				}
			}
		}
	}

	return tx.Commit()
}

func (r *UserBundleRepository) getByNameTx(tx *sql.Tx, userID int64, name string) (*UserBundle, error) {
	query := `
		SELECT id, user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
		FROM user_bundles WHERE user_id = ? AND name = ?
	`

	bundle := &UserBundle{}
	var subdomain sql.NullString
	var remotePort sql.NullInt64

	err := tx.QueryRow(query, userID, name).Scan(
		&bundle.ID,
		&bundle.UserID,
		&bundle.Name,
		&bundle.Type,
		&bundle.LocalPort,
		&subdomain,
		&remotePort,
		&bundle.AutoConnect,
		&bundle.CreatedAt,
		&bundle.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBundleNotFound
		}
		return nil, err
	}

	if subdomain.Valid {
		bundle.Subdomain = subdomain.String
	}
	if remotePort.Valid {
		bundle.RemotePort = int(remotePort.Int64)
	}

	return bundle, nil
}

func (r *UserBundleRepository) createTx(tx *sql.Tx, bundle *UserBundle) error {
	query := `
		INSERT INTO user_bundles (user_id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	if bundle.CreatedAt.IsZero() {
		bundle.CreatedAt = now
	}
	if bundle.UpdatedAt.IsZero() {
		bundle.UpdatedAt = now
	}

	result, err := tx.Exec(query,
		bundle.UserID,
		bundle.Name,
		bundle.Type,
		bundle.LocalPort,
		bundle.Subdomain,
		bundle.RemotePort,
		bundle.AutoConnect,
		bundle.CreatedAt,
		bundle.UpdatedAt,
	)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	bundle.ID = id
	return nil
}

func (r *UserBundleRepository) updateTx(tx *sql.Tx, bundle *UserBundle) error {
	query := `
		UPDATE user_bundles
		SET name = ?, type = ?, local_port = ?, subdomain = ?, remote_port = ?, auto_connect = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`

	_, err := tx.Exec(query,
		bundle.Name,
		bundle.Type,
		bundle.LocalPort,
		bundle.Subdomain,
		bundle.RemotePort,
		bundle.AutoConnect,
		bundle.UpdatedAt,
		bundle.ID,
		bundle.UserID,
	)
	return err
}

// Count returns the number of bundles for a user
func (r *UserBundleRepository) Count(userID int64) (int, error) {
	query := `SELECT COUNT(*) FROM user_bundles WHERE user_id = ?`
	var count int
	if err := r.db.QueryRow(query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count user bundles: %w", err)
	}
	return count, nil
}

// DeleteAll deletes all bundles for a user
func (r *UserBundleRepository) DeleteAll(userID int64) error {
	query := `DELETE FROM user_bundles WHERE user_id = ?`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("delete all user bundles: %w", err)
	}
	return nil
}
