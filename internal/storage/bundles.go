package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// Bundle represents a saved tunnel configuration
type Bundle struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // http, tcp, udp
	LocalPort   int       `json:"local_port"`
	Subdomain   string    `json:"subdomain,omitempty"`
	RemotePort  int       `json:"remote_port,omitempty"`
	AutoConnect bool      `json:"auto_connect"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BundleRepository provides CRUD operations for bundles
type BundleRepository struct {
	db *Database
}

// NewBundleRepository creates a new bundle repository
func NewBundleRepository(db *Database) *BundleRepository {
	return &BundleRepository{db: db}
}

// List returns all bundles
func (r *BundleRepository) List() ([]Bundle, error) {
	rows, err := r.db.db.Query(`
		SELECT id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
		FROM bundles
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("query bundles: %w", err)
	}
	defer rows.Close()

	var bundles []Bundle
	for rows.Next() {
		var b Bundle
		var subdomain, remotePort sql.NullString
		if err := rows.Scan(&b.ID, &b.Name, &b.Type, &b.LocalPort, &subdomain, &remotePort, &b.AutoConnect, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan bundle: %w", err)
		}
		if subdomain.Valid {
			b.Subdomain = subdomain.String
		}
		if remotePort.Valid {
			var port int
			fmt.Sscanf(remotePort.String, "%d", &port)
			b.RemotePort = port
		}
		bundles = append(bundles, b)
	}

	return bundles, rows.Err()
}

// GetByID returns a bundle by ID
func (r *BundleRepository) GetByID(id int64) (*Bundle, error) {
	var b Bundle
	var subdomain sql.NullString
	var remotePort sql.NullInt64

	err := r.db.db.QueryRow(`
		SELECT id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
		FROM bundles
		WHERE id = ?
	`, id).Scan(&b.ID, &b.Name, &b.Type, &b.LocalPort, &subdomain, &remotePort, &b.AutoConnect, &b.CreatedAt, &b.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query bundle: %w", err)
	}

	if subdomain.Valid {
		b.Subdomain = subdomain.String
	}
	if remotePort.Valid {
		b.RemotePort = int(remotePort.Int64)
	}

	return &b, nil
}

// Create creates a new bundle
func (r *BundleRepository) Create(b *Bundle) error {
	now := time.Now()
	result, err := r.db.db.Exec(`
		INSERT INTO bundles (name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, b.Name, b.Type, b.LocalPort, nullString(b.Subdomain), nullInt(b.RemotePort), b.AutoConnect, now, now)

	if err != nil {
		return fmt.Errorf("insert bundle: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	b.ID = id
	b.CreatedAt = now
	b.UpdatedAt = now

	return nil
}

// Update updates an existing bundle
func (r *BundleRepository) Update(b *Bundle) error {
	now := time.Now()
	_, err := r.db.db.Exec(`
		UPDATE bundles
		SET name = ?, type = ?, local_port = ?, subdomain = ?, remote_port = ?, auto_connect = ?, updated_at = ?
		WHERE id = ?
	`, b.Name, b.Type, b.LocalPort, nullString(b.Subdomain), nullInt(b.RemotePort), b.AutoConnect, now, b.ID)

	if err != nil {
		return fmt.Errorf("update bundle: %w", err)
	}

	b.UpdatedAt = now
	return nil
}

// Delete deletes a bundle
func (r *BundleRepository) Delete(id int64) error {
	_, err := r.db.db.Exec("DELETE FROM bundles WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete bundle: %w", err)
	}
	return nil
}

// GetByName returns a bundle by name
func (r *BundleRepository) GetByName(name string) (*Bundle, error) {
	var b Bundle
	var subdomain sql.NullString
	var remotePort sql.NullInt64

	err := r.db.db.QueryRow(`
		SELECT id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
		FROM bundles
		WHERE name = ?
	`, name).Scan(&b.ID, &b.Name, &b.Type, &b.LocalPort, &subdomain, &remotePort, &b.AutoConnect, &b.CreatedAt, &b.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query bundle: %w", err)
	}

	if subdomain.Valid {
		b.Subdomain = subdomain.String
	}
	if remotePort.Valid {
		b.RemotePort = int(remotePort.Int64)
	}

	return &b, nil
}

// GetAutoConnect returns all bundles marked for auto-connect
func (r *BundleRepository) GetAutoConnect() ([]Bundle, error) {
	rows, err := r.db.db.Query(`
		SELECT id, name, type, local_port, subdomain, remote_port, auto_connect, created_at, updated_at
		FROM bundles
		WHERE auto_connect = TRUE
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("query auto-connect bundles: %w", err)
	}
	defer rows.Close()

	var bundles []Bundle
	for rows.Next() {
		var b Bundle
		var subdomain sql.NullString
		var remotePort sql.NullInt64
		if err := rows.Scan(&b.ID, &b.Name, &b.Type, &b.LocalPort, &subdomain, &remotePort, &b.AutoConnect, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan bundle: %w", err)
		}
		if subdomain.Valid {
			b.Subdomain = subdomain.String
		}
		if remotePort.Valid {
			b.RemotePort = int(remotePort.Int64)
		}
		bundles = append(bundles, b)
	}

	return bundles, rows.Err()
}

// Helper functions
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}
