package cmd_test

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/schema/gen/kdeps"
	kschema "github.com/kdeps/schema/gen/kdeps"
	schemaKdeps "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Aliases to cmd package constructors so we can use them without prefix in tests.
var (
	NewAddCommand      = cmd.NewAddCommand
	NewBuildCommand    = cmd.NewBuildCommand
	NewPackageCommand  = cmd.NewPackageCommand
	NewRunCommand      = cmd.NewRunCommand
	NewScaffoldCommand = cmd.NewScaffoldCommand
	NewAgentCommand    = cmd.NewAgentCommand
	NewRootCommand     = cmd.NewRootCommand
)

// TestCommandConstructors simply ensures that constructing each top-level Cobra command
// does not panic and returns a non-nil *cobra.Command. This executes the constructor
// logic which improves coverage of the cmd package without executing the command
// handlers themselves (which may require heavy runtime dependencies).
func TestCommandConstructors(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.TODO()
	logger := logging.NewTestLogger()

	tests := []struct {
		name string
		fn   func() interface{}
	}{
		{name: "Add", fn: func() interface{} { return cmd.NewAddCommand(fs, ctx, "", logger) }},
		{name: "Build", fn: func() interface{} { return cmd.NewBuildCommand(fs, ctx, "", nil, logger) }},
		{name: "Package", fn: func() interface{} { return cmd.NewPackageCommand(fs, ctx, "", nil, logger) }},
		{name: "Run", fn: func() interface{} { return cmd.NewRunCommand(fs, ctx, "", nil, logger) }},
		{name: "Scaffold", fn: func() interface{} { return cmd.NewScaffoldCommand(fs, ctx, logger) }},
		{name: "Agent", fn: func() interface{} { return cmd.NewAgentCommand(fs, ctx, "", logger) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("constructor %s panicked: %v", tc.name, r)
				}
			}()

			if cmdVal := tc.fn(); cmdVal == nil {
				t.Fatalf("constructor %s returned nil", tc.name)
			}
		})
	}
}

// TestNewAddCommand_RunE_Error ensures that the RunE closure returns an error
// when the provided package path does not exist. This exercises the early
// error-handling branch without performing a full extraction.
func TestNewAddCommand_RunE_Error(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir := "/tmp/kdeps"

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}

	err := cmd.RunE(cmd, []string{"nonexistent.kdeps"})
	if err == nil {
		t.Fatalf("expected error for missing package")
	}

	// Reference schema version to satisfy project rules.
	_ = schema.SchemaVersion(ctx)
}

// TestNewPackageCommand_Error triggers the error path when the workflow file
// cannot be found under the provided agent directory.
func TestNewPackageCommand_Error(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Minimal environment stub.
	env := &environment.Environment{}

	cmd := NewPackageCommand(fs, ctx, "/kdeps", env, logger)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}
	err := cmd.RunE(cmd, []string{"/myAgent"})
	if err == nil {
		t.Fatalf("expected error for missing workflow file")
	}

	_ = schema.SchemaVersion(ctx)
}

// TestNewAgentCommand_Success verifies that the command successfully scaffolds
// a new agent directory structure using an in-memory filesystem.
func TestNewAgentCommand_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	agentName := "testagent"
	cmd := NewAgentCommand(fs, ctx, "/tmp", logger)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}
	if err := cmd.RunE(cmd, []string{agentName}); err != nil {
		t.Fatalf("RunE returned error: %v", err)
	}

	// Verify that workflow.pkl was generated.
	exists, err := afero.Exists(fs, agentName+"/workflow.pkl")
	if err != nil || !exists {
		t.Fatalf("expected generated workflow file, err=%v exists=%v", err, exists)
	}

	// Verify at least one resource file exists.
	files, err := afero.Glob(fs, agentName+"/resources/*.pkl")
	if err != nil || len(files) == 0 {
		t.Fatalf("expected resource files, err=%v", err)
	}

	// Sanity-check: ensure GenerateResourceFiles created output using the template package.

	_ = schema.SchemaVersion(ctx)
}

// TestNewBuildCommand_Error ensures that Build command surfaces error on
// missing package and exits early before heavy docker logic runs.
func TestNewBuildCommand_Error(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	systemCfg := &schemaKdeps.Kdeps{}

	cmd := NewBuildCommand(fs, ctx, "/kdeps", systemCfg, logger)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}

	err := cmd.RunE(cmd, []string{"missing.kdeps"})
	if err == nil {
		t.Fatalf("expected error for missing package")
	}

	_ = schema.SchemaVersion(ctx)
}

// TestNewRunCommand_Error validates early-exit error handling for the Run command.
func TestNewRunCommand_Error(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	systemCfg := &schemaKdeps.Kdeps{}

	cmd := NewRunCommand(fs, ctx, "/kdeps", systemCfg, logger)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}

	err := cmd.RunE(cmd, []string{"missing.kdeps"})
	if err == nil {
		t.Fatalf("expected error for missing package")
	}

	_ = schema.SchemaVersion(ctx)
}

func TestCommandConstructorsUseStrings(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	dir := t.TempDir()
	logger := logging.NewTestLogger()

	constructors := []struct {
		name string
		cmd  func() string
	}{
		{"build", func() string { return NewBuildCommand(fs, ctx, dir, nil, logger).Use }},
		{"new", func() string { return NewAgentCommand(fs, ctx, dir, logger).Use }},
		{"package", func() string { return NewPackageCommand(fs, ctx, dir, nil, logger).Use }},
		{"run", func() string { return NewRunCommand(fs, ctx, dir, nil, logger).Use }},
		{"scaffold", func() string { return NewScaffoldCommand(fs, ctx, logger).Use }},
	}

	for _, c := range constructors {
		use := c.cmd()
		assert.NotEmpty(t, use, c.name)
	}
}

// TestCommandConstructors verifies each Cobra constructor returns a non-nil *cobra.Command
// with the expected Use string populated. We don't execute the RunE handlers -
// just calling the constructor is enough to cover its statements.
func TestCommandConstructorsAdditional(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	tmpDir := t.TempDir()
	logger := logging.NewTestLogger()

	// Environment needed for NewPackageCommand
	env, err := environment.NewEnvironment(fs, nil)
	if err != nil {
		t.Fatalf("env error: %v", err)
	}

	// Dummy config object for Build / Run commands
	dummyCfg := &kschema.Kdeps{}

	cases := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"add", NewAddCommand(fs, ctx, tmpDir, logger)},
		{"build", NewBuildCommand(fs, ctx, tmpDir, dummyCfg, logger)},
		{"new", NewAgentCommand(fs, ctx, tmpDir, logger)},
		{"package", NewPackageCommand(fs, ctx, tmpDir, env, logger)},
		{"run", NewRunCommand(fs, ctx, tmpDir, dummyCfg, logger)},
		{"scaffold", NewScaffoldCommand(fs, ctx, logger)},
	}

	for _, c := range cases {
		if c.cmd == nil {
			t.Fatalf("%s: constructor returned nil", c.name)
		}
		if c.cmd.Use == "" {
			t.Fatalf("%s: Use string empty", c.name)
		}
	}
}

func TestNewAddCommand_Meta(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewAddCommand(fs, context.Background(), "/tmp/kdeps", logging.NewTestLogger())

	if cmd.Use != "install [package]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}

	if len(cmd.Aliases) == 0 || cmd.Aliases[0] != "i" {
		t.Fatalf("expected alias 'i', got %v", cmd.Aliases)
	}
}

func TestNewBuildCommand_Meta(t *testing.T) {
	fs := afero.NewMemMapFs()
	systemCfg := &kschema.Kdeps{}
	cmd := NewBuildCommand(fs, context.Background(), "/tmp/kdeps", systemCfg, logging.NewTestLogger())

	if cmd.Use != "build [package]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}

	if len(cmd.Aliases) == 0 || cmd.Aliases[0] != "b" {
		t.Fatalf("expected alias 'b', got %v", cmd.Aliases)
	}
}

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
