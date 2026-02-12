package server

import (
	"testing"

	"github.com/mephistofox/fxtunnel/internal/database"
)

func TestBuildCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		plan     *database.Plan
		wantNil  bool
		wantInsp bool
	}{
		{"nil plan", nil, true, false},
		{"free plan", &database.Plan{InspectorEnabled: false}, false, false},
		{"pro plan", &database.Plan{InspectorEnabled: true}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := buildCapabilities(tt.plan)
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
