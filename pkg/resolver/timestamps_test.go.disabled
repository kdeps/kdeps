package resolver_test

import (
	"context"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetResourceFilePath(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	dr := &resolver.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		ActionDir: "/test/actions",
		RequestID: "test-request",
	}

	// Test valid resource types
	testCases := []struct {
		resourceType string
		expectedPath string
	}{
		{"llm", "/test/actions/llm/test-request__llm_output.pkl"},
		{"client", "/test/actions/client/test-request__client_output.pkl"},
		{"exec", "/test/actions/exec/test-request__exec_output.pkl"},
		{"python", "/test/actions/python/test-request__python_output.pkl"},
	}

	for _, tc := range testCases {
		t.Run(tc.resourceType, func(t *testing.T) {
			path, err := dr.GetResourceFilePath(tc.resourceType)
			if err != nil {
				t.Errorf("unexpected error for %s: %v", tc.resourceType, err)
			}
			if path != tc.expectedPath {
				t.Errorf("expected path %s, got %s", tc.expectedPath, path)
			}
		})
	}

	// Test invalid resource type
	_, err := dr.GetResourceFilePath("invalid")
	if err == nil {
		t.Error("expected error for invalid resource type")
	}
}

func TestLoadPKLFile(t *testing.T) {
	fs := afero.NewOsFs()
	dir, err := afero.TempDir(fs, "", "load-pkl-file")
	require.NoError(t, err)
	defer fs.RemoveAll(dir)

	dr := &resolver.DependencyResolver{
		Fs:        fs,
		ActionDir: dir,
		RequestID: "test-request",
		Context:   context.Background(),
	}

	t.Run("UnsupportedResourceType", func(t *testing.T) {
		// Test through the public GetCurrentTimestamp method which calls loadPKLFile
		_, err := dr.GetCurrentTimestamp("test-id", "unsupported")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid resourceType unsupported provided")
	})
}

func TestGetResourceTimestamp(t *testing.T) {
	t.Run("UnknownPKLResultType", func(t *testing.T) {
		_, err := resolver.GetResourceTimestamp("test-id", "not-a-pkl-result")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown PKL result type")
	})
}

func TestGetCurrentTimestamp(t *testing.T) {
	t.Run("ValidResourceType", func(t *testing.T) {
		// Create a mock resolver with test data
		resolver := &resolver.DependencyResolver{
			ActionDir: "/test/action",
			RequestID: "test-request",
			Context:   context.Background(),
			Logger:    logging.NewTestLogger(),
		}

		// Test with a valid resource type
		_, err := resolver.GetCurrentTimestamp("test-resource", "llm")

		// Since we don't have actual PKL files, we expect an error
		// but the function should have been called and coverage increased
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to load llm PKL file")
	})

	t.Run("InvalidResourceType", func(t *testing.T) {
		resolver := &resolver.DependencyResolver{
			ActionDir: "/test/action",
			RequestID: "test-request",
			Context:   context.Background(),
			Logger:    logging.NewTestLogger(),
		}

		// Test with an invalid resource type
		timestamp, err := resolver.GetCurrentTimestamp("test-resource", "invalid")

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid resourceType invalid provided")
		require.Equal(t, pkl.Duration{}, timestamp)
	})

	t.Run("GetResourceFilePathError", func(t *testing.T) {
		resolver := &resolver.DependencyResolver{
			ActionDir: "/test/action",
			RequestID: "test-request",
			Context:   context.Background(),
			Logger:    logging.NewTestLogger(),
		}

		// Test with an invalid resource type to trigger GetResourceFilePath error
		timestamp, err := resolver.GetCurrentTimestamp("test-resource", "invalid")

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid resourceType invalid provided")
		require.Equal(t, pkl.Duration{}, timestamp)
	})
}

func TestFormatDuration(t *testing.T) {
	// Test seconds only
	result := resolver.FormatDuration(30 * time.Second)
	if result != "30s" {
		t.Errorf("expected '30s', got %s", result)
	}

	// Test minutes and seconds
	result = resolver.FormatDuration(90 * time.Second)
	if result != "1m 30s" {
		t.Errorf("expected '1m 30s', got %s", result)
	}

	// Test hours, minutes, and seconds
	result = resolver.FormatDuration(7325 * time.Second) // 2h 2m 5s
	if result != "2h 2m 5s" {
		t.Errorf("expected '2h 2m 5s', got %s", result)
	}

	// Test zero duration
	result = resolver.FormatDuration(0)
	if result != "0s" {
		t.Errorf("expected '0s', got %s", result)
	}

	// Test sub-second duration
	result = resolver.FormatDuration(500 * time.Millisecond)
	if result != "0s" {
		t.Errorf("expected '0s', got %s", result)
	}
}

func TestWaitForTimestampChange(t *testing.T) {
	fs := afero.NewOsFs()
	dir, err := afero.TempDir(fs, "", "wait-timestamp")
	require.NoError(t, err)
	defer fs.RemoveAll(dir)

	logger := logging.NewTestLogger()
	dr := &resolver.DependencyResolver{
		Fs:        fs,
		ActionDir: dir,
		RequestID: "test-request",
		Context:   context.Background(),
		Logger:    logger,
	}

	t.Run("InvalidResourceType", func(t *testing.T) {
		previousTimestamp := pkl.Duration{}
		timeout := 1 * time.Second

		err := dr.WaitForTimestampChange("test-id", previousTimestamp, timeout, "invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid resourceType")
	})
}

func TestGetResourceTimestamp_SuccessPaths(t *testing.T) {
	ts := &pkl.Duration{Value: 123, Unit: pkl.Second}
	resID := "res"

	// Exec
	execImpl := &pklExec.ExecImpl{Resources: &map[string]*pklExec.ResourceExec{resID: {Timestamp: ts}}}
	if got, _ := resolver.GetResourceTimestamp(resID, execImpl); got != ts {
		t.Errorf("exec timestamp mismatch")
	}

	// Python
	pyImpl := &pklPython.PythonImpl{Resources: &map[string]*pklPython.ResourcePython{resID: {Timestamp: ts}}}
	if got, _ := resolver.GetResourceTimestamp(resID, pyImpl); got != ts {
		t.Errorf("python timestamp mismatch")
	}

	// LLM
	llmImpl := &pklLLM.LLMImpl{Resources: &map[string]*pklLLM.ResourceChat{resID: {Timestamp: ts}}}
	if got, _ := resolver.GetResourceTimestamp(resID, llmImpl); got != ts {
		t.Errorf("llm timestamp mismatch")
	}

	// HTTP
	httpImpl := &pklHTTP.HTTPImpl{Resources: &map[string]*pklHTTP.ResourceHTTPClient{resID: {Timestamp: ts}}}
	if got, _ := resolver.GetResourceTimestamp(resID, httpImpl); got != ts {
		t.Errorf("http timestamp mismatch")
	}
}

func TestGetResourceTimestamp_Errors(t *testing.T) {
	ts := &pkl.Duration{Value: 1, Unit: pkl.Second}
	execImpl := &pklExec.ExecImpl{Resources: &map[string]*pklExec.ResourceExec{"id": {Timestamp: ts}}}

	if _, err := resolver.GetResourceTimestamp("missing", execImpl); err == nil {
		t.Errorf("expected error for missing resource id")
	}

	// nil timestamp
	execImpl2 := &pklExec.ExecImpl{Resources: &map[string]*pklExec.ResourceExec{"id": {Timestamp: nil}}}
	if _, err := resolver.GetResourceTimestamp("id", execImpl2); err == nil {
		t.Errorf("expected error for nil timestamp")
	}

	// unknown type
	if _, err := resolver.GetResourceTimestamp("id", 42); err == nil {
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
		got := resolver.FormatDuration(c.d)
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
		got := resolver.FormatDuration(c.dur)
		if got != c.want {
			t.Errorf("formatDuration(%v) = %s, want %s", c.dur, got, c.want)
		}
	}
}

func TestGetResourceFilePath_InvalidType(t *testing.T) {
	dr := &resolver.DependencyResolver{}
	_, err := dr.GetResourceFilePath("unknown")
	if err == nil {
		t.Fatalf("expected error for invalid resource type")
	}
}

func TestGetCurrentTimestamp_ErrorCases(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	dr := &resolver.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		ActionDir: "/test/actions",
		RequestID: "test-request",
	}

	// Test with invalid resource type
	_, err := dr.GetCurrentTimestamp("test-id", "invalid")
	if err == nil {
		t.Error("expected error for invalid resource type")
	}

	// Test with non-existent file
	_, err = dr.GetCurrentTimestamp("test-id", "llm")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestWaitForTimestampChange_InvalidResourceType(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	dr := &resolver.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		ActionDir: "/test/actions",
		RequestID: "test-request",
	}

	previousTimestamp := pkl.Duration{Value: 1, Unit: pkl.Second}
	timeout := 1 * time.Second

	err := dr.WaitForTimestampChange("test-id", previousTimestamp, timeout, "invalid")
	if err == nil {
		t.Error("expected error for invalid resource type")
	}
}

func TestGetResourceTimestamp_EdgeCases(t *testing.T) {
	// Test with nil interface
	_, err := resolver.GetResourceTimestamp("test-id", nil)
	if err == nil {
		t.Error("expected error for nil interface")
	}
	if err != nil && err.Error() != "unknown PKL result type" {
		t.Errorf("unexpected error message: %v", err)
	}

	// Test with unsupported type
	_, err = resolver.GetResourceTimestamp("test-id", "string")
	if err == nil {
		t.Error("expected error for unsupported type")
	}
	if err != nil && err.Error() != "unknown PKL result type" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestGetCurrentTimestamp_FocusedCoverage ensures GetCurrentTimestamp gets coverage
func TestGetCurrentTimestamp_FocusedCoverage(t *testing.T) {
	dr := &resolver.DependencyResolver{
		ActionDir: "/test/actions",
		RequestID: "test-req",
		Context:   context.Background(),
		Logger:    logging.NewTestLogger(),
	}

	// Test 1: Invalid resource type to trigger GetResourceFilePath error
	_, err := dr.GetCurrentTimestamp("resource-id", "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid resourceType")

	// Test 2: Valid resource types to trigger loadPKLFile error
	validTypes := []string{"llm", "client", "exec", "python"}
	for _, resourceType := range validTypes {
		_, err := dr.GetCurrentTimestamp("resource-id", resourceType)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load")
	}
}

// TestProcessResourceStep_GetCurrentTimestamp tests GetCurrentTimestamp through ProcessResourceStep
func TestProcessResourceStep_GetCurrentTimestamp(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	dr := &resolver.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		ActionDir: "/test/actions",
		RequestID: "test-req",
	}

	// Initialize injectable functions to point to actual methods (like NewGraphResolver does)
	dr.GetCurrentTimestampFn = dr.GetCurrentTimestamp
	dr.WaitForTimestampChangeFn = dr.WaitForTimestampChange

	t.Run("ProcessResourceStep_InvalidResourceType", func(t *testing.T) {
		// This will call GetCurrentTimestampFn which points to GetCurrentTimestamp method
		err := dr.ProcessResourceStep("test-resource", "invalid-type", nil, func() error {
			return nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid-type error")
		assert.Contains(t, err.Error(), "invalid resourceType")
	})

	t.Run("ProcessResourceStep_ValidResourceType", func(t *testing.T) {
		// This will call GetCurrentTimestampFn which points to GetCurrentTimestamp method
		err := dr.ProcessResourceStep("test-resource", "llm", nil, func() error {
			return nil
		})

		assert.Error(t, err) // Expected to fail since no PKL file exists
		assert.Contains(t, err.Error(), "llm error")
		assert.Contains(t, err.Error(), "failed to load")
	})
}
