package evaluator_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/apple/pkl-go/pkl"
	evaluator "github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndProcessPklFile_AmendsInPkg(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	processFunc := func(_ afero.Fs, _ context.Context, _ string, headerSection string, _ *logging.Logger) (string, error) {
		// Simply return the header section to verify it flows through
		return headerSection + "\nprocessed", nil
	}

	final := "output_amends.pkl"
	sections := []string{"section1", "section2"}

	err := evaluator.CreateAndProcessPklFile(fs, context.Background(), sections, final, "template.pkl", logger, processFunc, false)
	require.NoError(t, err)

	// Verify final file exists and contains expected text
	content, err := afero.ReadFile(fs, final)
	require.NoError(t, err)
	data := string(content)
	expectedAmends := fmt.Sprintf("amends \"%s\"", schema.ImportPath(context.Background(), "template.pkl"))
	require.Contains(t, data, expectedAmends, "should contain amends relationship")
	require.Contains(t, data, "processed")
}

func TestCreateAndProcessPklFile_ExtendsInPkg(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	processFunc := func(_ afero.Fs, _ context.Context, _ string, headerSection string, _ *logging.Logger) (string, error) {
		return "result-" + headerSection, nil
	}

	final := "output_extends.pkl"
	err := evaluator.CreateAndProcessPklFile(fs, context.Background(), nil, final, "template.pkl", logger, processFunc, true)
	require.NoError(t, err)

	content, _ := afero.ReadFile(fs, final)
	str := string(content)
	require.Contains(t, str, "extends \"package://schema.kdeps.com/core@")
	require.Contains(t, str, "result-extends")
}

func TestCreateAndProcessPklFile_ProcessErrorInPkg(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	processFunc := func(_ afero.Fs, _ context.Context, _ string, _ string, _ *logging.Logger) (string, error) {
		return "", assert.AnError
	}

	err := evaluator.CreateAndProcessPklFile(fs, context.Background(), nil, "file.pkl", "template.pkl", logger, processFunc, false)
	require.Error(t, err)
}

func TestEvalPkl_InvalidExtension(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	// Should error when file does not have .pkl extension
	_, err := evaluator.EvalPkl(fs, context.Background(), "file.txt", "header", nil, logger)
	require.Error(t, err)
	require.Contains(t, err.Error(), ".pkl extension")
}

func TestCreateAndProcessPklFile_Basic(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()
	fs := afero.NewMemMapFs()

	sections := []string{"section1", "section2"}
	finalFile := filepath.Join(t.TempDir(), "out.pkl")

	// simple process func echoes header + sections concatenated
	process := func(_ afero.Fs, _ context.Context, tmpFile string, _ string, _ *logging.Logger) (string, error) {
		data, err := afero.ReadFile(fs, tmpFile)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	err := evaluator.CreateAndProcessPklFile(fs, ctx, sections, finalFile, "Workflow.pkl", logger, process, false)
	require.NoError(t, err)

	content, err := afero.ReadFile(fs, finalFile)
	require.NoError(t, err)
	// ensure both sections exist
	require.Contains(t, string(content), "section1")
	require.Contains(t, string(content), "section2")
}

func TestCreateAndProcessPklFile_Simple(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	finalPath := "/out/result.pkl"
	sections := []string{"sec1", "sec2"}
	// processFunc writes content combining headerSection and sections
	var receivedHeader string
	processFunc := func(_ afero.Fs, _ context.Context, _ string, headerSection string, _ *logging.Logger) (string, error) {
		receivedHeader = headerSection
		return headerSection + "-processed", nil
	}

	err := evaluator.CreateAndProcessPklFile(fs, ctx, sections, finalPath, "Template.pkl", logger, processFunc, false)
	require.NoError(t, err)
	// Verify output file exists with expected content
	data, err := afero.ReadFile(fs, finalPath)
	require.NoError(t, err)
	require.Equal(t, receivedHeader+"-processed", string(data))
}

func TestCreateAndProcessPklFile_Extends(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	finalPath := "result_ext.pkl"
	sections := []string{"sec1", "sec2"}
	processFunc := func(_ afero.Fs, _ context.Context, _ string, headerSection string, _ *logging.Logger) (string, error) {
		return headerSection + "-processed", nil
	}

	err := evaluator.CreateAndProcessPklFile(fs, ctx, sections, finalPath, "Template.pkl", logger, processFunc, true)
	require.NoError(t, err)

	content, err := afero.ReadFile(fs, finalPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "extends")
}

func TestEvalPkl_InvalidExtensionAlt(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	_, err := evaluator.EvalPkl(fs, context.Background(), "file.txt", "header", nil, logger)
	require.Error(t, err)
	require.Contains(t, err.Error(), ".pkl extension")
}

func TestCreateAndProcessPklFile_Minimal(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	finalPath := "minimal.pkl"
	sections := []string{"test"}
	processFunc := func(_ afero.Fs, _ context.Context, _ string, _ string, _ *logging.Logger) (string, error) {
		return "processed", nil
	}

	err := evaluator.CreateAndProcessPklFile(fs, ctx, sections, finalPath, "Template.pkl", logger, processFunc, false)
	require.NoError(t, err)

	content, err := afero.ReadFile(fs, finalPath)
	require.NoError(t, err)
	require.Equal(t, "processed", string(content))
}

func stubProcessSuccess(_ afero.Fs, _ context.Context, _ string, _ string, _ *logging.Logger) (string, error) {
	return "success", nil
}

func stubProcessFail(_ afero.Fs, _ context.Context, _ string, _ string, _ *logging.Logger) (string, error) {
	return "", errors.New("process failed")
}

func TestCreateAndProcessPklFile_ProcessFuncError(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	finalPath := "error.pkl"

	err := evaluator.CreateAndProcessPklFile(fs, ctx, nil, finalPath, "Template.pkl", logger, stubProcessFail, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "process failed")
}

func TestCreateAndProcessPklFile_WritesFile(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	finalPath := "written.pkl"

	err := evaluator.CreateAndProcessPklFile(fs, ctx, nil, finalPath, "Template.pkl", logger, stubProcessSuccess, false)
	require.NoError(t, err)

	content, err := afero.ReadFile(fs, finalPath)
	require.NoError(t, err)
	require.Equal(t, "success", string(content))
}

func TestCreateAndProcessPklFile(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	finalPath := "test.pkl"
	sections := []string{"section1", "section2"}

	processFunc := func(f afero.Fs, _ context.Context, tmpFile string, _ string, _ *logging.Logger) (string, error) {
		data, err := afero.ReadFile(f, tmpFile)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	err := evaluator.CreateAndProcessPklFile(fs, ctx, sections, finalPath, "Template.pkl", logger, processFunc, false)
	require.NoError(t, err)

	content, err := afero.ReadFile(fs, finalPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "section1")
	require.Contains(t, string(content), "section2")
}

func TestCreateAndProcessPklFileNew(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	finalPath := "new.pkl"
	sections := []string{"new section"}

	processFunc := func(_ afero.Fs, _ context.Context, _ string, _ string, _ *logging.Logger) (string, error) {
		return "new processed", nil
	}

	err := evaluator.CreateAndProcessPklFile(fs, ctx, sections, finalPath, "Template.pkl", logger, processFunc, false)
	require.NoError(t, err)

	content, err := afero.ReadFile(fs, finalPath)
	require.NoError(t, err)
	require.Equal(t, "new processed", string(content))
}

func TestCreateAndProcessPklFileWithExtensionNew(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	finalPath := "extension.pkl"
	sections := []string{"ext section"}

	processFunc := func(_ afero.Fs, _ context.Context, _ string, _ string, _ *logging.Logger) (string, error) {
		return "extension processed", nil
	}

	err := evaluator.CreateAndProcessPklFile(fs, ctx, sections, finalPath, "Template.pkl", logger, processFunc, true)
	require.NoError(t, err)

	content, err := afero.ReadFile(fs, finalPath)
	require.NoError(t, err)
	require.Equal(t, "extension processed", string(content))
}

func TestEvalPkl(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	// Initialize evaluator for this test
	ctx := context.Background()
	logger := logging.NewTestLogger()
	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}
	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	// Test using TextSource instead of file-based evaluation
	evaluator, err := evaluator.GetEvaluator()
	require.NoError(t, err)

	source := pkl.TextSource("value = 42")
	result, err := evaluator.EvaluateOutputText(ctx, source)
	require.NoError(t, err)
	require.Contains(t, result, "42")
}

func TestEvalPkl_WithSingletonEvaluator(t *testing.T) {
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

	// Create a test PKL file in a real temp dir
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.pkl")
	testContent := "value = 123"
	err = afero.WriteFile(fs, testFile, []byte(testContent), 0o644)
	require.NoError(t, err)

	// Test EvalPkl with singleton
	headerSection := "amends \"package://test\""
	result, err := evaluator.EvalPkl(fs, ctx, testFile, headerSection, nil, logger)
	require.NoError(t, err)

	// Verify result contains both header and evaluated content
	assert.Contains(t, result, headerSection)
	assert.Contains(t, result, "123")
}

func TestEvaluateAllPklFilesInDirectory(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	// Initialize evaluator for this test
	ctx := context.Background()
	logger := logging.NewTestLogger()
	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}
	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	fs := afero.NewOsFs()
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

	err = evaluator.EvaluateAllPklFilesInDirectory(fs, ctx, testDir, logger)
	require.NoError(t, err)
}

func TestEvaluateAllPklFilesInDirectory_InvalidPkl(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	// Initialize evaluator for this test
	ctx := context.Background()
	logger := logging.NewTestLogger()
	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}
	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	fs := afero.NewMemMapFs()
	testDir := "testdir"
	err = fs.MkdirAll(testDir, 0o755)
	require.NoError(t, err)

	// Create invalid PKL file
	invalidFile := "testdir/invalid.pkl"
	err = afero.WriteFile(fs, invalidFile, []byte("invalid pkl content"), 0o644)
	require.NoError(t, err)

	err = evaluator.EvaluateAllPklFilesInDirectory(fs, ctx, testDir, logger)
	require.Error(t, err)
}

func TestEvaluateAllPklFilesInDirectory_NoPklFiles(t *testing.T) {
	// Reset singleton before test
	evaluator.Reset()

	// Initialize evaluator for this test
	ctx := context.Background()
	logger := logging.NewTestLogger()
	config := &evaluator.EvaluatorConfig{
		Logger: logger,
	}
	err := evaluator.InitializeEvaluator(ctx, config)
	require.NoError(t, err)

	fs := afero.NewMemMapFs()
	testDir := "testdir"
	err = fs.MkdirAll(testDir, 0o755)
	require.NoError(t, err)

	// Create non-PKL files
	nonPklFiles := []string{
		"testdir/file1.txt",
		"testdir/file2.go",
	}

	for _, file := range nonPklFiles {
		err = afero.WriteFile(fs, file, []byte("content"), 0o644)
		require.NoError(t, err)
	}

	// Should succeed (no PKL files to evaluate)
	err = evaluator.EvaluateAllPklFilesInDirectory(fs, ctx, testDir, logger)
	require.NoError(t, err)
}
