package evaluator

import (
	"context"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

// TestSetup initializes the singleton evaluator for testing
func TestSetup(t *testing.T) {
	// Reset singleton before test
	Reset()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	config := &EvaluatorConfig{
		Logger: logger,
	}

	err := InitializeEvaluator(ctx, config)
	if err != nil {
		// Skip test if PKL binary is not available or initialization fails
		if strings.Contains(err.Error(), "exit status 1") ||
			strings.Contains(err.Error(), "Could not determine current working directory") ||
			strings.Contains(err.Error(), "PKL evaluator not available") {
			t.Skipf("Skipping test - PKL binary not available or evaluator initialization failed: %v", err)
		}
		t.Fatalf("failed to initialize evaluator for test: %v", err)
	}
}

// TestTeardown cleans up the singleton evaluator after testing
func TestTeardown(t *testing.T) {
	if manager, err := GetEvaluatorManager(); err == nil {
		if err := manager.Close(); err != nil {
			t.Logf("warning: failed to close evaluator during test teardown: %v", err)
		}
	}
	Reset()
}
