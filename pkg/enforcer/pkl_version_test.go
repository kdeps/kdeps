package enforcer

import (
	"context"
	"fmt"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
)

func TestEnforcePklVersionScenarios(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()
	schemaVer := schema.SchemaVersion(ctx)

	tests := []struct {
		name         string
		amendVersion string
	}{
		{"lower", "0.0.1"},
		{"equal", schemaVer},
		{"higher", "9.9.9"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			line := fmt.Sprintf("amends \"package://schema.kdeps.com/core@%s#/Kdeps.pkl\"", tc.amendVersion)
			if err := EnforcePklVersion(line, "dummy.pkl", schemaVer, ctx, logger); err != nil {
				t.Fatalf("unexpected error for version %s: %v", tc.amendVersion, err)
			}
		})
	}
}
