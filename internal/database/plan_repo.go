package database

import (
	"database/sql"
	"fmt"
)

// PlanRepository handles plan database operations
type PlanRepository struct {
	db *sql.DB
}

// NewPlanRepository creates a new plan repository
func NewPlanRepository(db *sql.DB) *PlanRepository {
	return &PlanRepository{db: db}
}

func (r *PlanRepository) scanPlan(row interface{ Scan(dest ...any) error }) (*Plan, error) {
	plan := &Plan{}
	err := row.Scan(
		&plan.ID,
		&plan.Slug,
		&plan.Name,
		&plan.Price,
		&plan.MaxTunnels,
		&plan.MaxDomains,
		&plan.MaxCustomDomains,
		&plan.MaxTokens,
		&plan.MaxTunnelsPerToken,
		&plan.InspectorEnabled,
		&plan.IsPublic,
		&plan.IsRecommended,
	)
	return plan, err
}

const planColumns = `id, slug, name, price, max_tunnels, max_domains, max_custom_domains, max_tokens, max_tunnels_per_token, inspector_enabled, is_public, is_recommended`

// GetByID retrieves a plan by ID
func (r *PlanRepository) GetByID(id int64) (*Plan, error) {
	query := `SELECT ` + planColumns + ` FROM plans WHERE id = ?`
	plan, err := r.scanPlan(r.db.QueryRow(query, id))
	if err != nil {
		return nil, notFoundOrError(err, ErrPlanNotFound, "get plan by id")
	}
	return plan, nil
}

// GetBySlug retrieves a plan by slug
func (r *PlanRepository) GetBySlug(slug string) (*Plan, error) {
	query := `SELECT ` + planColumns + ` FROM plans WHERE slug = ?`
	plan, err := r.scanPlan(r.db.QueryRow(query, slug))
	if err != nil {
		return nil, notFoundOrError(err, ErrPlanNotFound, "get plan by slug")
	}
	return plan, nil
}

// List returns all plans
func (r *PlanRepository) List() ([]*Plan, error) {
	query := `SELECT ` + planColumns + ` FROM plans ORDER BY price ASC, id ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	var plans []*Plan
	for rows.Next() {
		plan, err := r.scanPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// Create creates a new plan
func (r *PlanRepository) Create(plan *Plan) error {
	query := `
		INSERT INTO plans (slug, name, price, max_tunnels, max_domains, max_custom_domains, max_tokens, max_tunnels_per_token, inspector_enabled, is_public, is_recommended)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query,
		plan.Slug,
		plan.Name,
		plan.Price,
		plan.MaxTunnels,
		plan.MaxDomains,
		plan.MaxCustomDomains,
		plan.MaxTokens,
		plan.MaxTunnelsPerToken,
		plan.InspectorEnabled,
		plan.IsPublic,
		plan.IsRecommended,
	)
	if err != nil {
		return fmt.Errorf("create plan: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	plan.ID = id
	return nil
}

// Update updates an existing plan
func (r *PlanRepository) Update(plan *Plan) error {
	query := `
		UPDATE plans SET slug = ?, name = ?, price = ?, max_tunnels = ?, max_domains = ?,
		max_custom_domains = ?, max_tokens = ?, max_tunnels_per_token = ?, inspector_enabled = ?,
		is_public = ?, is_recommended = ?
		WHERE id = ?
	`
	result, err := r.db.Exec(query,
		plan.Slug,
		plan.Name,
		plan.Price,
		plan.MaxTunnels,
		plan.MaxDomains,
		plan.MaxCustomDomains,
		plan.MaxTokens,
		plan.MaxTunnelsPerToken,
		plan.InspectorEnabled,
		plan.IsPublic,
		plan.IsRecommended,
		plan.ID,
	)
	if err != nil {
		return fmt.Errorf("update plan: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrPlanNotFound
	}
	return nil
}

// Delete deletes a plan by ID, returning an error if users are assigned to it
func (r *PlanRepository) Delete(id int64) error {
	count, err := r.CountUsers(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrPlanHasUsers
	}

	query := `DELETE FROM plans WHERE id = ?`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete plan: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrPlanNotFound
	}
	return nil
}

// GetDefault returns the default (free) plan
func (r *PlanRepository) GetDefault() (*Plan, error) {
	return r.GetBySlug("free")
}

// CountUsers returns the number of users assigned to a plan
func (r *PlanRepository) CountUsers(planID int64) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE plan_id = ?`
	var count int
	if err := r.db.QueryRow(query, planID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count plan users: %w", err)
	}
	return count, nil
}

// ListPublic returns all public plans (visible on landing page)
func (r *PlanRepository) ListPublic() ([]*Plan, error) {
	query := `SELECT ` + planColumns + ` FROM plans WHERE is_public = 1 ORDER BY price ASC, id ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("list public plans: %w", err)
	}
	defer rows.Close()

	var plans []*Plan
	for rows.Next() {
		plan, err := r.scanPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// ListAll returns all plans with pagination
func (r *PlanRepository) ListAll(limit, offset int) ([]*Plan, int, error) {
	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM plans").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count plans: %w", err)
	}

	query := `SELECT ` + planColumns + ` FROM plans ORDER BY price ASC, id ASC LIMIT ? OFFSET ?`
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list all plans: %w", err)
	}
	defer rows.Close()

	var plans []*Plan
	for rows.Next() {
		plan, err := r.scanPlan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan plan: %w", err)
		}
		plans = append(plans, plan)
	}
	return plans, total, nil
}
