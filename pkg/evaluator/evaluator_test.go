package evaluator

import (
	"context"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCreateAndProcessPklFile_AmendsInPkg(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		// Simply return the header section to verify it flows through
		return headerSection + "\nprocessed", nil
	}

	final := "output_amends.pkl"
	sections := []string{"section1", "section2"}

	err := CreateAndProcessPklFile(fs, context.Background(), sections, final, "template.pkl", logger, processFunc, false)
	assert.NoError(t, err)

	// Verify final file exists and contains expected text
	content, err := afero.ReadFile(fs, final)
	assert.NoError(t, err)
	data := string(content)
	assert.True(t, strings.Contains(data, "amends \"package://schema.kdeps.com/core@"), "should contain amends relationship")
	assert.True(t, strings.Contains(data, "processed"))
}

func TestCreateAndProcessPklFile_ExtendsInPkg(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		return "result-" + headerSection, nil
	}

	final := "output_extends.pkl"
	err := CreateAndProcessPklFile(fs, context.Background(), nil, final, "template.pkl", logger, processFunc, true)
	assert.NoError(t, err)

	content, _ := afero.ReadFile(fs, final)
	str := string(content)
	assert.Contains(t, str, "extends \"package://schema.kdeps.com/core@")
	assert.Contains(t, str, "result-extends")
}

func TestCreateAndProcessPklFile_ProcessErrorInPkg(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		return "", assert.AnError
	}

	err := CreateAndProcessPklFile(fs, context.Background(), nil, "file.pkl", "template.pkl", logger, processFunc, false)
	assert.Error(t, err)
}

func TestEnsurePklBinaryExists(t *testing.T) {
	// Since mocking exec.LookPath directly is not possible, we can't easily test the binary lookup
	// Instead, we'll note that this test is limited and may need environment setup or alternative mocking
	// For now, we'll run the function as is, acknowledging it depends on the actual PATH
	ctx := context.Background()
	logger := logging.GetLogger()
	// This test will pass if 'pkl' is in PATH, fail with Fatal if not
	// We can't control the environment fully in this context
	err := EnsurePklBinaryExists(ctx, logger)
	if err != nil {
		t.Errorf("Expected no error if binary is in PATH, got: %v", err)
	}
	t.Log("EnsurePklBinaryExists test passed (dependent on PATH)")
}
