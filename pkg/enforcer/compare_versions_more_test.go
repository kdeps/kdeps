package enforcer

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestCompareVersionsAdditional(t *testing.T) {
	logger := logging.NewTestLogger()
	tests := []struct {
		name   string
		v1, v2 string
		want   int
	}{
		{"equal", "1.2.3", "1.2.3", 0},
		{"v1< v2", "0.9", "1.0", -1},
		{"v1>v2", "2.0", "1.5", 1},
		{"different lengths", "1.2.3", "1.2", 1},
	}
	for _, tc := range tests {
		got, err := compareVersions(tc.v1, tc.v2, logger)
		assert.NoError(t, err)
		assert.Equal(t, tc.want, got, tc.name)
	}
}
