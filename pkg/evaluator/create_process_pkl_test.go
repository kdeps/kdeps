package evaluator

import (
	"context"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

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
