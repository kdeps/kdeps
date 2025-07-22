package resolver_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	resolverpkg "github.com/kdeps/kdeps/pkg/resolver"
	pklRes "github.com/kdeps/schema/gen/resource"
	"github.com/spf13/afero"
)

// testLogger is a simple implementation of the Logger interface for testing
type testLogger struct{}

func (l *testLogger) Info(_ string, _ ...interface{})  {}
func (l *testLogger) Error(_ string, _ ...interface{}) {}
func (l *testLogger) Debug(_ string, _ ...interface{}) {}
func (l *testLogger) Warn(_ string, _ ...interface{})  {}
func (l *testLogger) Fatal(_ string, _ ...interface{}) {}

// TestLoadResourceEntriesStub ensures that the LoadResourceEntries function works when
// given a stub with a simple list of pkl files to process.
func TestLoadResourceEntriesStub(t *testing.T) {
	fs := afero.NewMemMapFs()
	projectDir := "/project"
	resourcesDir := projectDir + "/resources"
	logger := logging.NewTestLogger()

	// Create resources directory
	if err := fs.MkdirAll(resourcesDir, 0o755); err != nil {
		t.Fatalf("failed to create resources directory: %v", err)
	}

	// Create some dummy pkl files
	dummyFiles := []string{"action1.pkl", "action2.pkl", "action3.pkl"}
	for _, fileName := range dummyFiles {
		if err := afero.WriteFile(fs, filepath.Join(resourcesDir, fileName), []byte("dummy"), 0o644); err != nil {
			t.Fatalf("failed to write dummy pkl: %v", err)
		}
	}

	dr := &resolverpkg.DependencyResolver{
		Fs:                   fs,
		Logger:               logger,
		ProjectDir:           projectDir,
		ActionDir:            "/action",
		RequestID:            "req1",
		RequestPklFile:       filepath.Join("/action", "api/req1__request.pkl"),
		ResourceDependencies: make(map[string][]string),
		Resources:            []resolverpkg.ResourceNodeEntry{},
		Context:              context.Background(),
	}

	// stub LoadResourceFn to avoid real evaluation; just return a Resource with ActionID = filename (no extension)
	dr.LoadResourceFn = func(_ context.Context, file string, _ resolverpkg.ResourceType) (interface{}, error) {
		baseName := filepath.Base(file)
		nameWithoutExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]
		return &pklRes.ResourceImpl{ActionID: nameWithoutExt}, nil
	}

	// Call LoadResourceEntries
	if err := dr.LoadResourceEntries(); err != nil {
		t.Fatalf("LoadResourceEntries returned error: %v", err)
	}

	// Verify that the resources were loaded correctly
	if len(dr.Resources) != len(dummyFiles) {
		t.Errorf("expected %d resources, got %d", len(dummyFiles), len(dr.Resources))
	}

	// Verify that each resource has the correct ActionID
	for i, resource := range dr.Resources {
		expectedActionID := dummyFiles[i][:len(dummyFiles[i])-4] // Remove .pkl extension
		if resource.ActionID != expectedActionID {
			t.Errorf("expected ActionID %s, got %s", expectedActionID, resource.ActionID)
		}
	}
}

func TestHandleFileImports_DelegatesToInjectedFns(t *testing.T) {
	// PrependDynamicImportsFn and AddPlaceholderImportsFn removed - deprecated functionality
	// HandleFileImports functionality removed - imports included in templates
	// This test is deprecated - functions no longer exist
	t.Skip("Import functionality removed - imports included in templates")
}

func TestHandleFileImports_PropagatesError(t *testing.T) {
	// PrependDynamicImportsFn and AddPlaceholderImportsFn removed - deprecated functionality
	// HandleFileImports functionality removed - imports included in templates
	// This test is deprecated - functions no longer exist
	t.Skip("Import functionality removed - imports included in templates")
}
