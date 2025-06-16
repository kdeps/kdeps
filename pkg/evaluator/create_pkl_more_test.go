package evaluator

import (
	"context"
	"path/filepath"
	"testing"

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
