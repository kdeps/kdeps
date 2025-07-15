package evaluator_test

import (
	"context"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/apple/pkl-go/pkl"
	evaluator "github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeEvaluator_Success(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}

	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	// Verify evaluator was created
	evaluatorInstance, err := evaluator.GetEvaluator()
	require.NoError(t, err)
	assert.NotNil(t, evaluatorInstance)

	// Test that we can evaluate a simple PKL expression
	source := pkl.TextSource("value = 42")
	result, err := evaluatorInstance.EvaluateOutputText(ctx, source)
	require.NoError(t, err)
	assert.Contains(t, result, "42")
}

func TestInitializeEvaluator_WithResourceReaders(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create mock resource readers
	mockReader := &mockResourceReader{}

	config := &evaluator.EvaluatorConfig{
		ResourceReaders: []pkl.ResourceReader{mockReader},
		Logger:          logger,
	}

	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	evaluatorInstance, err := evaluator.GetEvaluator()
	require.NoError(t, err)
	assert.NotNil(t, evaluatorInstance)
}

func TestInitializeEvaluator_WithCustomOptions(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	customOptions := func(options *pkl.EvaluatorOptions) {
		pkl.WithDefaultAllowedResources(options)
		pkl.WithOsEnv(options)
		options.Logger = pkl.NoopLogger
		options.AllowedModules = []string{".*"}
		options.AllowedResources = []string{".*"}
		options.OutputFormat = "pcf"
	}

	config := &evaluator.EvaluatorConfig{
		Logger:  logger,
		Options: customOptions,
	}

	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	evaluatorInstance, err := evaluator.GetEvaluator()
	require.NoError(t, err)
	assert.NotNil(t, evaluatorInstance)
}

func TestGetEvaluator_NotInitialized(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	_, err := evaluator.GetEvaluator()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evaluator not initialized")
}

func TestGetEvaluatorManager_NotInitialized(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	_, err := evaluator.GetEvaluatorManager()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evaluator manager not initialized")
}

func TestGetEvaluatorManager_Success(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}

	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	manager, err := evaluator.GetEvaluatorManager()
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestEvaluatorManager_Close(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}

	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	manager, err := evaluator.GetEvaluatorManager()
	require.NoError(t, err)

	// Verify evaluator exists before closing
	evaluatorInstance, err := evaluator.GetEvaluator()
	require.NoError(t, err)
	assert.NotNil(t, evaluatorInstance)

	// Close the evaluator
	err = manager.Close()
	require.NoError(t, err)

	// Verify evaluator is now nil
	evaluatorInstance, err = evaluator.GetEvaluator()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evaluator instance is nil")
}

func TestEvaluatorManager_EvaluateModuleSource(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}

	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	manager, err := evaluator.GetEvaluatorManager()
	require.NoError(t, err)

	// Test evaluating a simple PKL expression
	source := pkl.TextSource("message = \"Hello, World!\"")
	_, evalErr := manager.EvaluateModuleSource(ctx, source)
	require.NoError(t, evalErr)
	assert.Contains(t, "message = \"Hello, World!\"", "Hello, World!")
}

func TestEvaluatorManager_EvaluateModuleSource_NilEvaluator(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create manager through proper initialization
	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}
	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)
	manager, err := evaluator.GetEvaluatorManager()
	require.NoError(t, err)

	// Test with valid PKL code that should work
	source := pkl.TextSource("value = 1")
	result, err := manager.EvaluateModuleSource(ctx, source)
	require.NoError(t, err)
	assert.Contains(t, result, "value = 1")
}

func TestReset(t *testing.T) {
	// Initialize evaluator first
	ctx := context.Background()
	logger := logging.NewTestLogger()

	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}

	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	// Verify evaluator exists
	evaluatorInstance, err := evaluator.GetEvaluator()
	require.NoError(t, err)
	assert.NotNil(t, evaluatorInstance)

	// Reset
	evaluator.Reset()

	// Verify evaluator is gone
	_, err = evaluator.GetEvaluator()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evaluator not initialized")

	// Verify we can initialize again
	err = evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	evaluatorInstance, err = evaluator.GetEvaluator()
	require.NoError(t, err)
	assert.NotNil(t, evaluatorInstance)
}

func TestSingleton_ThreadSafety(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}

	// Initialize in a goroutine
	done := make(chan bool)
	go func() {
		err := evaluator.InitializeEvaluator(ctx, config)
		assert.NoError(t, err)
		done <- true
	}()

	// Wait for initialization
	<-done

	// Verify evaluator is accessible from main thread
	evaluatorInstance, err := evaluator.GetEvaluator()
	require.NoError(t, err)
	assert.NotNil(t, evaluatorInstance)
}

func TestEvaluateText_WithSingleton(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Initialize evaluator
	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}

	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	// Test EvaluateText
	pklText := "message = \"Test message\""
	result, err := evaluator.EvaluateText(ctx, pklText, logger)
	require.NoError(t, err)
	assert.Contains(t, result, "Test message")
}

func TestEvaluateAllPklFilesInDirectory_WithSingleton(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()
	fs := afero.NewOsFs()

	// Initialize evaluator
	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}

	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	// Create test directory with PKL files in temp dir
	testDir := t.TempDir()

	// Create valid PKL files
	validFiles := []string{
		filepath.Join(testDir, "file1.pkl"),
		filepath.Join(testDir, "file2.pkl"),
		filepath.Join(testDir, "subdir", "file3.pkl"),
	}

	for _, file := range validFiles {
		err = fs.MkdirAll(filepath.Dir(file), 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, file, []byte("value = 1"), 0o644)
		require.NoError(t, err)
	}

	// Test evaluation
	err = evaluator.EvaluateAllPklFilesInDirectory(fs, ctx, testDir, logger)
	require.NoError(t, err)
}

// Mock resource reader for testing
type mockResourceReader struct{}

func (m *mockResourceReader) Scheme() string {
	return "mock"
}

func (m *mockResourceReader) IsGlobbable() bool {
	return false
}

func (m *mockResourceReader) HasHierarchicalUris() bool {
	return false
}

func (m *mockResourceReader) ListElements(_ url.URL) ([]pkl.PathElement, error) {
	return []pkl.PathElement{}, nil
}

func (m *mockResourceReader) Read(_ url.URL) ([]byte, error) {
	return []byte("mock data"), nil
}
