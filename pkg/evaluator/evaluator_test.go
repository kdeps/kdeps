package evaluator

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestEnsurePklBinaryExists(t *testing.T) {
	tests := []struct {
		name         string
		setupMocks   func()
		expectExit   bool
		expectedCode int
	}{
		{
			name: "pkl binary found",
			setupMocks: func() {
				ExecLookPathFunc = func(file string) (string, error) {
					if file == "pkl" {
						return "/usr/bin/pkl", nil
					}
					return "", errors.New("not found")
				}
			},
			expectExit: false,
		},
		{
			name: "pkl.exe binary found",
			setupMocks: func() {
				ExecLookPathFunc = func(file string) (string, error) {
					if file == "pkl.exe" {
						return "C:\\pkl.exe", nil
					}
					return "", errors.New("not found")
				}
			},
			expectExit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset evaluator state for clean testing
			ResetEvaluatorForTest()
			defer ResetEvaluatorForTest()

			// Setup mocks
			tt.setupMocks()

			// Reset to original functions after test
			defer func() {
				ExecLookPathFunc = func(file string) (string, error) {
					return exec.LookPath(file)
				}
				OsExitFunc = func(code int) {
					os.Exit(code)
				}
			}()

			ctx := context.Background()
			logger := logging.NewTestSafeLogger()

			if tt.expectExit {
				// Expect panic from mocked os.Exit
				assert.Panics(t, func() {
					EnsurePklBinaryExists(ctx, logger)
				})
			} else {
				// Should not panic
				assert.NotPanics(t, func() {
					err := EnsurePklBinaryExists(ctx, logger)
					assert.NoError(t, err)
				})
			}
		})
	}
}

func TestEvalPkl_InvalidExtension(t *testing.T) {
	// Reset evaluator state and skip binary check for testing
	ResetEvaluatorForTest()
	defer ResetEvaluatorForTest()

	originalSkip := SkipBinaryCheck
	SkipBinaryCheck = true
	defer func() { SkipBinaryCheck = originalSkip }()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	_, err := EvalPkl(fs, ctx, "test.txt", "header", nil, logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have a .pkl extension")
}

func TestEvalPkl_PackageURI(t *testing.T) {
	// Reset evaluator state and skip binary check for testing
	ResetEvaluatorForTest()
	defer ResetEvaluatorForTest()

	originalSkip := SkipBinaryCheck
	SkipBinaryCheck = true
	defer func() { SkipBinaryCheck = originalSkip }()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	// This test will likely fail due to PKL evaluation, but it tests the package URI detection logic
	_, err := EvalPkl(fs, ctx, "package://example.com/test.pkl", "header", nil, logger)

	// We expect an error since we don't have a real PKL evaluator, but it should not be an extension error
	if err != nil {
		assert.NotContains(t, err.Error(), "must have a .pkl extension")
	}
}

func TestEvalPkl_FileReadError(t *testing.T) {
	// Reset evaluator state and skip binary check for testing
	ResetEvaluatorForTest()
	defer ResetEvaluatorForTest()

	originalSkip := SkipBinaryCheck
	SkipBinaryCheck = true
	defer func() { SkipBinaryCheck = originalSkip }()

	// Test with read-only filesystem to trigger file read error
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	// This should handle the file read error gracefully
	_, err := EvalPkl(fs, ctx, "nonexistent.pkl", "header", nil, logger)

	// We expect an error, but it should attempt to treat it as a URI
	assert.Error(t, err)
}

func TestCreateAndProcessPklFile_TempDirError(t *testing.T) {
	// Reset evaluator state for testing
	ResetEvaluatorForTest()
	defer ResetEvaluatorForTest()

	// Mock organized temp dir creation to fail
	originalCreateKdepsTempDirFunc := CreateKdepsTempDirFunc
	defer func() {
		CreateKdepsTempDirFunc = originalCreateKdepsTempDirFunc
	}()

	CreateKdepsTempDirFunc = func(fs afero.Fs, requestID string, suffix string) (string, error) {
		return "", errors.New("temp dir error")
	}

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error) {
		return "processed", nil
	}

	err := CreateAndProcessPklFile(fs, ctx, []string{"section1"}, "output.pkl", "TestTemplate", nil, logger, processFunc, false, "test-request")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temporary directory")
}

func TestCreateAndProcessPklFile_TempFileError(t *testing.T) {
	// Reset evaluator state for testing
	ResetEvaluatorForTest()
	defer ResetEvaluatorForTest()

	// Mock organized temp file creation to fail
	originalCreateKdepsTempFileFunc := CreateKdepsTempFileFunc
	originalCreateKdepsTempDirFunc := CreateKdepsTempDirFunc
	defer func() {
		CreateKdepsTempFileFunc = originalCreateKdepsTempFileFunc
		CreateKdepsTempDirFunc = originalCreateKdepsTempDirFunc
	}()

	CreateKdepsTempDirFunc = func(fs afero.Fs, requestID string, suffix string) (string, error) {
		return "/tmp/kdeps/test-request", nil
	}

	CreateKdepsTempFileFunc = func(fs afero.Fs, requestID string, pattern string) (afero.File, error) {
		return nil, errors.New("temp file error")
	}

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error) {
		return "processed", nil
	}

	err := CreateAndProcessPklFile(fs, ctx, []string{"section1"}, "output.pkl", "TestTemplate", nil, logger, processFunc, false, "test-request")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temporary file")
}

func TestCreateAndProcessPklFile_ProcessError(t *testing.T) {
	// Reset evaluator state for testing
	ResetEvaluatorForTest()
	defer ResetEvaluatorForTest()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	// Process function that returns an error
	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error) {
		return "", errors.New("process error")
	}

	err := CreateAndProcessPklFile(fs, ctx, []string{"section1"}, "output.pkl", "TestTemplate", nil, logger, processFunc, false, "test-request")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to process temporary file")
}

func TestCreateAndProcessPklFile_WriteError(t *testing.T) {
	// Reset evaluator state for testing
	ResetEvaluatorForTest()
	defer ResetEvaluatorForTest()

	// Mock file writing to fail
	originalWriteFileFunc := AferoWriteFileFunc
	defer func() {
		AferoWriteFileFunc = originalWriteFileFunc
	}()

	AferoWriteFileFunc = func(fs afero.Fs, filename string, data []byte, perm os.FileMode) error {
		return errors.New("write error")
	}

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error) {
		return "processed", nil
	}

	err := CreateAndProcessPklFile(fs, ctx, []string{"section1"}, "output.pkl", "TestTemplate", nil, logger, processFunc, false, "test-request")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write final file")
}

func TestCreateAndProcessPklFile_Success(t *testing.T) {
	// Reset evaluator state for testing
	ResetEvaluatorForTest()
	defer ResetEvaluatorForTest()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error) {
		return "processed content", nil
	}

	err := CreateAndProcessPklFile(fs, ctx, []string{"section1", "section2"}, "output.pkl", "TestTemplate", nil, logger, processFunc, false, "test-request")

	assert.NoError(t, err)

	// Test with isExtension=true
	err = CreateAndProcessPklFile(fs, ctx, []string{"section1"}, "output2.pkl", "TestTemplate", nil, logger, processFunc, true, "test-request")

	assert.NoError(t, err)
}
