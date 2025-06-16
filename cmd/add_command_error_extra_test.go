package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestNewAddCommand_ErrorPath ensures RunE returns an error when ExtractPackage fails.
func TestNewAddCommand_ErrorPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	cmd := NewAddCommand(fs, ctx, "/tmp/kdeps", logging.NewTestLogger())
	cmd.SetArgs([]string{"nonexistent.kdeps"})

	err := cmd.Execute()
	assert.Error(t, err, "expected error when package file does not exist")
}
