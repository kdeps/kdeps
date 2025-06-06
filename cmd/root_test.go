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

func TestNewRootCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	cmd := NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "kdeps", cmd.Use)
}

func TestNewAgentCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAgentCommand(fs, ctx, kdepsDir, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "new [agentName]", cmd.Use)
}

func TestNewScaffoldCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(fs, ctx, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "scaffold [agentName] [fileNames...]", cmd.Use)
}

func TestNewAddCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "install [package]", cmd.Use)
}

func TestNewPackageCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	cmd := NewPackageCommand(fs, ctx, kdepsDir, env, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "package [agent-dir]", cmd.Use)
}

func TestNewBuildCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "build [package]", cmd.Use)
}

func TestNewRunCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "run [package]", cmd.Use)
}
