package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mephistofox/fxtunnel/internal/server/database/sqlc"
)

// UserBundleRepository handles user bundle database operations using PostgreSQL via sqlc.
type UserBundleRepository struct {
	q *sqlc.Queries
}

// sqlcBundleToDomain converts a sqlc.UserBundle to a domain UserBundle.
func sqlcBundleToDomain(b sqlc.UserBundle) *UserBundle {
	return &UserBundle{
		ID:          b.ID,
		UserID:      b.UserID,
		Name:        b.Name,
		Type:        b.Type,
		LocalPort:   int(b.LocalPort),
		Subdomain:   textToString(b.Subdomain),
		RemotePort:  int4ToInt(b.RemotePort),
		AutoConnect: b.AutoConnect,
		CreatedAt:   tsToTime(b.CreatedAt),
		UpdatedAt:   tsToTime(b.UpdatedAt),
	}
}

// bundleSubdomainToPg converts a domain subdomain string to pgtype.Text.
func bundleSubdomainToPg(s string) pgtype.Text {
	return stringToPgtext(s)
}

// bundleRemotePortToPg converts a domain remote port int to pgtype.Int4.
func bundleRemotePortToPg(port int) pgtype.Int4 {
	if port == 0 {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(port), Valid: true}
}

// Create creates a new user bundle.
func (r *UserBundleRepository) Create(bundle *UserBundle) error {
	ctx := context.Background()
	row, err := r.q.CreateBundle(ctx, sqlc.CreateBundleParams{
		UserID:      bundle.UserID,
		Name:        bundle.Name,
		Type:        bundle.Type,
		LocalPort:   int32(bundle.LocalPort),
		Subdomain:   bundleSubdomainToPg(bundle.Subdomain),
		RemotePort:  bundleRemotePortToPg(bundle.RemotePort),
		AutoConnect: bundle.AutoConnect,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrBundleAlreadyExists
		}
		return fmt.Errorf("create bundle: %w", err)
	}
	bundle.ID = row.ID
	bundle.CreatedAt = tsToTime(row.CreatedAt)
	bundle.UpdatedAt = tsToTime(row.UpdatedAt)
	return nil
}

// Update updates an existing user bundle.
func (r *UserBundleRepository) Update(bundle *UserBundle) error {
	ctx := context.Background()
	err := r.q.UpdateBundle(ctx, sqlc.UpdateBundleParams{
		ID:          bundle.ID,
		UserID:      bundle.UserID,
		Name:        bundle.Name,
		Type:        bundle.Type,
		LocalPort:   int32(bundle.LocalPort),
		Subdomain:   bundleSubdomainToPg(bundle.Subdomain),
		RemotePort:  bundleRemotePortToPg(bundle.RemotePort),
		AutoConnect: bundle.AutoConnect,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrBundleAlreadyExists
		}
		return fmt.Errorf("update bundle: %w", err)
	}
	return nil
}

// Delete deletes a user bundle by ID and user ID.
func (r *UserBundleRepository) Delete(id, userID int64) error {
	ctx := context.Background()
	err := r.q.DeleteBundle(ctx, sqlc.DeleteBundleParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("delete bundle: %w", err)
	}
	return nil
}

// DeleteByName deletes a user bundle by name.
func (r *UserBundleRepository) DeleteByName(userID int64, name string) error {
	ctx := context.Background()
	err := r.q.DeleteBundleByName(ctx, sqlc.DeleteBundleByNameParams{
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return fmt.Errorf("delete bundle by name: %w", err)
	}
	return nil
}

// DeleteAll deletes all bundles for a user.
func (r *UserBundleRepository) DeleteAll(userID int64) error {
	ctx := context.Background()
	err := r.q.DeleteAllBundles(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete all bundles: %w", err)
	}
	return nil
}

// GetByID retrieves a user bundle by ID and user ID.
func (r *UserBundleRepository) GetByID(id, userID int64) (*UserBundle, error) {
	ctx := context.Background()
	b, err := r.q.GetBundleByID(ctx, sqlc.GetBundleByIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrBundleNotFound
		}
		return nil, fmt.Errorf("get bundle by id: %w", err)
	}
	return sqlcBundleToDomain(b), nil
}

// GetByName retrieves a user bundle by name.
func (r *UserBundleRepository) GetByName(userID int64, name string) (*UserBundle, error) {
	ctx := context.Background()
	b, err := r.q.GetBundleByName(ctx, sqlc.GetBundleByNameParams{
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrBundleNotFound
		}
		return nil, fmt.Errorf("get bundle by name: %w", err)
	}
	return sqlcBundleToDomain(b), nil
}

// GetByUserID retrieves all bundles for a user.
func (r *UserBundleRepository) GetByUserID(userID int64) ([]*UserBundle, error) {
	ctx := context.Background()
	rows, err := r.q.ListBundlesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get bundles by user id: %w", err)
	}
	bundles := make([]*UserBundle, 0, len(rows))
	for _, b := range rows {
		bundles = append(bundles, sqlcBundleToDomain(b))
	}
	return bundles, nil
}

// Count returns the number of bundles for a user.
func (r *UserBundleRepository) Count(userID int64) (int, error) {
	ctx := context.Background()
	count, err := r.q.CountBundlesByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count bundles: %w", err)
	}
	return int(count), nil
}

// SyncBulk synchronizes bundles for a user using upsert with conflict resolution by updated_at.
func (r *UserBundleRepository) SyncBulk(userID int64, bundles []*UserBundle) error {
	ctx := context.Background()
	for _, bundle := range bundles {
		bundle.UserID = userID
		row, err := r.q.UpsertBundle(ctx, sqlc.UpsertBundleParams{
			UserID:      bundle.UserID,
			Name:        bundle.Name,
			Type:        bundle.Type,
			LocalPort:   int32(bundle.LocalPort),
			Subdomain:   bundleSubdomainToPg(bundle.Subdomain),
			RemotePort:  bundleRemotePortToPg(bundle.RemotePort),
			AutoConnect: bundle.AutoConnect,
			CreatedAt:   timeToPgtz(bundle.CreatedAt),
			UpdatedAt:   timeToPgtz(bundle.UpdatedAt),
		})
		if err != nil {
			return fmt.Errorf("upsert bundle %q: %w", bundle.Name, err)
		}
		bundle.ID = row.ID
		bundle.CreatedAt = tsToTime(row.CreatedAt)
		bundle.UpdatedAt = tsToTime(row.UpdatedAt)
	}
	return nil
}
