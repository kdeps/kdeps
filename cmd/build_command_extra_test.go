package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewBuildCommand_MetadataAndErrorPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	cmd := NewBuildCommand(fs, ctx, "/tmp/kdeps", nil, logging.NewTestLogger())

	// Verify metadata
	assert.Equal(t, "build [package]", cmd.Use)
	assert.Contains(t, cmd.Short, "dockerized")

	// Execute with missing arg should error due to cobra Args check
	err := cmd.Execute()
	assert.Error(t, err)

	// Provide non-existent file – RunE should propagate ExtractPackage error.
	cmd.SetArgs([]string{"nonexistent.kdeps"})
	err = cmd.Execute()
	assert.Error(t, err)
}
