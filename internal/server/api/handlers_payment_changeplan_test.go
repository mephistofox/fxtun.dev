package api

import (
	"testing"

	"github.com/mephistofox/fxtun.dev/internal/server/database"
)

// TestIsPlanUpgrade guards the rule behind handleChangePlan: scheduling a plan
// change is free and takes no payment, so it must reject upgrades to a pricier
// plan (those have to go through paid checkout). Only downgrades and lateral
// moves may be scheduled.
func TestIsPlanUpgrade(t *testing.T) {
	base := &database.Plan{Slug: "base", Price: 2.50}
	pro := &database.Plan{Slug: "pro", Price: 5.00}
	business := &database.Plan{Slug: "business", Price: 7.50}

	cases := []struct {
		name    string
		current *database.Plan
		next    *database.Plan
		want    bool
	}{
		{"upgrade base->business", base, business, true},
		{"upgrade base->pro", base, pro, true},
		{"downgrade business->base", business, base, false},
		{"downgrade pro->base", pro, base, false},
		{"lateral same price", pro, &database.Plan{Slug: "pro2", Price: 5.00}, false},
		{"nil current", nil, business, false},
		{"nil next", base, nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPlanUpgrade(tc.current, tc.next); got != tc.want {
				t.Errorf("isPlanUpgrade(%v, %v) = %v, want %v", tc.current, tc.next, got, tc.want)
			}
		})
	}
}
