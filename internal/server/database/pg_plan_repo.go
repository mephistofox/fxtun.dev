package database

import (
	"context"
	"fmt"

	"github.com/mephistofox/fxtunnel/internal/server/database/sqlc"
)

// PlanRepository handles plan database operations using PostgreSQL via sqlc.
type PlanRepository struct {
	q *sqlc.Queries
}

// sqlcPlanToDomain converts a sqlc.Plan to a domain Plan.
func sqlcPlanToDomain(p sqlc.Plan) *Plan {
	return &Plan{
		ID:                 p.ID,
		Slug:               p.Slug,
		Name:               p.Name,
		Price:              p.Price,
		MaxTunnels:         int(p.MaxTunnels),
		MaxDomains:         int(p.MaxDomains),
		MaxCustomDomains:   int(p.MaxCustomDomains),
		MaxTokens:          int(p.MaxTokens),
		MaxTunnelsPerToken: int(p.MaxTunnelsPerToken),
		BandwidthMbps:      int(p.BandwidthMbps),
		InspectorEnabled:   p.InspectorEnabled,
		IsPublic:           p.IsPublic,
		IsRecommended:      p.IsRecommended,
		RateLimitTCP:       int(p.RateLimitTcp),
		RateLimitUDP:       int(p.RateLimitUdp),
		RateLimitHTTP:      int(p.RateLimitHttp),
		CreemProductID:     p.CreemProductID,
		MaxDataSessions:    int(p.MaxDataSessions),
		UDPEnabled:         p.UdpEnabled,
	}
}

// GetByID retrieves a plan by ID.
func (r *PlanRepository) GetByID(id int64) (*Plan, error) {
	ctx := context.Background()
	p, err := r.q.GetPlanByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan by id: %w", err)
	}
	return sqlcPlanToDomain(p), nil
}

// GetBySlug retrieves a plan by slug.
func (r *PlanRepository) GetBySlug(slug string) (*Plan, error) {
	ctx := context.Background()
	p, err := r.q.GetPlanBySlug(ctx, slug)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan by slug: %w", err)
	}
	return sqlcPlanToDomain(p), nil
}

// GetDefault returns the default (free) plan.
func (r *PlanRepository) GetDefault() (*Plan, error) {
	ctx := context.Background()
	p, err := r.q.GetDefaultPlan(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get default plan: %w", err)
	}
	return sqlcPlanToDomain(p), nil
}

// Create creates a new plan.
func (r *PlanRepository) Create(plan *Plan) error {
	ctx := context.Background()
	id, err := r.q.CreatePlan(ctx, sqlc.CreatePlanParams{
		Slug:               plan.Slug,
		Name:               plan.Name,
		Price:              plan.Price,
		MaxTunnels:         int32(plan.MaxTunnels),
		MaxDomains:         int32(plan.MaxDomains),
		MaxCustomDomains:   int32(plan.MaxCustomDomains),
		MaxTokens:          int32(plan.MaxTokens),
		MaxTunnelsPerToken: int32(plan.MaxTunnelsPerToken),
		InspectorEnabled:   plan.InspectorEnabled,
		IsPublic:           plan.IsPublic,
		IsRecommended:      plan.IsRecommended,
		BandwidthMbps:      int32(plan.BandwidthMbps),
		RateLimitTcp:       int32(plan.RateLimitTCP),
		RateLimitUdp:       int32(plan.RateLimitUDP),
		RateLimitHttp:      int32(plan.RateLimitHTTP),
		CreemProductID:     plan.CreemProductID,
		MaxDataSessions:    int32(plan.MaxDataSessions),
		UdpEnabled:         plan.UDPEnabled,
	})
	if err != nil {
		return fmt.Errorf("create plan: %w", err)
	}
	plan.ID = id
	return nil
}

// Update updates an existing plan.
func (r *PlanRepository) Update(plan *Plan) error {
	ctx := context.Background()
	err := r.q.UpdatePlan(ctx, sqlc.UpdatePlanParams{
		ID:                 plan.ID,
		Name:               plan.Name,
		Price:              plan.Price,
		MaxTunnels:         int32(plan.MaxTunnels),
		MaxDomains:         int32(plan.MaxDomains),
		MaxCustomDomains:   int32(plan.MaxCustomDomains),
		MaxTokens:          int32(plan.MaxTokens),
		MaxTunnelsPerToken: int32(plan.MaxTunnelsPerToken),
		InspectorEnabled:   plan.InspectorEnabled,
		IsPublic:           plan.IsPublic,
		IsRecommended:      plan.IsRecommended,
		BandwidthMbps:      int32(plan.BandwidthMbps),
		RateLimitTcp:       int32(plan.RateLimitTCP),
		RateLimitUdp:       int32(plan.RateLimitUDP),
		RateLimitHttp:      int32(plan.RateLimitHTTP),
		CreemProductID:     plan.CreemProductID,
		MaxDataSessions:    int32(plan.MaxDataSessions),
		UdpEnabled:         plan.UDPEnabled,
	})
	if err != nil {
		return fmt.Errorf("update plan: %w", err)
	}
	return nil
}

// Delete deletes a plan by ID, returning an error if users are assigned to it.
func (r *PlanRepository) Delete(id int64) error {
	count, err := r.CountUsers(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrPlanHasUsers
	}

	ctx := context.Background()
	err = r.q.DeletePlan(ctx, id)
	if err != nil {
		return fmt.Errorf("delete plan: %w", err)
	}
	return nil
}

// List returns all plans.
func (r *PlanRepository) List() ([]*Plan, error) {
	ctx := context.Background()
	rows, err := r.q.ListPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	plans := make([]*Plan, 0, len(rows))
	for _, p := range rows {
		plans = append(plans, sqlcPlanToDomain(p))
	}
	return plans, nil
}

// ListPublic returns all public plans (visible on landing page).
func (r *PlanRepository) ListPublic() ([]*Plan, error) {
	ctx := context.Background()
	rows, err := r.q.ListPublicPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("list public plans: %w", err)
	}
	plans := make([]*Plan, 0, len(rows))
	for _, p := range rows {
		plans = append(plans, sqlcPlanToDomain(p))
	}
	return plans, nil
}

// ListAll returns all plans with pagination.
func (r *PlanRepository) ListAll(limit, offset int) ([]*Plan, int, error) {
	ctx := context.Background()
	total, err := r.q.CountAllPlans(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count plans: %w", err)
	}

	rows, err := r.q.ListAllPlans(ctx, sqlc.ListAllPlansParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list all plans: %w", err)
	}
	plans := make([]*Plan, 0, len(rows))
	for _, p := range rows {
		plans = append(plans, sqlcPlanToDomain(p))
	}
	return plans, int(total), nil
}

// CountUsers returns the number of users assigned to a plan.
func (r *PlanRepository) CountUsers(planID int64) (int, error) {
	ctx := context.Background()
	count, err := r.q.CountPlanUsers(ctx, int64ToPgint8(planID))
	if err != nil {
		return 0, fmt.Errorf("count plan users: %w", err)
	}
	return int(count), nil
}
