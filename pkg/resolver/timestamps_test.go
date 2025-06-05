package resolver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/schema/gen/exec"
	"github.com/kdeps/schema/gen/http"
	"github.com/kdeps/schema/gen/llm"
	"github.com/kdeps/schema/gen/python"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{time.Hour + 2*time.Minute + 3*time.Second, "1h 2m 3s"},
		{2*time.Minute + 5*time.Second, "2m 5s"},
		{45 * time.Second, "45s"},
		{0, "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetResourceFilePath(t *testing.T) {
	dr := &DependencyResolver{
		ActionDir: "/test/action",
		RequestID: "test123",
	}

	tests := []struct {
		name         string
		resourceType ResourceType
		wantErr      bool
		expectedPath string
	}{
		{
			name:         "valid llm type",
			resourceType: LLMResource,
			wantErr:      false,
			expectedPath: filepath.Join("/test/action", "llm", "test123__llm_output.pkl"),
		},
		{
			name:         "valid client type",
			resourceType: HTTPResource,
			wantErr:      false,
			expectedPath: filepath.Join("/test/action", "client", "test123__client_output.pkl"),
		},
		{
			name:         "valid exec type",
			resourceType: ExecResource,
			wantErr:      false,
			expectedPath: filepath.Join("/test/action", "exec", "test123__exec_output.pkl"),
		},
		{
			name:         "valid python type",
			resourceType: PythonResource,
			wantErr:      false,
			expectedPath: filepath.Join("/test/action", "python", "test123__python_output.pkl"),
		},
		{
			name:         "invalid type",
			resourceType: ResourceType("invalid"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := dr.GetResourceFilePath(tt.resourceType)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if path != tt.expectedPath {
					t.Errorf("expected path %q, got %q", tt.expectedPath, path)
				}
			}
		})
	}
}

func TestGetResourceTimestamp(t *testing.T) {
	// Create test resources
	execRes := &exec.ExecImpl{
		Resources: &map[string]*exec.ResourceExec{
			"test1": {
				Timestamp: &pkl.Duration{Value: 100, Unit: pkl.Second},
			},
		},
	}

	pythonRes := &python.PythonImpl{
		Resources: &map[string]*python.ResourcePython{
			"test2": {
				Timestamp: &pkl.Duration{Value: 200, Unit: pkl.Second},
			},
		},
	}

	llmRes := &llm.LLMImpl{
		Resources: &map[string]*llm.ResourceChat{
			"test3": {
				Timestamp: &pkl.Duration{Value: 300, Unit: pkl.Second},
			},
		},
	}

	httpRes := &http.HTTPImpl{
		Resources: &map[string]*http.ResourceHTTPClient{
			"test4": {
				Timestamp: &pkl.Duration{Value: 400, Unit: pkl.Second},
			},
		},
	}

	tests := []struct {
		name        string
		resourceID  string
		pklRes      interface{}
		wantErr     bool
		expectedDur pkl.Duration
	}{
		{
			name:        "valid exec resource",
			resourceID:  "test1",
			pklRes:      execRes,
			wantErr:     false,
			expectedDur: pkl.Duration{Value: 100, Unit: pkl.Second},
		},
		{
			name:        "valid python resource",
			resourceID:  "test2",
			pklRes:      pythonRes,
			wantErr:     false,
			expectedDur: pkl.Duration{Value: 200, Unit: pkl.Second},
		},
		{
			name:        "valid llm resource",
			resourceID:  "test3",
			pklRes:      llmRes,
			wantErr:     false,
			expectedDur: pkl.Duration{Value: 300, Unit: pkl.Second},
		},
		{
			name:        "valid http resource",
			resourceID:  "test4",
			pklRes:      httpRes,
			wantErr:     false,
			expectedDur: pkl.Duration{Value: 400, Unit: pkl.Second},
		},
		{
			name:       "non-existent resource",
			resourceID: "nonexistent",
			pklRes:     execRes,
			wantErr:    true,
		},
		{
			name:       "invalid resource type",
			resourceID: "test1",
			pklRes:     "invalid",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timestamp, err := getResourceTimestamp(tt.resourceID, tt.pklRes)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if timestamp.Value != tt.expectedDur.Value || timestamp.Unit != tt.expectedDur.Unit {
					t.Errorf("expected duration %v, got %v", tt.expectedDur, timestamp)
				}
			}
		})
	}
}

func TestGetCurrentTimestamp(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kdeps-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test directories
	resourceTypes := []ResourceType{LLMResource, HTTPResource, ExecResource, PythonResource}
	for _, rt := range resourceTypes {
		dir := filepath.Join(tempDir, string(rt))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	dr := &DependencyResolver{
		ActionDir: tempDir,
		RequestID: "test123",
		Context:   context.Background(),
	}

	// Test with invalid resource type
	_, err = dr.GetCurrentTimestamp("test1", ResourceType("invalid"))
	if err == nil {
		t.Error("expected error for invalid resource type, got none")
	}

	// Test with non-existent file
	_, err = dr.GetCurrentTimestamp("test1", LLMResource)
	if err == nil {
		t.Error("expected error for non-existent file, got none")
	}
}
