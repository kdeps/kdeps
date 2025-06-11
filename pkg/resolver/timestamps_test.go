package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestGetResourceFilePath(t *testing.T) {
	dr := &DependencyResolver{
		ActionDir: "/test/action",
		RequestID: "test123",
	}

	tests := []struct {
		name         string
		resourceType string
		want         string
		wantErr      bool
	}{
		{
			name:         "valid llm resource",
			resourceType: "llm",
			want:         "/test/action/llm/test123__llm_output.pkl",
			wantErr:      false,
		},
		{
			name:         "valid exec resource",
			resourceType: "exec",
			want:         "/test/action/exec/test123__exec_output.pkl",
			wantErr:      false,
		},
		{
			name:         "invalid resource type",
			resourceType: "invalid",
			want:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dr.getResourceFilePath(tt.resourceType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "hours minutes seconds",
			duration: 2*time.Hour + 30*time.Minute + 15*time.Second,
			want:     "2h 30m 15s",
		},
		{
			name:     "minutes seconds",
			duration: 45*time.Minute + 30*time.Second,
			want:     "45m 30s",
		},
		{
			name:     "seconds only",
			duration: 30 * time.Second,
			want:     "30s",
		},
		{
			name:     "zero duration",
			duration: 0,
			want:     "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWaitForTimestampChange(t *testing.T) {
	// Create a mock file system
	fs := afero.NewMemMapFs()
	testLogger := logging.NewTestLogger()

	// Create necessary directories
	dirs := []string{
		"/test/action/exec",
		"/test/action/llm",
		"/test/action/python",
		"/test/action/client",
	}
	for _, dir := range dirs {
		err := fs.MkdirAll(dir, 0755)
		assert.NoError(t, err)
	}

	dr := &DependencyResolver{
		Context:   context.Background(),
		Logger:    testLogger,
		ActionDir: "/test/action",
		RequestID: "test123",
		Fs:        fs,
	}

	t.Run("missing PKL file", func(t *testing.T) {
		// Test with a very short timeout
		previousTimestamp := pkl.Duration{
			Value: 0,
			Unit:  pkl.Second,
		}
		err := dr.WaitForTimestampChange("test-resource", previousTimestamp, 100*time.Millisecond, "exec")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot find module")
		assert.Contains(t, err.Error(), "test123__exec_output.pkl")
	})

	// Note: Testing the successful case would require mocking the PKL file loading
	// and timestamp retrieval, which would be more complex. This would require
	// additional setup and mocking of the PKL-related dependencies.
}
