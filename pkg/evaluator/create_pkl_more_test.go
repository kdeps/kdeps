package evaluator

import (
	"context"
	"path/filepath"
	"testing"

	"errors"
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCreateAndProcessPklFile_Minimal(t *testing.T) {
	memFs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	tmpDir := t.TempDir()
	finalFile := filepath.Join(tmpDir, "out.pkl")

	// Stub processFunc: just returns the header section.
	stub := func(fs afero.Fs, ctx context.Context, tmpFile string, header string, logger *logging.Logger) (string, error) {
		return header + "\ncontent", nil
	}

	err := CreateAndProcessPklFile(memFs, context.Background(), nil, finalFile, "Dummy.pkl", logger, stub, false)
	assert.NoError(t, err)

	// Verify file written with expected content.
	data, readErr := afero.ReadFile(memFs, finalFile)
	assert.NoError(t, readErr)
	assert.Contains(t, string(data), "content")
}

// stubProcessSuccess returns dummy content without error.
func stubProcessSuccess(fs afero.Fs, ctx context.Context, tmpFile string, header string, logger *logging.Logger) (string, error) {
	return header + "\ncontent", nil
}

// stubProcessFail returns an error to simulate processing failure.
func stubProcessFail(fs afero.Fs, ctx context.Context, tmpFile string, header string, logger *logging.Logger) (string, error) {
	return "", errors.New("process failed")
}

func TestCreateAndProcessPklFile_ProcessFuncError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	err := CreateAndProcessPklFile(fs, ctx, []string{"x = 1"}, "/ignored.pkl", "template.pkl", logger, stubProcessFail, false)
	if err == nil {
		t.Fatalf("expected error from processFunc, got nil")
	}
}

func TestCreateAndProcessPklFile_WritesFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	finalPath := "/out/final.pkl"

	if err := CreateAndProcessPklFile(fs, ctx, []string{"x = 1"}, finalPath, "template.pkl", logger, stubProcessSuccess, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert file now exists
	if ok, _ := afero.Exists(fs, finalPath); !ok {
		t.Fatalf("expected output file to be created")
	}
}

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

func TestCreateAndProcessPklFileNew(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := &logging.Logger{}
	sections := []string{
		`key = "value"`,
	}
	finalFileName := "test_output.pkl"
	pklTemplate := "template.pkl"
	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		// Simulate processing by reading the temp file
		content, err := afero.ReadFile(fs, tmpFile)
		if err != nil {
			return "", err
		}
		return string(content) + "\nprocessed", nil
	}

	err := CreateAndProcessPklFile(fs, ctx, sections, finalFileName, pklTemplate, logger, processFunc, false)
	if err != nil {
		t.Errorf("CreateAndProcessPklFile failed: %v", err)
	}

	// Check if the final file was created and has content
	content, err := afero.ReadFile(fs, finalFileName)
	if err != nil {
		t.Errorf("Final file was not created or readable: %v", err)
	} else if len(content) == 0 {
		t.Errorf("Final file is empty")
	} else if !strings.Contains(string(content), "processed") {
		t.Errorf("Final file does not contain processed content: %s", string(content))
	}

	// Clean up
	fs.Remove(finalFileName)
}

func TestCreateAndProcessPklFileWithExtensionNew(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := &logging.Logger{}
	sections := []string{
		`key = "value"`,
	}
	finalFileName := "test_output_ext.pkl"
	pklTemplate := "template.pkl"
	processFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		// Simulate processing by reading the temp file
		content, err := afero.ReadFile(fs, tmpFile)
		if err != nil {
			return "", err
		}
		return string(content) + "\nprocessed with extension", nil
	}

	err := CreateAndProcessPklFile(fs, ctx, sections, finalFileName, pklTemplate, logger, processFunc, true)
	if err != nil {
		t.Errorf("CreateAndProcessPklFile with extension failed: %v", err)
	}

	// Check if the final file was created and has content
	content, err := afero.ReadFile(fs, finalFileName)
	if err != nil {
		t.Errorf("Final file was not created or readable: %v", err)
	} else if len(content) == 0 {
		t.Errorf("Final file is empty")
	} else if !strings.Contains(string(content), "processed with extension") {
		t.Errorf("Final file does not contain processed content: %s", string(content))
	}

	// Clean up
	fs.Remove(finalFileName)
}
