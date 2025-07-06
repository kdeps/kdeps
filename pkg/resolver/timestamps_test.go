package resolver

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestGetResourceFilePath(t *testing.T) {
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")

	dr := &DependencyResolver{
		ActionDir: actionDir,
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
			want:         filepath.Join(actionDir, "llm", "test123__llm_output.pkl"),
			wantErr:      false,
		},
		{
			name:         "valid exec resource",
			resourceType: "exec",
			want:         filepath.Join(actionDir, "exec", "test123__exec_output.pkl"),
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
		assert.NoError(t, err)
	}

	dr := &DependencyResolver{
		Context:   context.Background(),
		Logger:    testLogger,
		ActionDir: actionDir,
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

func TestGetResourceTimestamp_SuccessPaths(t *testing.T) {
	ts := &pkl.Duration{Value: 123, Unit: pkl.Second}
	resID := "res"

	// Exec
	execImpl := &pklExec.ExecImpl{Resources: &map[string]*pklExec.ResourceExec{resID: {Timestamp: ts}}}
	if got, _ := getResourceTimestamp(resID, execImpl); got != ts {
		t.Errorf("exec timestamp mismatch")
	}

	// Python
	pyImpl := &pklPython.PythonImpl{Resources: &map[string]*pklPython.ResourcePython{resID: {Timestamp: ts}}}
	if got, _ := getResourceTimestamp(resID, pyImpl); got != ts {
		t.Errorf("python timestamp mismatch")
	}

	// LLM
	llmImpl := &pklLLM.LLMImpl{Resources: &map[string]*pklLLM.ResourceChat{resID: {Timestamp: ts}}}
	if got, _ := getResourceTimestamp(resID, llmImpl); got != ts {
		t.Errorf("llm timestamp mismatch")
	}

	// HTTP
	httpImpl := &pklHTTP.HTTPImpl{Resources: &map[string]*pklHTTP.ResourceHTTPClient{resID: {Timestamp: ts}}}
	if got, _ := getResourceTimestamp(resID, httpImpl); got != ts {
		t.Errorf("http timestamp mismatch")
	}
}

func TestGetResourceTimestamp_Errors(t *testing.T) {
	ts := &pkl.Duration{Value: 1, Unit: pkl.Second}
	execImpl := &pklExec.ExecImpl{Resources: &map[string]*pklExec.ResourceExec{"id": {Timestamp: ts}}}

	if _, err := getResourceTimestamp("missing", execImpl); err == nil {
		t.Errorf("expected error for missing resource id")
	}

	// nil timestamp
	execImpl2 := &pklExec.ExecImpl{Resources: &map[string]*pklExec.ResourceExec{"id": {Timestamp: nil}}}
	if _, err := getResourceTimestamp("id", execImpl2); err == nil {
		t.Errorf("expected error for nil timestamp")
	}

	// unknown type
	if _, err := getResourceTimestamp("id", 42); err == nil {
		t.Errorf("expected error for unknown type")
	}
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

func TestGetResourceFilePath_InvalidType(t *testing.T) {
	dr := &DependencyResolver{}
	_, err := dr.getResourceFilePath("unknown")
	if err == nil {
		t.Fatalf("expected error for invalid resource type")
	}
}
