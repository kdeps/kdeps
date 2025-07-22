package resolver_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	pklres "github.com/kdeps/kdeps/pkg/pklres"
	resolverpkg "github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	var parts []string
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if s > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", s))
	}
	return strings.Join(parts, " ")
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
	// Initialize evaluator for this test
	evaluator.TestSetup(t)

	// Create a mock file system
	fs := afero.NewMemMapFs()
	testLogger := logging.NewTestLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")

	// Create necessary directories
	dirs := []string{
		filepath.Join(actionDir, "exec"),
		filepath.Join(actionDir, "llm"),
		filepath.Join(actionDir, "python"),
		filepath.Join(actionDir, "client"),
	}
	for _, dir := range dirs {
		err := fs.MkdirAll(dir, 0o755)
		require.NoError(t, err)
	}

	dr := &resolverpkg.DependencyResolver{
		Context:   context.Background(),
		Logger:    testLogger,
		ActionDir: actionDir,
		RequestID: "test123",
		Fs:        fs,
	}

	dr.PklresReader, _ = pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
	dr.PklresHelper = resolverpkg.NewPklresHelper(dr)

	t.Run("missing PKL data", func(t *testing.T) {
		// Test with a very short timeout
		// Use a timestamp close to current time so the default timestamp won't be greater
		previousTimestamp := pkl.Duration{
			Value: float64(time.Now().UnixNano()),
			Unit:  pkl.Nanosecond,
		}
		err := dr.WaitForTimestampChange("test-resource", previousTimestamp, 100*time.Millisecond, "exec")
		// Since GetCurrentTimestamp returns a default timestamp for missing resources,
		// WaitForTimestampChange will return nil immediately if the timestamp is >= previousTimestamp.
		require.NoError(t, err)
	})

	// Note: Testing the successful case would require mocking the PKL file loading
	// and timestamp retrieval, which would be more complex. This would require
	// additional setup and mocking of the PKL-related dependencies.
}

func TestFormatDuration_Simple(t *testing.T) {
	cases := []struct {
		d        time.Duration
		expected string
	}{
		{3 * time.Second, "3s"},
		{2*time.Minute + 5*time.Second, "2m 5s"},
		{1*time.Hour + 10*time.Minute + 30*time.Second, "1h 10m 30s"},
		{0, "0s"},
	}
	for _, c := range cases {
		got := formatDuration(c.d)
		if got != c.expected {
			t.Errorf("formatDuration(%v) = %q, want %q", c.d, got, c.expected)
		}
	}
}

func TestFormatDurationExtra(t *testing.T) {
	cases := []struct {
		dur  time.Duration
		want string
	}{
		{time.Second * 5, "5s"},
		{time.Minute*2 + time.Second*10, "2m 10s"},
		{time.Hour*1 + time.Minute*3 + time.Second*4, "1h 3m 4s"},
	}

	for _, c := range cases {
		got := formatDuration(c.dur)
		if got != c.want {
			t.Errorf("formatDuration(%v) = %s, want %s", c.dur, got, c.want)
		}
	}
}
