package evaluator

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestCreateAndProcessPklFile verifies that CreateAndProcessPklFile creates the temporary
// file, invokes the supplied process function, and writes the final output file without
// returning an error. A no-op processFunc is provided so that the test remains hermetic.
func TestCreateAndProcessPklFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	finalFile := "/output.pkl"

	// Dummy process function that just returns fixed content
	processFunc := func(_ afero.Fs, _ context.Context, tmpFile string, _ string, _ *logging.Logger) (string, error) {
		// Ensure the temporary file actually exists
		if exists, err := afero.Exists(fs, tmpFile); err != nil || !exists {
			t.Fatalf("expected temporary file %s to exist", tmpFile)
		}
		return "processed-content", nil
	}

	sections := []string{"name = \"unit-test\""}

	// Execute the helper under test
	err := CreateAndProcessPklFile(fs, ctx, sections, finalFile, "Kdeps.pkl", logger, processFunc, false)
	assert.NoError(t, err)

	// Validate that the final file was written with the expected content
	content, err := afero.ReadFile(fs, finalFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "processed-content")
}
