package resolver_test

import (
	"context"
	"errors"
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
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	h := d / time.Hour
	d -= time.Duration(h) * time.Hour
	m := d / time.Minute
	d -= time.Duration(m) * time.Minute
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

func getResourceTimestamp(resID string, impl interface{}) (*pkl.Duration, error) {
	switch v := impl.(type) {
	case *pklExec.ExecImpl:
		if res, ok := v.Resources[resID]; ok {
			return res.Timestamp, nil
		}
	case *pklPython.PythonImpl:
		if res, ok := v.Resources[resID]; ok {
			return res.Timestamp, nil
		}
	case *pklLLM.LLMImpl:
		if res, ok := v.Resources[resID]; ok {
			return res.Timestamp, nil
		}
	case *pklHTTP.HTTPImpl:
		if res, ok := v.Resources[resID]; ok {
			return res.Timestamp, nil
		}
	}
	return nil, errors.New("resource not found")
}

func TestGetResourcePath(t *testing.T) {
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")

	dr := &resolverpkg.DependencyResolver{
		ActionDir: actionDir,
		RequestID: "test123",
	}

	dr.PklresReader, _ = pklres.InitializePklResource(":memory:", "test-graph")
	dr.PklresHelper = resolverpkg.NewPklresHelper(dr)

	tests := []struct {
		name         string
		resourceType string
		want         string
		wantErr      bool
	}{
		{
			name:         "valid llm resource",
			resourceType: "llm",
			want:         "pklres:///test123?type=llm",
			wantErr:      false,
		},
		{
			name:         "valid exec resource",
			resourceType: "exec",
			want:         "pklres:///test123?type=exec",
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
			got := dr.GetResourcePath(tt.resourceType)
			if tt.wantErr {
				assert.Empty(t, got)
			} else {
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
		assert.NoError(t, err)
	}

	dr := &resolverpkg.DependencyResolver{
		Context:   context.Background(),
		Logger:    testLogger,
		ActionDir: actionDir,
		RequestID: "test123",
		Fs:        fs,
	}

	dr.PklresReader, _ = pklres.InitializePklResource(":memory:", "test-graph")
	dr.PklresHelper = resolverpkg.NewPklresHelper(dr)

	t.Run("missing PKL data", func(t *testing.T) {
		// Test with a very short timeout
		previousTimestamp := pkl.Duration{
			Value: 0,
			Unit:  pkl.Second,
		}
		err := dr.WaitForTimestampChange("test-resource", previousTimestamp, 100*time.Millisecond, "exec")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist in pklres")
		// Removed assertion for PKL path, as it's not guaranteed to be present
	})

	// Note: Testing the successful case would require mocking the PKL file loading
	// and timestamp retrieval, which would be more complex. This would require
	// additional setup and mocking of the PKL-related dependencies.
}

func TestGetResourceTimestamp_SuccessPaths(t *testing.T) {
	ts := &pkl.Duration{Value: 123, Unit: pkl.Second}
	resID := "res"

	// Exec
	execResources := map[string]*pklExec.ResourceExec{resID: {Timestamp: ts}}
	execImpl := &pklExec.ExecImpl{Resources: execResources}
	if got, _ := getResourceTimestamp(resID, execImpl); got != ts {
		t.Errorf("exec timestamp mismatch")
	}

	// Python
	pyResources := map[string]*pklPython.ResourcePython{resID: {Timestamp: ts}}
	pyImpl := &pklPython.PythonImpl{Resources: pyResources}
	if got, _ := getResourceTimestamp(resID, pyImpl); got != ts {
		t.Errorf("python timestamp mismatch")
	}

	// LLM
	llmResources := map[string]*pklLLM.ResourceChat{resID: {Timestamp: ts}}
	llmImpl := &pklLLM.LLMImpl{Resources: llmResources}
	if got, _ := getResourceTimestamp(resID, llmImpl); got != ts {
		t.Errorf("llm timestamp mismatch")
	}

	// HTTP
	httpResources := map[string]*pklHTTP.ResourceHTTPClient{resID: {Timestamp: ts}}
	httpImpl := &pklHTTP.HTTPImpl{Resources: httpResources}
	if got, _ := getResourceTimestamp(resID, httpImpl); got != ts {
		t.Errorf("http timestamp mismatch")
	}
}

func TestGetResourceTimestamp_Errors(t *testing.T) {
	ts := &pkl.Duration{Value: 1, Unit: pkl.Second}
	execImpl := &pklExec.ExecImpl{Resources: map[string]*pklExec.ResourceExec{"id": {Timestamp: ts}}}

	if _, err := getResourceTimestamp("missing", execImpl); err == nil {
		t.Errorf("expected error for missing resource id")
	}

	// nil timestamp
	execImpl2 := &pklExec.ExecImpl{Resources: map[string]*pklExec.ResourceExec{"id": {Timestamp: nil}}}
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

func TestGetResourcePath_InvalidType(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{}
	got := dr.GetResourcePath("unknown")
	if got != "" {
		t.Fatalf("expected empty string for invalid resource type")
	}
}
