package cmd

import (
	"context"
	"strings"
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
	kdepsDir := "/test/kdeps"
	systemCfg := &kdeps.Kdeps{}
	env := &environment.Environment{}
	logger := logging.GetLogger()

	rootCmd := NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)

	// Test case 1: Check if root command is created
	if rootCmd == nil {
		t.Errorf("Expected non-nil root command, got nil")
	}
	if rootCmd.Use != "kdeps" {
		t.Errorf("Expected root command use to be 'kdeps', got '%s'", rootCmd.Use)
	}

	// Test case 2: Check if subcommands are added
	subcommands := rootCmd.Commands()
	expectedSubcommands := []string{"new", "scaffold", "install", "package", "build", "run", "upgrade"}
	if len(subcommands) != len(expectedSubcommands) {
		t.Errorf("Expected %d subcommands, got %d", len(expectedSubcommands), len(subcommands))
	}

	for i, expected := range expectedSubcommands {
		if i < len(subcommands) {
			actual := subcommands[i].Use
			// Extract base command name by taking the first part before any space or bracket
			if idx := strings.Index(actual, " "); idx != -1 {
				actual = actual[:idx]
			}
			if actual != expected {
				t.Errorf("Expected subcommand at index %d to be '%s', got '%s'", i, expected, actual)
			}
		}
	}

	// Test case 3: Check if persistent flag is set
	flag := rootCmd.PersistentFlags().Lookup("latest")
	if flag == nil {
		t.Errorf("Expected 'latest' persistent flag to be set, got nil")
	}

	t.Log("NewRootCommand test passed")
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

func TestNewRootCommandMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	env := &environment.Environment{}
	cmd := NewRootCommand(fs, context.Background(), "/kdeps", nil, env, logging.NewTestLogger())
	if cmd.Use != "kdeps" {
		t.Fatalf("expected root command name kdeps, got %s", cmd.Use)
	}
	if cmd.Version == "" {
		t.Fatalf("version string should be set")
	}
	if len(cmd.Commands()) == 0 {
		t.Fatalf("expected subcommands attached")
	}
}
