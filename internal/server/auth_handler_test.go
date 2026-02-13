package server

import (
	"testing"

	"github.com/mephistofox/fxtun.dev/internal/database"
)

func TestBuildCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		plan     *database.Plan
		isAdmin  bool
		wantNil  bool
		wantInsp bool
	}{
		{"nil plan non-admin", nil, false, true, false},
		{"nil plan admin", nil, true, false, true},
		{"free plan non-admin", &database.Plan{InspectorEnabled: false}, false, false, false},
		{"free plan admin", &database.Plan{InspectorEnabled: false}, true, false, true},
		{"pro plan non-admin", &database.Plan{InspectorEnabled: true}, false, false, true},
		{"pro plan admin", &database.Plan{InspectorEnabled: true}, true, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := buildCapabilities(tt.plan, tt.isAdmin)
			if tt.wantNil {
				if caps != nil {
					t.Error("expected nil capabilities")
				}
				return
			}
			if caps == nil {
				t.Fatal("expected non-nil capabilities")
			}
			if caps.InspectorEnabled != tt.wantInsp {
				t.Errorf("inspector_enabled = %v, want %v", caps.InspectorEnabled, tt.wantInsp)
			}
		})
	}
}
