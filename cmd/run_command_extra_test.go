package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewRunCommand_MetadataAndErrorPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	cmd := NewRunCommand(fs, ctx, "/tmp/kdeps", nil, logging.NewTestLogger())

	// metadata assertions
	assert.Equal(t, "run [package]", cmd.Use)
	assert.Contains(t, cmd.Short, "dockerized")

	// missing arg should error
	err := cmd.Execute()
	assert.Error(t, err)

	// non-existent file should propagate error
	cmd.SetArgs([]string{"nonexistent.kdeps"})
	err = cmd.Execute()
	assert.Error(t, err)
}
