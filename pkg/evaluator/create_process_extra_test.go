package evaluator

import (
	"context"
	"errors"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

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
