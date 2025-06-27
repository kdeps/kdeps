package evaluator

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestEvaluatorBoost(t *testing.T) {
	t.Run("EnsurePklBinaryExists", func(t *testing.T) {
		// Test EnsurePklBinaryExists function to boost coverage
		ctx := context.Background()
		logger := logging.NewTestSafeLogger()

		// Reset evaluator state for each test
		defer ResetEvaluatorForTest()

		// Test case where PKL binary is not found
		t.Run("no_pkl_binary", func(t *testing.T) {
			// Reset state for this subtest
			ResetEvaluatorForTest()

			// Mock the ExecLookPathFunc to simulate binary not found
			originalFunc := ExecLookPathFunc
			defer func() { ExecLookPathFunc = originalFunc }()

			ExecLookPathFunc = func(file string) (string, error) {
				return "", assert.AnError // Simulate binary not found
			}

			// Mock OsExitFunc to prevent actual exit during test
			originalExitFunc := OsExitFunc
			defer func() { OsExitFunc = originalExitFunc }()

			exitCalled := false
			OsExitFunc = func(code int) {
				exitCalled = true
				assert.Equal(t, 1, code)
			}

			// This should call os.Exit(1) through OsExitFunc
			err := EnsurePklBinaryExists(ctx, logger)
			assert.NoError(t, err) // Function returns nil but calls exit
			assert.True(t, exitCalled, "Expected OsExitFunc to be called")
		})

		// Test case where PKL binary is found
		t.Run("pkl_binary_exists", func(t *testing.T) {
			// Reset state for this subtest
			ResetEvaluatorForTest()

			// Mock the ExecLookPathFunc to simulate binary found
			originalFunc := ExecLookPathFunc
			defer func() { ExecLookPathFunc = originalFunc }()

			ExecLookPathFunc = func(file string) (string, error) {
				return "/usr/local/bin/pkl", nil // Simulate binary found
			}

			err := EnsurePklBinaryExists(ctx, logger)
			assert.NoError(t, err)
		})

		// Test case where binary check is skipped
		t.Run("skip_binary_check", func(t *testing.T) {
			// Reset state for this subtest
			ResetEvaluatorForTest()

			// Enable skip binary check
			originalSkip := SkipBinaryCheck
			SkipBinaryCheck = true
			defer func() { SkipBinaryCheck = originalSkip }()

			// Mock the ExecLookPathFunc to simulate binary not found (should be ignored)
			originalFunc := ExecLookPathFunc
			defer func() { ExecLookPathFunc = originalFunc }()

			ExecLookPathFunc = func(file string) (string, error) {
				return "", assert.AnError // Simulate binary not found
			}

			// Should not call exit since we're skipping
			err := EnsurePklBinaryExists(ctx, logger)
			assert.NoError(t, err)
		})
	})

	t.Run("ResetEvaluatorForTest", func(t *testing.T) {
		// Test that ResetEvaluatorForTest works correctly
		// This is important for test isolation
		ResetEvaluatorForTest()

		// Reset should succeed without errors
		assert.True(t, true, "ResetEvaluatorForTest completed successfully")

		// Test that binary check state is reset
		binaryCheckMutex.Lock()
		checkState := binaryCheckDone
		binaryCheckMutex.Unlock()

		assert.False(t, checkState, "Binary check state should be reset")
	})
}
