package enforcer

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	logger := logging.NewTestLogger()

	tests := []struct {
		name     string
		v1, v2   string
		expected int
		wantErr  bool
	}{
		{"equal versions", "1.2.3", "1.2.3", 0, false},
		{"v1 greater patch", "1.2.4", "1.2.3", 1, false},
		{"v1 greater minor", "1.3.0", "1.2.9", 1, false},
		{"v1 less major", "1.2.3", "2.0.0", -1, false},
		{"different length v1 longer", "1.2.3.1", "1.2.3", 1, false},
		{"different length v2 longer", "1.2", "1.2.0.1", -1, false},
		{"invalid v1 format", "1.2.x", "1.2.0", 0, true},
		{"invalid v2 format", "1.2.0", "1.2.x", 0, true},
	}

	for _, tc := range tests {
		tc := tc // capture
		t.Run(tc.name, func(t *testing.T) {
			result, err := compareVersions(tc.v1, tc.v2, logger)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}
