package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestNewAddCommandExtra(t *testing.T) {
	cmd := NewAddCommand(afero.NewMemMapFs(), context.Background(), "kd", logging.NewTestLogger())
	require.Equal(t, "install [package]", cmd.Use)
	require.Equal(t, []string{"i"}, cmd.Aliases)
	require.Equal(t, "Install an AI agent locally", cmd.Short)
	require.Equal(t, "$ kdeps install ./myAgent.kdeps", cmd.Example)
	require.Error(t, cmd.Args(nil, []string{}))
	require.NoError(t, cmd.Args(nil, []string{"pkg"}))
}

func TestNewAgentCommandExtra(t *testing.T) {
	cmd := NewAgentCommand(afero.NewMemMapFs(), context.Background(), "kd", logging.NewTestLogger())
	require.Equal(t, "new [agentName]", cmd.Use)
	require.Equal(t, []string{"n"}, cmd.Aliases)
	require.Equal(t, "Create a new AI agent", cmd.Short)
	require.Error(t, cmd.Args(nil, []string{}))
	require.Error(t, cmd.Args(nil, []string{"a", "b"}))
	require.NoError(t, cmd.Args(nil, []string{"a"}))
}

func TestNewPackageCommandExtra(t *testing.T) {
	env := &environment.Environment{}
	cmd := NewPackageCommand(afero.NewMemMapFs(), context.Background(), "kd", env, logging.NewTestLogger())
	require.Equal(t, "package [agent-dir]", cmd.Use)
	require.Equal(t, []string{"p"}, cmd.Aliases)
	require.Equal(t, "Package an AI agent to .kdeps file", cmd.Short)
	require.Equal(t, "$ kdeps package ./myAgent/", cmd.Example)
	require.Error(t, cmd.Args(nil, []string{}))
	require.NoError(t, cmd.Args(nil, []string{"dir"}))
}

func TestNewBuildCommandExtra(t *testing.T) {
	cfg := &kdeps.Kdeps{}
	cmd := NewBuildCommand(afero.NewMemMapFs(), context.Background(), "kd", cfg, logging.NewTestLogger())
	require.Equal(t, "build [package]", cmd.Use)
	require.Equal(t, []string{"b"}, cmd.Aliases)
	require.Equal(t, "Build a dockerized AI agent", cmd.Short)
	require.Equal(t, "$ kdeps build ./myAgent.kdeps", cmd.Example)
	require.Error(t, cmd.Args(nil, []string{}))
	require.NoError(t, cmd.Args(nil, []string{"pkg"}))
}

func TestNewRunCommandExtra(t *testing.T) {
	cfg := &kdeps.Kdeps{}
	cmd := NewRunCommand(afero.NewMemMapFs(), context.Background(), "kd", cfg, logging.NewTestLogger())
	require.Equal(t, "run [package]", cmd.Use)
	require.Equal(t, []string{"r"}, cmd.Aliases)
	require.Equal(t, "Build and run a dockerized AI agent container", cmd.Short)
	require.Equal(t, "$ kdeps run ./myAgent.kdeps", cmd.Example)
	require.Error(t, cmd.Args(nil, []string{}))
	require.NoError(t, cmd.Args(nil, []string{"pkg"}))
}

func TestNewScaffoldCommandExtra(t *testing.T) {
	cmd := NewScaffoldCommand(afero.NewMemMapFs(), context.Background(), logging.NewTestLogger())
	require.Equal(t, "scaffold [agentName] [fileNames...]", cmd.Use)
	require.Empty(t, cmd.Aliases)
	require.Equal(t, "Scaffold specific files for an agent", cmd.Short)
	require.Error(t, cmd.Args(nil, []string{}))
	require.NoError(t, cmd.Args(nil, []string{"agent"}))
	require.NoError(t, cmd.Args(nil, []string{"agent", "file1"}))
}
