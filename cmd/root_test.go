package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	kdepsgen "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCommand(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdepsgen.Kdeps{}
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	cmd := NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)
	require.NotNil(t, cmd)
	assert.Equal(t, "kdeps", cmd.Use)
	assert.Contains(t, cmd.Short, "Multi-model AI agent framework")

	// Check persistent flag
	flag := cmd.PersistentFlags().Lookup("latest")
	require.NotNil(t, flag)
	assert.Equal(t, "l", flag.Shorthand)

	// Check subcommands
	subs := cmd.Commands()
	var subNames []string
	for _, sub := range subs {
		subNames = append(subNames, sub.Use)
	}
	assert.Contains(t, subNames, "new [agentName]")
	assert.Contains(t, subNames, "scaffold [agentName] [fileNames...]")
	assert.Contains(t, subNames, "install [package]")
	assert.Contains(t, subNames, "package [agent-dir]")
	assert.Contains(t, subNames, "build [package]")
	assert.Contains(t, subNames, "run [package]")

	// Test that the flag affects the schema package variable
	schema.UseLatest = false
	cmd.SetArgs([]string{"--latest"})
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.True(t, schema.UseLatest)
}
