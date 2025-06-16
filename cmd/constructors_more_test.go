package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCommandConstructorsMetadata(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	tmpDir := t.TempDir()
	logger := logging.NewTestLogger()

	env, _ := environment.NewEnvironment(fs, nil)
	root := NewRootCommand(fs, ctx, tmpDir, &kdeps.Kdeps{}, env, logger)
	assert.Equal(t, "kdeps", root.Use)

	addCmd := NewAddCommand(fs, ctx, tmpDir, logger)
	assert.Contains(t, addCmd.Aliases, "i")
	assert.Equal(t, "install [package]", addCmd.Use)

	scaffold := NewScaffoldCommand(fs, ctx, logger)
	assert.Equal(t, "scaffold", scaffold.Name())
}
